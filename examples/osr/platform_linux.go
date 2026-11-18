//go:build linux

package main

import (
	"path/filepath"

	"github.com/timzifer/gomium"
)

func libcefPath(dir string) string { return filepath.Join(dir, "libcef.so") }

func platformOptions() []gomium.Option { return nil }
func beforeSubprocess() error          { return nil }
func afterRun()                        {}
