//go:build darwin

package main

import "github.com/timzifer/gomium"

// installMacApplication puts the CefAppProtocol-aware NSApplication in place.
// This is what an embedding application has to do on macOS as well, unless it
// already runs a Cocoa application object of its own.
func installMacApplication() error {
	return gomium.InstallMacApplication()
}

// finishMacApplicationLaunch completes the AppKit launch after cef_initialize.
// Only the browser process may do this - a helper never gets here, it leaves
// through ExecuteSubprocess.
func finishMacApplicationLaunch() {
	gomium.FinishMacApplicationLaunch()
}

// activateMacApplication pulls the process to the front. Only the window
// test needs it - an offscreen run has nothing to show.
func activateMacApplication() {
	gomium.ActivateMacApplication()
}

// installMacDefaultMenu gives the harness a menu bar, without which cmd+C and
// cmd+V never reach the page in GOMIUM_WINDOW_TEST mode.
func installMacDefaultMenu() {
	gomium.InstallMacDefaultMenu("")
}

// libcefFrameworkInBundle names the CEF framework inside the application
// bundle this executable belongs to, or "" when it is not running from one -
// which is the normal case for `go run` out of the repository root. Helper
// processes have no other way to find the framework, since they are started
// from inside the bundle with a working directory of their own.
func libcefFrameworkInBundle() string {
	_, framework, _, err := gomium.MacBundlePaths()
	if err != nil {
		return ""
	}
	return framework
}
