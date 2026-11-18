package gomium

// The CEF runtime libraries are not checked into the repository. They are
// fetched from the official CEF build CDN into the git-ignored ./Release
// directory. Nothing links against them at build time - LoadLibcef binds
// libcef.dll at runtime through the generated shim (see tools/cefshim), so a
// wrong or missing distribution is a startup error, not a link error.
//
// The version is pinned in cefdl/cef.version and embedded into the cefdl
// package (cefdl.PinnedVersion); tools/cefdl is the command that uses it, the cefdl package is the same logic as a library, and FetchRuntime wraps
// it with CEFVersion() already filled in - so anything depending on gomium can
// put the matching runtime into a directory of its own, on any platform CEF
// builds for:
//
//	gomium.FetchRuntime(ctx, exeDir)
// It has to match the vendored headers in include/, because the loader
// compares both before the first real call.
//
//go:generate go run ./tools/cefdl
