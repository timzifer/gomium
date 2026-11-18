//go:build darwin

package cef

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// MacFrameworkName is the base name of the CEF framework bundle, both in the
// downloaded distribution (Release/) and inside an application bundle.
const MacFrameworkName = "Chromium Embedded Framework"

// WithFrameworkDirPath points CEF at the framework bundle. path must be
// absolute; a relative one is resolved against the working directory.
//
// Leave it unset for an application bundle built by tools/cefbundle - CEF then
// finds the framework at its conventional place,
// "Contents/Frameworks/Chromium Embedded Framework.framework". Setting it is
// for hosts that keep the framework somewhere else, and for running a bare
// executable against ./Release during development.
//
// This is macOS only; the same setting does not exist on other platforms,
// so keep the call behind a darwin build tag in cross-platform hosts.
func WithFrameworkDirPath(path string) Option {
	return func(c *cefApp) {
		if abs, ok := absoluteOption(c, "WithFrameworkDirPath", path); ok {
			c.settings.FrameworkDirPath = &abs
		}
	}
}

// WithMainBundlePath names the .app bundle CEF should treat as the main
// bundle. path must be absolute.
//
// Only needed when the running executable is not inside the bundle it belongs
// to - CEF otherwise takes the top-level bundle of the current executable,
// which is what tools/cefbundle produces.
func WithMainBundlePath(path string) Option {
	return func(c *cefApp) {
		if abs, ok := absoluteOption(c, "WithMainBundlePath", path); ok {
			c.settings.MainBundlePath = &abs
		}
	}
}

// MacBundlePaths derives the CEF paths of the running process from its own
// executable. It answers for both layouts CEF launches:
//
//	<App>.app/Contents/MacOS/<App>                                        browser
//	<App>.app/Contents/Frameworks/<App> Helper.app/Contents/MacOS/<App> Helper
//
// bundle is the top-level .app, framework the CEF framework bundle inside it
// (the path LoadLibcef takes), and helper the default helper executable - the
// one CEF derives itself when browser_subprocess_path stays empty.
//
// It reports an error when the executable does not sit inside an application
// bundle, which is the normal case during development: callers fall back to
// the downloaded ./Release distribution then. Nothing here touches the file
// system beyond resolving the executable, so a returned path is a claim about
// the layout, not proof that the framework is there.
func MacBundlePaths() (bundle, framework, helper string, err error) {
	exe, err := os.Executable()
	if err != nil {
		return "", "", "", fmt.Errorf("gomium: locating the executable: %w", err)
	}
	// The executable may be reached through a symlink; the bundle layout is
	// only visible on the resolved path.
	if resolved, resolveErr := filepath.EvalSymlinks(exe); resolveErr == nil {
		exe = resolved
	}
	return macBundlePathsFor(exe)
}

// macBundlePathsFor is MacBundlePaths on an explicit executable path, so the
// layout rules can be exercised without an actual bundle on disk.
func macBundlePathsFor(exe string) (bundle, framework, helper string, err error) {
	bundle, err = enclosingAppBundle(exe)
	if err != nil {
		return "", "", "", err
	}
	// A helper executable lives in a bundle of its own, nested in the
	// Frameworks directory of the application bundle:
	//   <App>.app/Contents/Frameworks/<App> Helper.app
	// Climb out of it, once - helper bundles are never nested deeper.
	if frameworks := filepath.Dir(bundle); filepath.Base(frameworks) == "Frameworks" {
		if contents := filepath.Dir(frameworks); filepath.Base(contents) == "Contents" {
			if outer := filepath.Dir(contents); filepath.Ext(outer) == ".app" {
				bundle = outer
			}
		}
	}

	name := strings.TrimSuffix(filepath.Base(bundle), ".app")
	frameworks := filepath.Join(bundle, "Contents", "Frameworks")
	return bundle,
		filepath.Join(frameworks, MacFrameworkName+".framework"),
		filepath.Join(frameworks, name+" Helper.app", "Contents", "MacOS", name+" Helper"),
		nil
}

// enclosingAppBundle returns the ".app" directory an executable belongs to.
// Only the exact layout "<Something>.app/Contents/MacOS/<exe>" counts - that
// is the one AppKit and CEF recognise, and a looser match would hand back a
// bundle whose Frameworks directory has nothing to do with this binary.
func enclosingAppBundle(exe string) (string, error) {
	macOS := filepath.Dir(exe)
	contents := filepath.Dir(macOS)
	app := filepath.Dir(contents)
	if filepath.Base(macOS) != "MacOS" || filepath.Base(contents) != "Contents" || filepath.Ext(app) != ".app" {
		return "", fmt.Errorf("gomium: %s is not inside a .app bundle", exe)
	}
	return app, nil
}
