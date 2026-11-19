// Command refcheck is a heuristic lint for the reference counting at the
// boundary between internal/cef and the CEF C API. It checks the two rules
// the manual unsafe audit before v0.5.0 found broken eight times:
//
//  1. Every //export go* callback receives each reference-counted struct
//     parameter - all but the first, which is self - with a reference it owns
//     (CEF's CToCpp::Unwrap adds one and never takes it back). The callback
//     must release the parameter or hand it on: to a call whose name starts
//     with "release", a wrapper composite literal (cefFrame{frame}), a Go
//     function marked "refcheck:adopts <param>", or a consuming C parameter.
//     Anything else needs a "refcheck:handoff <param> <reason>" comment in
//     the function.
//
//  2. A C function parameter of a reference-counted struct type is either
//     borrowed or consumed. CEF's exported functions consume all of them; the
//     first parameter of a bridge function is borrowed (it is the object whose
//     method runs); every other one must say which with REF_CONSUMED or
//     REF_BORROWED (empty macros from memory.h) after its name. An argument
//     for a consumed parameter must be a reference the function owns and
//     gives away: nil, the result of a C call (CEF returns every struct with
//     a new reference) or of a Go function marked "refcheck:fresh", a
//     callback parameter, or a value an addRef of the same expression
//     precedes - each of those good for one consuming call. Otherwise the
//     line or the one above needs a "refcheck:owned <reason>" comment, for a
//     wrapper method that gives away its caller's reference.
//
// Reference-counted means a struct that derives from cef_base_ref_counted_t
// through its first member, read from the vendored CEF headers, the bridge
// headers and the cgo preambles. The analysis is syntactic and per function:
// it does not follow branches, so a release on one path satisfies rule 1 for
// all of them. It catches the forgotten release and the missing addRef before
// a consuming call (the Equals bugs of the audit), not the release on the
// wrong path - that is what the refdebug ledger build is for.
//
//	go run ./tools/refcheck            # from the module root
//	go run ./tools/refcheck -dir internal/cef -include include
package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

func main() {
	dir := flag.String("dir", "internal/cef", "package directory to check")
	include := flag.String("include", "include", "vendored CEF include directory")
	flag.Parse()

	findings, err := run(*dir, *include)
	if err != nil {
		fmt.Fprintln(os.Stderr, "refcheck:", err)
		os.Exit(2)
	}
	for _, f := range findings {
		fmt.Println(f)
	}
	if len(findings) > 0 {
		fmt.Fprintf(os.Stderr, "refcheck: %d finding(s)\n", len(findings))
		os.Exit(1)
	}
}

type finding struct {
	pos token.Position
	msg string
}

func (f finding) String() string {
	return fmt.Sprintf("%s:%d: %s", filepath.ToSlash(f.pos.Filename), f.pos.Line, f.msg)
}

// paramMode is how a C function treats one reference-counted parameter.
type paramMode int

const (
	modeUnknown paramMode = iota
	modeBorrows
	modeConsumes
)

type cFunc struct {
	name   string
	params map[int]paramMode // refcounted pointer parameters only
	pos    token.Position
}

type checker struct {
	fset     *token.FileSet
	refcount map[string]bool   // cef_frame_t -> true
	bases    map[string]string // struct -> the struct of its base member
	funcs    map[string]*cFunc
	adopts   map[string]map[int]bool // Go function -> parameter indices it adopts
	fresh    map[string]bool         // Go functions returning an owned reference
	findings []finding
}

func run(dir, include string) ([]finding, error) {
	c := &checker{
		fset:     token.NewFileSet(),
		refcount: map[string]bool{},
		bases:    map[string]string{},
		funcs:    map[string]*cFunc{},
		adopts:   map[string]map[int]bool{},
		fresh:    map[string]bool{},
	}

	var headers []string
	err := filepath.WalkDir(include, func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(p, ".h") {
			headers = append(headers, p)
		}
		return err
	})
	if err != nil {
		return nil, err
	}
	bridge, _ := filepath.Glob(filepath.Join(dir, "*.h"))
	sort.Strings(headers)
	sort.Strings(bridge)

	goFiles, _ := filepath.Glob(filepath.Join(dir, "*.go"))
	sort.Strings(goFiles)
	var files []*ast.File
	var preambles []source
	for _, p := range goFiles {
		f, err := parser.ParseFile(c.fset, p, nil, parser.ParseComments)
		if err != nil {
			return nil, err
		}
		files = append(files, f)
		for _, imp := range f.Imports {
			if imp.Path.Value == `"C"` {
				if doc := importDoc(f, imp); doc != nil {
					for _, cm := range doc.List {
						preambles = append(preambles, source{path: p, line: c.fset.Position(cm.Pos()).Line, text: commentCode(cm.Text)})
					}
				}
			}
		}
	}

	for _, h := range append(headers, bridge...) {
		b, err := os.ReadFile(h)
		if err != nil {
			return nil, err
		}
		c.scanStructs(string(b))
	}
	for _, p := range preambles {
		c.scanStructs(p.text)
	}
	c.resolveRefcount()
	for _, h := range headers {
		b, _ := os.ReadFile(h)
		c.scanFuncs(source{path: h, line: 1, text: string(b)}, true)
	}
	for _, h := range bridge {
		b, _ := os.ReadFile(h)
		c.scanFuncs(source{path: h, line: 1, text: string(b)}, false)
	}
	for _, p := range preambles {
		c.scanFuncs(p, false)
	}

	names := make([]string, 0, len(c.funcs))
	for name := range c.funcs {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		fn := c.funcs[name]
		for i := 0; i < 64; i++ {
			if mode, ok := fn.params[i]; ok && mode == modeUnknown {
				c.findings = append(c.findings, finding{pos: fn.pos,
					msg: fmt.Sprintf("%s: parameter %d takes a reference-counted struct; mark it REF_CONSUMED or REF_BORROWED", name, i)})
			}
		}
	}

	for _, f := range files {
		c.collectAdopts(f)
	}
	for _, f := range files {
		c.checkFile(f)
	}

	sort.Slice(c.findings, func(i, j int) bool {
		a, b := c.findings[i].pos, c.findings[j].pos
		if a.Filename != b.Filename {
			return a.Filename < b.Filename
		}
		return a.Line < b.Line
	})
	return c.findings, nil
}

type source struct {
	path string
	line int
	text string
}

// importDoc returns the cgo preamble of import "C": the comment directly
// above it, either on the import spec or on its declaration.
func importDoc(f *ast.File, imp *ast.ImportSpec) *ast.CommentGroup {
	if imp.Doc != nil {
		return imp.Doc
	}
	for _, d := range f.Decls {
		if g, ok := d.(*ast.GenDecl); ok && g.Tok == token.IMPORT {
			for _, s := range g.Specs {
				if s == imp {
					return g.Doc
				}
			}
		}
	}
	return nil
}

// commentCode returns the C code of one preamble comment, with the comment
// markers blanked so line numbers stay put.
func commentCode(text string) string {
	if strings.HasPrefix(text, "//") {
		return "  " + text[2:]
	}
	return "  " + strings.TrimSuffix(text[2:], "*/")
}

// structRe matches a struct definition up to its first member, the base a
// CEF struct derives from: cef_base_ref_counted_t, or another struct such as
// cef_preference_manager_t for cef_request_context_t.
var structRe = regexp.MustCompile(`typedef\s+struct\s+_(cef_\w+_t)\s*\{(?:\s|/\*(?:[^*]|\*[^/])*\*/|//[^\n]*\n)*(?:struct\s+_?)?(cef_\w+_t)\s+base\s*;`)

// scanStructs records the base of every CEF struct in text; resolveRefcount
// then marks the ones that derive from cef_base_ref_counted_t.
func (c *checker) scanStructs(text string) {
	for _, m := range structRe.FindAllStringSubmatch(text, -1) {
		c.bases[m[1]] = m[2]
	}
}

func (c *checker) resolveRefcount() {
	for name := range c.bases {
		seen := map[string]bool{}
		for t := name; t != "" && !seen[t]; t = c.bases[t] {
			seen[t] = true
			if t == "cef_base_ref_counted_t" {
				c.refcount[name] = true
				break
			}
		}
	}
}

var (
	lineComment = regexp.MustCompile(`//[^\n]*`)
	// a declaration or definition head: return type, name, parameter list
	funcRe    = regexp.MustCompile(`(?m)^[ \t]*((?:[A-Za-z_][\w]*[ \t\*]+)+?)([A-Za-z_]\w*)\s*\(([^()]*)\)\s*[;{]`)
	paramRe   = regexp.MustCompile(`^(?:const\s+)?(?:struct\s+)?_?(cef_\w+_t)\s*\*\s*(?:const\s+)?\w*\s*(REF_CONSUMED|REF_BORROWED)?\s*$`)
	cKeywords = map[string]bool{"if": true, "while": true, "for": true, "switch": true, "return": true, "sizeof": true, "else": true}
)

// scanFuncs records the C functions of a header or preamble that take a
// reference-counted struct by pointer, with the mode of each such parameter.
func (c *checker) scanFuncs(src source, cefHeader bool) {
	text := lineComment.ReplaceAllStringFunc(src.text, func(s string) string { return strings.Repeat(" ", len(s)) })
	for _, m := range funcRe.FindAllStringSubmatchIndex(text, -1) {
		ret, name, params := text[m[2]:m[3]], text[m[4]:m[5]], text[m[6]:m[7]]
		if cKeywords[name] || strings.Contains(ret, "return") || strings.Contains(ret, "typedef") {
			continue
		}
		export := cefHeader && strings.Contains(ret, "CEF_EXPORT")
		if cefHeader && !export {
			continue // CEF headers: only the exported free functions are callable
		}
		fn := &cFunc{name: name, params: map[int]paramMode{},
			pos: token.Position{Filename: src.path, Line: src.line + strings.Count(text[:m[0]], "\n")}}
		for i, p := range splitParams(params) {
			pm := paramRe.FindStringSubmatch(strings.TrimSpace(p))
			if pm == nil || !c.refcount[pm[1]] {
				continue
			}
			switch {
			case export:
				fn.params[i] = modeConsumes
			case pm[2] == "REF_CONSUMED":
				fn.params[i] = modeConsumes
			case pm[2] == "REF_BORROWED" || i == 0:
				fn.params[i] = modeBorrows
			default:
				fn.params[i] = modeUnknown
			}
		}
		if len(fn.params) == 0 {
			continue
		}
		if prev, ok := c.funcs[name]; ok {
			// a prototype and its definition: an annotation on either counts
			for i, mode := range fn.params {
				if mode > prev.params[i] {
					prev.params[i] = mode
				}
			}
			continue
		}
		c.funcs[name] = fn
	}
}

func splitParams(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" || s == "void" {
		return nil
	}
	return strings.Split(s, ",")
}

var adoptsRe = regexp.MustCompile(`refcheck:adopts\s+(\w+)`)

// collectAdopts records the Go functions whose doc comment says they adopt
// the reference of a parameter (refcheck:adopts <param>) or return one the
// caller owns (refcheck:fresh).
func (c *checker) collectAdopts(f *ast.File) {
	for _, d := range f.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if !ok || fd.Doc == nil || fd.Recv != nil {
			continue
		}
		if strings.Contains(fd.Doc.Text(), "refcheck:fresh") {
			c.fresh[fd.Name.Name] = true
		}
		for _, m := range adoptsRe.FindAllStringSubmatch(fd.Doc.Text(), -1) {
			i := 0
			for _, field := range fd.Type.Params.List {
				for _, n := range field.Names {
					if n.Name == m[1] {
						if c.adopts[fd.Name.Name] == nil {
							c.adopts[fd.Name.Name] = map[int]bool{}
						}
						c.adopts[fd.Name.Name][i] = true
					}
					i++
				}
			}
		}
	}
}

func (c *checker) report(pos token.Pos, format string, args ...any) {
	c.findings = append(c.findings, finding{pos: c.fset.Position(pos), msg: fmt.Sprintf(format, args...)})
}

// cTypeName returns cef_frame_t for *C.cef_frame_t and *C.struct__cef_frame_t.
func cTypeName(e ast.Expr) (string, bool) {
	star, ok := e.(*ast.StarExpr)
	if !ok {
		return "", false
	}
	sel, ok := star.X.(*ast.SelectorExpr)
	if !ok {
		return "", false
	}
	if x, ok := sel.X.(*ast.Ident); !ok || x.Name != "C" {
		return "", false
	}
	return strings.TrimPrefix(sel.Sel.Name, "struct__"), true
}

func (c *checker) checkFile(f *ast.File) {
	for _, d := range f.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if !ok || fd.Body == nil {
			continue
		}
		comments := commentsIn(f, fd)
		c.checkCalls(fd, comments)
		if isExport(fd) {
			c.checkExport(fd, comments)
		}
	}
}

func isExport(fd *ast.FuncDecl) bool {
	if fd.Doc == nil {
		return false
	}
	for _, cm := range fd.Doc.List {
		if strings.HasPrefix(cm.Text, "//export ") {
			return true
		}
	}
	return false
}

// commentsIn returns the comments of a function, its doc comment included.
func commentsIn(f *ast.File, fd *ast.FuncDecl) []*ast.Comment {
	var out []*ast.Comment
	for _, g := range f.Comments {
		if g.End() < fd.Pos() || g.Pos() > fd.End() {
			continue
		}
		out = append(out, g.List...)
	}
	return out
}

var handoffRe = regexp.MustCompile(`refcheck:handoff\s+(\w+)`)

// checkExport applies rule 1 to an exported callback.
func (c *checker) checkExport(fd *ast.FuncDecl, comments []*ast.Comment) {
	if !isExport(fd) {
		return
	}
	handedOff := map[string]bool{}
	for _, cm := range comments {
		for _, m := range handoffRe.FindAllStringSubmatch(cm.Text, -1) {
			handedOff[m[1]] = true
		}
	}

	sunk := map[string]bool{}
	ast.Inspect(fd.Body, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.CompositeLit:
			for _, e := range n.Elts {
				if kv, ok := e.(*ast.KeyValueExpr); ok {
					e = kv.Value
				}
				sunk[exprKey(e)] = true
			}
		case *ast.CallExpr:
			name := callName(n)
			lower := strings.ToLower(name)
			if strings.HasPrefix(lower, "release") || strings.HasPrefix(lower, "base_release") {
				for _, a := range n.Args {
					sunk[exprKey(a)] = true
				}
			}
			if idx := c.adopts[name]; idx != nil {
				for i, a := range n.Args {
					if idx[i] {
						sunk[exprKey(a)] = true
					}
				}
			}
			if fn := c.cFuncOf(n); fn != nil {
				for i, a := range n.Args {
					if fn.params[i] == modeConsumes {
						sunk[exprKey(a)] = true
					}
				}
			}
		}
		return true
	})

	i := 0
	for _, field := range fd.Type.Params.List {
		tn, ok := cTypeName(field.Type)
		for _, n := range field.Names {
			first := i == 0
			i++
			if first || !ok || !c.refcount[tn] || n.Name == "_" {
				continue
			}
			if sunk[n.Name] || handedOff[n.Name] {
				continue
			}
			c.report(n.Pos(), "%s: parameter %s (*%s) arrives with a reference that is never released or handed off", fd.Name.Name, n.Name, tn)
		}
		if len(field.Names) == 0 {
			i++
		}
	}
}

// cFuncOf returns the C function a call C.name(...) targets, when it takes a
// reference-counted parameter.
func (c *checker) cFuncOf(call *ast.CallExpr) *cFunc {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil
	}
	if x, ok := sel.X.(*ast.Ident); !ok || x.Name != "C" {
		return nil
	}
	return c.funcs[sel.Sel.Name]
}

var ownedRe = regexp.MustCompile(`refcheck:owned`)

// checkCalls applies rule 2 to every C call of a function.
func (c *checker) checkCalls(fd *ast.FuncDecl, comments []*ast.Comment) {
	ownedLines := map[int]bool{}
	for _, cm := range comments {
		if ownedRe.MatchString(cm.Text) {
			line := c.fset.Position(cm.Pos()).Line
			ownedLines[line] = true
			ownedLines[line+1] = true
		}
	}

	// the references the function owns: an addRef, or a variable assigned
	// the result of a C call (CEF returns every struct with a new reference)
	// or of a Go function marked refcheck:fresh. Each one can be given away
	// once.
	type addRef struct {
		key  string
		pos  token.Pos
		used bool
	}
	var addRefs []*addRef
	if isExport(fd) {
		// the struct parameters of a callback but self arrive owned (rule 1)
		i := 0
		for _, field := range fd.Type.Params.List {
			for _, n := range field.Names {
				if i > 0 {
					addRefs = append(addRefs, &addRef{key: n.Name, pos: fd.Pos()})
				}
				i++
			}
		}
	}
	ast.Inspect(fd.Body, func(n ast.Node) bool {
		if as, ok := n.(*ast.AssignStmt); ok {
			for i, rhs := range as.Rhs {
				// x, err := fresh() owns x, the first result
				if call, ok := stripConversions(rhs).(*ast.CallExpr); ok && c.isFresh(call) && i < len(as.Lhs) {
					addRefs = append(addRefs, &addRef{key: exprKey(as.Lhs[i]), pos: as.Pos()})
				}
			}
		}
		if vs, ok := n.(*ast.ValueSpec); ok {
			for i, v := range vs.Values {
				if call, ok := stripConversions(v).(*ast.CallExpr); ok && c.isFresh(call) && i < len(vs.Names) {
					addRefs = append(addRefs, &addRef{key: vs.Names[i].Name, pos: vs.Pos()})
				}
			}
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		lower := strings.ToLower(callName(call))
		if strings.Contains(lower, "addref") || strings.Contains(lower, "add_ref") {
			if sel, ok := call.Fun.(*ast.SelectorExpr); ok && len(call.Args) == 0 {
				addRefs = append(addRefs, &addRef{key: exprKey(sel.X), pos: call.Pos()})
			}
			for _, a := range call.Args {
				addRefs = append(addRefs, &addRef{key: exprKey(a), pos: call.Pos()})
			}
		}
		return true
	})

	ast.Inspect(fd.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		fn := c.cFuncOf(call)
		if fn == nil {
			return true
		}
		for i, a := range call.Args {
			mode, ok := fn.params[i]
			if !ok {
				continue
			}
			if mode != modeConsumes {
				continue
			}
			inner := stripConversions(a)
			if call, ok := inner.(*ast.CallExpr); ok && c.isFresh(call) {
				continue
			}
			if id, ok := inner.(*ast.Ident); ok && id.Name == "nil" {
				continue
			}
			if ownedLines[c.fset.Position(call.Pos()).Line] || ownedLines[c.fset.Position(a.Pos()).Line] {
				continue
			}
			key := exprKey(a)
			referenced := false
			for _, r := range addRefs {
				if r.key == key && r.pos < call.Pos() && !r.used {
					r.used, referenced = true, true
					break
				}
			}
			if !referenced {
				c.report(a.Pos(), "%s consumes a reference to %s (argument %d) the function does not own: no addRef, C call result or refcheck:fresh function result precedes it", fn.name, key, i)
			}
		}
		return true
	})
}

// isFresh reports whether a call returns a reference the caller owns: a C
// function call (not a conversion) or a Go function marked refcheck:fresh.
func (c *checker) isFresh(call *ast.CallExpr) bool {
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		if x, ok := sel.X.(*ast.Ident); ok && x.Name == "C" {
			return !isConversion(call.Fun)
		}
	}
	return c.fresh[callName(call)]
}

func callName(call *ast.CallExpr) string {
	switch f := call.Fun.(type) {
	case *ast.Ident:
		return f.Name
	case *ast.SelectorExpr:
		return f.Sel.Name
	}
	return ""
}

// stripConversions removes parentheses and the pointer conversions around a
// value: unsafe.Pointer(x), (*C.T)(x), C.uintptr_t(x).
func stripConversions(e ast.Expr) ast.Expr {
	for {
		switch v := e.(type) {
		case *ast.ParenExpr:
			e = v.X
			continue
		case *ast.CallExpr:
			if len(v.Args) == 1 && isConversion(v.Fun) {
				e = v.Args[0]
				continue
			}
		}
		return e
	}
}

func isConversion(fun ast.Expr) bool {
	switch f := fun.(type) {
	case *ast.ParenExpr:
		_, star := f.X.(*ast.StarExpr)
		return star
	case *ast.SelectorExpr:
		x, ok := f.X.(*ast.Ident)
		if !ok {
			return false
		}
		if x.Name == "unsafe" && f.Sel.Name == "Pointer" {
			return true
		}
		// C.cef_x_t(...) and friends are type conversions, C.func(...) calls
		return x.Name == "C" && strings.HasSuffix(f.Sel.Name, "_t")
	}
	return false
}

func exprKey(e ast.Expr) string {
	var b strings.Builder
	printer.Fprint(&b, token.NewFileSet(), stripConversions(e))
	return b.String()
}
