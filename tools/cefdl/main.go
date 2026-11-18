// Command cefdl downloads the pinned CEF binary distribution and unpacks the
// runtime libraries into ./Release (or -dest), where LoadLibcef binds them.
//
// The pinned version is embedded (cefdl.PinnedVersion), so the command works
// from any module that depends on gomium, not only from a gomium checkout:
//
//	go get github.com/timzifer/gomium@latest
//	go run github.com/timzifer/gomium/tools/cefdl -dest ./Release
//
// The work itself lives in the cefdl package, so anything depending on gomium
// can fetch the matching runtime without a gomium checkout:
//
//	gomium.FetchRuntime(ctx, dir)                                  // this binary's version
//	cefdl.Fetch(ctx, cefdl.Options{Version: "151.3", Dest: dir})    // any other
//
// The build index is the official Spotify-hosted CEF automated build CDN
// (https://cef-builds.spotifycdn.com). Only the "minimal" archive is used: it
// carries the release binaries plus the resource files and no debug symbols.
//
// Usage:
//
//	go generate ./...            # fetch the version pinned in cefdl/cef.version
//	go run ./tools/cefdl -list   # show the newest builds for this platform
//	go run ./tools/cefdl -version 151.3.16+gbe1e15d+chromium-151.0.7922.109
//	go run ./tools/cefdl -headers -force
//	go run ./tools/cefdl -dest ../elsewhere/Release
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/timzifer/gomium/cefdl"
)

func main() {
	var (
		versionFlag  = flag.String("version", "", "CEF version to fetch (default: the version this gomium module pins, "+cefdl.PinnedVersion()+")")
		versionFile  = flag.String("version-file", "", "read the CEF version from this file instead of the embedded pin")
		platformFlag = flag.String("platform", cefdl.DefaultPlatform(), "CEF platform key, e.g. windows64, linux64, macosarm64")
		destFlag     = flag.String("dest", "Release", "directory receiving the CEF runtime files")
		cacheFlag    = flag.String("cache", "", "download cache directory (default: os.UserCacheDir()/gomium-cef)")
		headersFlag  = flag.Bool("headers", false, "also unpack include/ headers into ./include (gomium development; overwrites local header patches)")
		forceFlag    = flag.Bool("force", false, "re-unpack even if the destination already holds this version")
		listFlag     = flag.Bool("list", false, "list the newest available builds for the platform and exit")
	)
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := run(ctx, *versionFlag, *versionFile, *platformFlag, *destFlag, *cacheFlag, *headersFlag, *forceFlag, *listFlag); err != nil {
		fmt.Fprintln(os.Stderr, "cefdl:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, wantVersion, versionFile, platform, dest, cacheDir string, headers, force, list bool) error {
	if list {
		builds, err := cefdl.List(ctx, platform, 15, nil)
		if err != nil {
			return err
		}
		fmt.Printf("newest %s builds:\n", platform)
		for _, build := range builds {
			fmt.Printf("  %-8s %s\n", build.Channel, build.CEFVersion)
		}
		return nil
	}

	if wantVersion == "" && versionFile != "" {
		var err error
		if wantVersion, err = cefdl.ReadVersionFile(versionFile); err != nil {
			return err
		}
	}
	if wantVersion == "" {
		wantVersion = cefdl.PinnedVersion()
	}

	// The checked-in headers carry local patches, so include/ is only touched
	// on request - and then in place, next to the module.
	headerDest := ""
	if headers {
		headerDest = "include"
	}

	_, err := cefdl.Fetch(ctx, cefdl.Options{
		Version:    wantVersion,
		Platform:   platform,
		Dest:       dest,
		CacheDir:   cacheDir,
		HeaderDest: headerDest,
		Force:      force,
		Log:        func(line string) { fmt.Println("cefdl: " + line) },
	})
	return err
}
