//go:build darwin

package main

import (
	"path/filepath"

	"github.com/timzifer/gomium"
)

// libcefPath prefers the framework inside the .app this executable runs
// from (tools/cefbundle layout): helper processes start from inside the
// bundle and have no other way to find it.
func libcefPath(dir string) string {
	if _, framework, _, err := gomium.MacBundlePaths(); err == nil {
		return framework
	}
	return filepath.Join(dir, "Chromium Embedded Framework.framework")
}

// platformOptions keeps Chromium out of the keychain (WithMockKeychain): an
// ad-hoc signed bundle is a new code identity on every build, and each one
// would raise "Chromium Safe Storage" prompts that stall the first page
// load. It also leaves the profile on disk unprotected - an application
// signed with a stable identity should drop it and let the user allow the
// keychain once.
func platformOptions() []gomium.Option {
	return []gomium.Option{gomium.WithMockKeychain(true)}
}

// beforeSubprocess installs the CefAppProtocol-aware NSApplication; CEF
// refuses to start in any process without it.
func beforeSubprocess() error { return gomium.InstallMacApplication() }

// afterRun finishes the AppKit launch that -[NSApplication run] would do -
// gomium owns the loop instead - and brings the window to the front.
func afterRun() {
	gomium.FinishMacApplicationLaunch()
	gomium.InstallMacDefaultMenu("gomium window example")
	gomium.ActivateMacApplication()
}
