package cefdl

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"
)

// TestPinnedVersionMatchesHeaders fails when cefdl/cef.version and the
// vendored include/cef_version.h disagree. The loader catches the mismatch
// too, but only at runtime and only against a real distribution; this runs
// in CI without one. A CEF upgrade that bumps the pin without regenerating
// the headers (or the other way round) stops here.
func TestPinnedVersionMatchesHeaders(t *testing.T) {
	header, err := os.ReadFile(filepath.Join("..", "include", "cef_version.h"))
	if err != nil {
		t.Fatalf("reading the vendored headers: %v", err)
	}

	define := func(name string) string {
		t.Helper()
		match := regexp.MustCompile(`(?m)^#define ` + name + ` (\S+)\s*$`).FindSubmatch(header)
		if match == nil {
			t.Fatalf("include/cef_version.h defines no %s", name)
		}
		return string(match[1])
	}

	headerVersion, err := strconv.Unquote(define("CEF_VERSION"))
	if err != nil {
		t.Fatalf("CEF_VERSION is not a string literal: %v", err)
	}
	if pinned := PinnedVersion(); pinned != headerVersion {
		t.Fatalf("cefdl/cef.version pins %q, include/cef_version.h is %q - "+
			"bump both together (see docs/upgrading-cef.md)", pinned, headerVersion)
	}

	// The numeric defines are what the loader compares against the loaded
	// library; they must describe the same version as the string.
	parts := regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)\+g[0-9a-f]+\+chromium-(\d+)\.(\d+)\.(\d+)\.(\d+)$`).
		FindStringSubmatch(headerVersion)
	if parts == nil {
		t.Fatalf("CEF_VERSION %q has an unexpected format", headerVersion)
	}
	for index, name := range []string{
		"CEF_VERSION_MAJOR", "CEF_VERSION_MINOR", "CEF_VERSION_PATCH",
		"CHROME_VERSION_MAJOR", "CHROME_VERSION_MINOR", "CHROME_VERSION_BUILD", "CHROME_VERSION_PATCH",
	} {
		if got := define(name); got != parts[index+1] {
			t.Errorf("%s is %s, CEF_VERSION says %s", name, got, parts[index+1])
		}
	}
}
