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

func platformOptions() []gomium.Option { return nil }

// beforeSubprocess installs the CefAppProtocol-aware NSApplication; CEF
// refuses to start in any process without it.
func beforeSubprocess() error { return gomium.InstallMacApplication() }

// afterRun finishes the AppKit launch that -[NSApplication run] would do -
// gomium owns the loop instead.
func afterRun() {
	gomium.FinishMacApplicationLaunch()
	gomium.InstallMacDefaultMenu("gomium osr example")
}
