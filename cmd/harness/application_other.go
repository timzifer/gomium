//go:build !darwin

package main

// installMacApplication is a no-op away from macOS: no other platform needs
// an application object before CEF starts.
func installMacApplication() error { return nil }

// finishMacApplicationLaunch is a no-op away from macOS.
func finishMacApplicationLaunch() {}

// activateMacApplication is a no-op away from macOS.
func activateMacApplication() {}

// installMacDefaultMenu is a no-op away from macOS.
func installMacDefaultMenu() {}

// libcefFrameworkInBundle is a no-op away from macOS - application bundles
// are a macOS concept.
func libcefFrameworkInBundle() string { return "" }
