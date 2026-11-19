package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRun checks the findings on testdata/pkg against its "want: <text>"
// comments: every comment needs a finding on its line that contains the text,
// and every finding needs a comment.
func TestRun(t *testing.T) {
	dir := filepath.Join("testdata", "pkg")
	findings, err := run(dir, filepath.Join("testdata", "include"))
	if err != nil {
		t.Fatal(err)
	}

	type key struct {
		file string
		line int
	}
	want := map[key]string{}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		for i, line := range strings.Split(string(b), "\n") {
			if _, text, ok := strings.Cut(line, "// want: "); ok {
				want[key{e.Name(), i + 1}] = strings.TrimSpace(text)
			}
		}
	}

	for _, f := range findings {
		k := key{filepath.Base(f.pos.Filename), f.pos.Line}
		text, ok := want[k]
		if !ok {
			t.Errorf("unexpected finding %s", f)
			continue
		}
		if !strings.Contains(f.msg, text) {
			t.Errorf("finding %s does not contain %q", f, text)
		}
		delete(want, k)
	}
	for k, text := range want {
		t.Errorf("%s:%d: missing finding %q", k.file, k.line, text)
	}
}
