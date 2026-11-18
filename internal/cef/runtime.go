package cef

import "github.com/timzifer/gomium/internal/cefshim"

// libcefRuntimeReady gates every path that leads into libcef. Importing
// cefshim is also what pulls its objects - the trampolines in cef_shim.S and
// the pointer table in cef_shim.c - into the link, so the cef_* symbols this
// package calls resolve at all.
func libcefRuntimeReady() error { return cefshim.Ready() }
