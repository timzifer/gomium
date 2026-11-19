// Package cefdl fetches a CEF binary distribution from the official build CDN
// and unpacks its runtime into a directory of the caller's choosing.
//
// It exists so that anything depending on gomium can produce the runtime that
// belongs to the version it was built against, without a checked-out gomium
// working copy: the caller passes the version (gomium.CEFVersion(), or a
// pinned string of its own) and a destination, and gets back a directory that
// LoadLibcef can bind against.
//
// The package is deliberately free of cgo so that build tooling can import it
// without dragging in the CEF shim.
//
// Command-line front end: tools/cefdl.
package cefdl

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	// IndexURL lists every build the CDN offers, per platform.
	IndexURL = "https://cef-builds.spotifycdn.com/index.json"
	// FileURL is the prefix the index' file names hang off.
	FileURL = "https://cef-builds.spotifycdn.com/"
	// ArchiveType is the distribution flavour that is downloaded. "minimal"
	// carries Release/ + Resources/ + include/ and no debug symbols, and is
	// bzip2 compressed, which the standard library decompresses without any
	// external dependency.
	ArchiveType = "minimal"
	// DebugArchiveType is the flavour Options.Debug downloads. "standard" is
	// the only one that carries Debug/: a libcef built with DCHECKs, which
	// turn reference-count errors into crashes and report the wrapper
	// objects still alive at cef_shutdown.
	DebugArchiveType = "standard"
	// MarkerFile records the version installed in a destination directory so
	// that a repeated Fetch can skip the work.
	MarkerFile = ".cef-version"
)

// Options configures Fetch.
type Options struct {
	// Version is the CEF version to fetch. Besides an exact version string it
	// accepts a branch prefix ("151", "151.3") and the channel names "stable"
	// and "beta". Required.
	Version string
	// Platform is a CEF platform key such as windows64, linux64 or
	// macosarm64. Empty means the platform this binary runs on.
	Platform string
	// Dest receives the runtime files: Release/ (Debug/ with Debug) and
	// Resources/ of the archive
	// are flattened into it, which is the layout the loader expects. Required.
	Dest string
	// CacheDir holds the downloaded archives. Empty means
	// os.UserCacheDir()/gomium-cef.
	CacheDir string
	// HeaderDest, when set, also unpacks the archive's include/ directory
	// there. Off by default: a gomium checkout carries patched headers that
	// must not be overwritten.
	HeaderDest string
	// Force unpacks even when Dest already holds this version.
	Force bool
	// Debug unpacks the debug runtime - Debug/ of the standard archive, about
	// twice the download of the minimal one - instead of Release/. It is for
	// checking gomium itself, not for shipping: the debug libcef is slow and
	// stops the process on the first failed DCHECK.
	Debug bool
	// Log receives progress lines. Nil is silent.
	Log func(string)
	// Client is used for index and download. Nil uses a default client with a
	// generous timeout - the archives are a few hundred MiB.
	Client *http.Client
}

// Result describes what Fetch did.
type Result struct {
	// Version is the resolved CEF version, even when Options.Version was a
	// branch prefix or a channel name.
	Version string
	// Dest is the directory holding the runtime.
	Dest string
	// Archive is the downloaded (or cached) archive file.
	Archive string
	// Files counts the entries written; zero when Skipped.
	Files int
	// Skipped reports that Dest already held this version.
	Skipped bool
}

func (o Options) logf(format string, args ...any) {
	if o.Log != nil {
		o.Log(fmt.Sprintf(format, args...))
	}
}

func (o Options) archiveType() string {
	if o.Debug {
		return DebugArchiveType
	}
	return ArchiveType
}

// binDir is the directory of the archive whose binaries land in Dest.
func (o Options) binDir() string {
	if o.Debug {
		return "Debug"
	}
	return "Release"
}

// markerValue is what MarkerFile holds for a version: the version, followed
// by " debug" for the debug runtime, so switching flavours unpacks again.
func (o Options) markerValue(version string) string {
	if o.Debug {
		return version + " debug"
	}
	return version
}

func (o Options) client() *http.Client {
	if o.Client != nil {
		return o.Client
	}
	return &http.Client{Timeout: 30 * time.Minute}
}

func (o Options) cacheDir() (string, error) {
	if o.CacheDir != "" {
		return o.CacheDir, nil
	}
	base, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("resolve cache dir: %w", err)
	}
	return filepath.Join(base, "gomium-cef"), nil
}

// Fetch downloads the distribution and unpacks its runtime into o.Dest.
//
// A destination that already carries the wanted version is left alone unless
// Force is set; a cached archive whose checksum still matches is not
// downloaded again.
func Fetch(ctx context.Context, o Options) (Result, error) {
	if strings.TrimSpace(o.Version) == "" {
		return Result{}, fmt.Errorf("no CEF version given")
	}
	if strings.TrimSpace(o.Dest) == "" {
		return Result{}, fmt.Errorf("no destination given")
	}
	if o.Platform == "" {
		o.Platform = DefaultPlatform()
	}
	if o.Platform == "" {
		return Result{}, fmt.Errorf("no CEF platform key for this system - set Options.Platform")
	}

	build, archiveFile, err := resolve(ctx, o)
	if err != nil {
		return Result{}, err
	}
	result := Result{Version: build.CEFVersion, Dest: o.Dest}

	marker := filepath.Join(o.Dest, MarkerFile)
	if !o.Force {
		if current, err := os.ReadFile(marker); err == nil && strings.TrimSpace(string(current)) == o.markerValue(build.CEFVersion) {
			o.logf("%s already installed in %s", build.CEFVersion, o.Dest)
			result.Skipped = true
			return result, nil
		}
	}

	cacheDir, err := o.cacheDir()
	if err != nil {
		return result, err
	}
	archive, err := download(ctx, o, archiveFile, cacheDir)
	if err != nil {
		return result, err
	}
	result.Archive = archive

	o.logf("unpacking %s -> %s", filepath.Base(archive), o.Dest)
	written, err := extract(archive, o.Dest, o.HeaderDest, o.binDir())
	if err != nil {
		return result, err
	}
	result.Files = written

	if err := os.WriteFile(marker, []byte(o.markerValue(build.CEFVersion)+"\n"), 0o644); err != nil {
		return result, fmt.Errorf("write marker: %w", err)
	}
	o.logf("%d files written, %s ready in %s", written, build.CEFVersion, o.Dest)
	return result, nil
}

// ReadVersionFile reads a pinned version from a file such as gomium's
// cefdl/cef.version. The pin itself is embedded: see PinnedVersion.
func ReadVersionFile(name string) (string, error) {
	content, err := os.ReadFile(name)
	if err != nil {
		return "", fmt.Errorf("read pinned version: %w", err)
	}
	version := strings.TrimSpace(string(content))
	if version == "" {
		return "", fmt.Errorf("%s is empty", name)
	}
	return version, nil
}
