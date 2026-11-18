//go:build windows

package main

import (
	"log"
	"path/filepath"

	"github.com/timzifer/gomium"
)

func libcefPath(dir string) string { return filepath.Join(dir, "libcef.dll") }

// platformOptions hands CEF the process HINSTANCE, which it wants in its
// main arguments on Windows.
func platformOptions() []gomium.Option {
	handle, err := gomium.GetModuleHandle(nil)
	if err != nil {
		log.Fatal(err)
	}
	return []gomium.Option{gomium.WithModuleHandle(handle)}
}

func beforeSubprocess() error { return nil }
func afterRun()               {}
