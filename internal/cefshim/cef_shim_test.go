package cefshim

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// symbolsFromFile reads cef_shim_symbols.txt the way the generator does.
func symbolsFromFile(t *testing.T) []string {
	t.Helper()

	f, err := os.Open("cef_shim_symbols.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	var syms []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		syms = append(syms, line)
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	return syms
}

// TestShimSymbolsMatchSymbolFile guards the index contract: slot i in
// cef_shim.c is the symbol at index i in cefShimSymbols, and Load binds
// by that index. A symbol added to the list without regenerating would shift
// every slot behind it and bind every call to the wrong entry point.
func TestShimSymbolsMatchSymbolFile(t *testing.T) {
	want := symbolsFromFile(t)
	if len(want) != len(cefShimSymbols) {
		t.Fatalf("cef_shim_symbols.txt lists %d symbols, cefShimSymbols has %d - rerun go generate",
			len(want), len(cefShimSymbols))
	}
	for i := range want {
		if want[i] != cefShimSymbols[i] {
			t.Errorf("slot %d: symbol file has %q, cefShimSymbols has %q", i, want[i], cefShimSymbols[i])
		}
	}
}

// TestShimTrampolinesPerArchitecture checks that every symbol has a
// trampoline in every architecture block of cef_shim.S. The blocks are
// selected by the preprocessor, so a missing one only shows up when someone
// builds for that architecture - here it shows up right away.
func TestShimTrampolinesPerArchitecture(t *testing.T) {
	asm, err := os.ReadFile("cef_shim.S")
	if err != nil {
		t.Fatal(err)
	}
	text := string(asm)

	for _, sym := range symbolsFromFile(t) {
		for _, tc := range []struct {
			arch string
			line string
		}{
			{"amd64", "jmp *CSYM(__pfn_" + sym + ")(%rip)"},
			{"arm64 (mach-o)", "adrp x16, CSYM(__pfn_" + sym + ")@GOTPAGE"},
			{"arm64 (elf)", "adrp x16, :got:__pfn_" + sym},
		} {
			if !strings.Contains(text, tc.line) {
				t.Errorf("cef_shim.S has no %s trampoline for %s (missing %q)", tc.arch, sym, tc.line)
			}
		}
	}
}

// TestShimGeneratorIsUpToDate reruns the generator and compares its output
// with the checked-in files, so a hand edit or a stale generate does not
// survive a test run.
func TestShimGeneratorIsUpToDate(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the generator")
	}
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("no go toolchain in PATH")
	}

	out := t.TempDir()
	cmd := exec.Command("go", "run", "../../tools/cefshim", "-out", out)
	if combined, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("running the generator: %v\n%s", err, combined)
	}

	for _, name := range []string{"cef_shim.S", "cef_shim.c", "cef_shim_symbols.go"} {
		want, err := os.ReadFile(filepath.Join(out, name))
		if err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != string(want) {
			t.Errorf("%s differs from generator output - rerun go generate", name)
		}
	}
}
