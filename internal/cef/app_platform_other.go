//go:build !darwin && !windows

package cef

// pumpPlatformEvents is a no-op on Linux: offscreen browsers have no native
// event queue next to CEF's own message loop, and windowed browsers are not
// implemented there. macOS (AppKit) and Windows (Win32) have their own.
func pumpPlatformEvents() (more bool) { return false }
