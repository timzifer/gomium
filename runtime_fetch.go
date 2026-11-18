package gomium

import (
	"context"

	"github.com/timzifer/gomium/cefdl"
)

// FetchRuntime downloads the CEF binary distribution this binary was compiled
// against and unpacks its runtime into dest. It is the one-liner behind
// tools/cefdl for anything embedding gomium:
//
//	if _, err := gomium.FetchRuntime(ctx, exeDir); err != nil { ... }
//	if err := gomium.LoadLibcef(filepath.Join(exeDir, "libcef.dll")); err != nil { ... }
//
// The version comes from CEFVersion(), so the runtime always matches the
// headers this binary was built against - which is exactly what LoadLibcef
// verifies against the loaded library before the first real call. A dest that
// already holds this version is left alone, and a downloaded archive is cached
// (os.UserCacheDir()/gomium-cef) and checksum-verified, so calling this on
// every start is cheap after the first run.
//
// The platform is the one this binary runs on; cross-platform fetches go
// through FetchRuntimeWith.
func FetchRuntime(ctx context.Context, dest string) (cefdl.Result, error) {
	return FetchRuntimeWith(ctx, cefdl.Options{Dest: dest})
}

// FetchRuntimeWith is FetchRuntime with the knobs of cefdl.Options - a
// different platform, a cache directory of your own, a progress log. An empty
// Options.Version means CEFVersion(), which is the point of calling this
// instead of cefdl.Fetch directly.
func FetchRuntimeWith(ctx context.Context, options cefdl.Options) (cefdl.Result, error) {
	return cefdl.Fetch(ctx, runtimeOptions(options))
}

// runtimeOptions fills in what this binary knows about itself.
func runtimeOptions(options cefdl.Options) cefdl.Options {
	if options.Version == "" {
		options.Version = CEFVersion()
	}
	return options
}
