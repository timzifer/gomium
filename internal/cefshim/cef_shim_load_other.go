//go:build !windows && !darwin && !linux

package cefshim

/*
// This package ships cef_shim.c and cef_shim.S. The go command refuses to
// build a package that carries C or assembly sources without at least one cgo
// file ("C source files not allowed when not using cgo or SWIG"), and on the
// platforms this file covers none of the real loaders apply - so the preamble
// below exists purely to keep the package loadable there.
#include "cef_shim.h"
*/
import "C"

import "errors"

var errLibcefLoadUnsupported = errors.New("gomium: runtime libcef loading is implemented for windows, macos and linux only")

// Load is a stub on platforms CEF does not publish builds for.
func Load(path string) error {
	return errLibcefLoadUnsupported
}

// Ready reports that no libcef can have been bound on this platform.
func Ready() error {
	return errLibcefLoadUnsupported
}
