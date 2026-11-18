package gomium

import (
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// notReExported names identifiers that internal/cef exports for this package's
// benefit and that are deliberately absent from the public API.
var notReExported = map[string]bool{
	// Plumbing: the facade owns the JavaScriptEvaluationTimeout variable and
	// hands the binding a pointer to it (see vars.go). Callers set the
	// variable, never call this.
	"BindJavaScriptEvaluationTimeout": true,

	// Raw CEF value wrappers: process-message plumbing between browser and
	// renderer (argument lists, the binary payload of a binding call). They
	// left the public API with the in-process DevTools protocol - callers get
	// Go values, never ref-counted CEF structs they would have to release.
	"CefValue":              true,
	"CefListValue":          true,
	"CefBinaryValue":        true,
	"BinaryValueFromBuffer": true,
}

// TestFacadeCoversBinding fails when internal/cef grows an exported identifier
// that the facade does not forward. Without it the public API silently shrinks
// relative to the binding: everything under internal/ is unreachable from
// outside the module, so an un-forwarded symbol is not merely undocumented,
// it is gone.
//
// The check is per-GOOS because a chunk of the surface is build-tagged
// (WithModuleHandle is Windows-only, InstallMacApplication macOS-only), and a
// facade that only compiles on the host would hide exactly those.
func TestFacadeCoversBinding(t *testing.T) {
	for _, goos := range []string{"linux", "darwin", "windows"} {
		t.Run(goos, func(t *testing.T) {
			binding := exportedIn(t, "internal/cef", goos)
			facade := exportedIn(t, ".", goos)

			var missing []string
			for name := range binding {
				if !facade[name] && !notReExported[name] {
					missing = append(missing, name)
				}
			}
			sort.Strings(missing)
			for _, name := range missing {
				t.Errorf("internal/cef exports %s but the facade does not forward it "+
					"(add it, or add it to notReExported and say why)", name)
			}

			for name := range notReExported {
				if !binding[name] {
					t.Errorf("notReExported lists %s, which internal/cef no longer exports", name)
				}
			}
		})
	}
}

// TestImportsResolvePerPlatform fails when a file uses a package qualifier it
// does not import, on any platform.
//
// This is the cheap half of what a compiler would tell us, and it exists
// because the expensive half is very expensive: api_windows.go shipped calling
// syscall.Handle without importing syscall, which meant the root package had
// not compiled on Windows since the facade landed. Nothing caught it until a
// Windows runner spent 18 minutes reaching the same conclusion. Parsing finds
// it in milliseconds, before a push.
//
// The per-GOOS loop is the point: a host-only check sees neither
// api_windows.go nor api_darwin.go, which is exactly where mechanically
// generated forwards are most likely to miss an import.
func TestImportsResolvePerPlatform(t *testing.T) {
	for _, dir := range []string{".", "internal/cef", "internal/cefshim"} {
		for _, goos := range []string{"linux", "darwin", "windows"} {
			t.Run(dir+"/"+goos, func(t *testing.T) {
				files := filesFor(t, dir, goos)

				// Package-level names first: a file may qualify through a var
				// declared in a sibling file (internal/cef does, with mtx and
				// v8contextLock), and the parser cannot bind those from one
				// file alone.
				fset := token.NewFileSet()
				parsedFiles := make(map[string]*ast.File, len(files))
				packageLevel := map[string]bool{}
				for _, file := range files {
					parsed, err := parser.ParseFile(fset, file, nil, 0)
					if err != nil {
						t.Fatalf("parsing %s: %v", file, err)
					}
					parsedFiles[file] = parsed
					for name := range parsed.Scope.Objects {
						packageLevel[name] = true
					}
				}

				for _, file := range files {
					for _, name := range unimportedQualifiers(parsedFiles[file], packageLevel) {
						t.Errorf("%s uses %s. but does not import %s", file, name, name)
					}
				}
			})
		}
	}
}

// unimportedQualifiers returns the names used as package qualifiers in f that f
// does not import. packageLevel holds the names declared at package level
// across every file of the package.
//
// An identifier qualifies if it is the left side of a selector, resolves to no
// declaration in the file, and the parser could not bind it. That is what keeps
// locals out: parameters, receivers and local variables all resolve, so
// `handler.OnLoadEnd` and `ctx.GOOS` are never mistaken for packages. Names
// from a sibling file look identical to a package qualifier from inside one
// file, which is why packageLevel is subtracted too.
func unimportedQualifiers(f *ast.File, packageLevel map[string]bool) []string {
	imported := map[string]bool{}
	for _, spec := range f.Imports {
		if spec.Name != nil {
			imported[spec.Name.Name] = true
			continue
		}
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			continue
		}
		imported[path[strings.LastIndex(path, "/")+1:]] = true
	}

	unbound := map[string]bool{}
	for _, ident := range f.Unresolved {
		unbound[ident.Name] = true
	}

	missing := map[string]bool{}
	ast.Inspect(f, func(node ast.Node) bool {
		selector, ok := node.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		ident, ok := selector.X.(*ast.Ident)
		if ok && ident.Obj == nil && unbound[ident.Name] &&
			!imported[ident.Name] && !packageLevel[ident.Name] {
			missing[ident.Name] = true
		}
		return true
	})

	names := make([]string, 0, len(missing))
	for name := range missing {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// filesFor returns the Go files of a package directory that build for goos,
// as full paths. go/build only picks files - it neither compiles nor needs a C
// toolchain - so this runs the same on every host.
func filesFor(t *testing.T, dir, goos string) []string {
	t.Helper()

	ctx := build.Default
	ctx.GOOS = goos
	ctx.CgoEnabled = true
	pkg, err := ctx.ImportDir(dir, 0)
	if err != nil {
		t.Fatalf("selecting %s files for %s: %v", dir, goos, err)
	}

	var files []string
	for _, group := range [][]string{pkg.GoFiles, pkg.CgoFiles, pkg.TestGoFiles} {
		for _, file := range group {
			files = append(files, filepath.Join(dir, file))
		}
	}
	return files
}

// exportedIn returns the exported top-level identifiers of a package directory
// as selected for goos.
func exportedIn(t *testing.T, dir, goos string) map[string]bool {
	t.Helper()

	names := map[string]bool{}
	fset := token.NewFileSet()
	for _, file := range filesFor(t, dir, goos) {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		parsed, err := parser.ParseFile(fset, file, nil, 0)
		if err != nil {
			t.Fatalf("parsing %s: %v", file, err)
		}
		for _, decl := range parsed.Decls {
			switch decl := decl.(type) {
			case *ast.FuncDecl:
				// Methods travel with their type, which is aliased whole.
				if decl.Recv == nil && decl.Name.IsExported() {
					names[decl.Name.Name] = true
				}
			case *ast.GenDecl:
				for _, spec := range decl.Specs {
					switch spec := spec.(type) {
					case *ast.TypeSpec:
						if spec.Name.IsExported() {
							names[spec.Name.Name] = true
						}
					case *ast.ValueSpec:
						for _, name := range spec.Names {
							if name.IsExported() {
								names[name.Name] = true
							}
						}
					}
				}
			}
		}
	}
	return names
}
