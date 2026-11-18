package gomium

import (
	"testing"

	"github.com/timzifer/gomium/cefdl"
)

// An empty version means "the one this binary was compiled against" - that is
// the whole point of going through the root package instead of calling
// cefdl.Fetch directly.
func TestRuntimeOptionsDefaultToCompiledVersion(t *testing.T) {
	filled := runtimeOptions(cefdl.Options{Dest: "Release"})
	if filled.Version != CEFVersion() {
		t.Fatalf("version %q, want %q", filled.Version, CEFVersion())
	}
	if filled.Dest != "Release" {
		t.Fatalf("dest %q", filled.Dest)
	}
}

func TestRuntimeOptionsKeepExplicitValues(t *testing.T) {
	given := cefdl.Options{
		Version:  "150.0.20+ga832838+chromium-150.0.7871.253",
		Platform: "linuxarm64",
		Dest:     "out",
		CacheDir: "cache",
		Force:    true,
	}
	filled := runtimeOptions(given)
	if filled.Version != given.Version || filled.Platform != given.Platform ||
		filled.Dest != given.Dest || filled.CacheDir != given.CacheDir || !filled.Force {
		t.Fatalf("options were changed: %+v", filled)
	}
}
