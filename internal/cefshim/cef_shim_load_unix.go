//go:build darwin || linux

package cefshim

/*
#cgo linux LDFLAGS: -ldl

#include <dlfcn.h>
#include <stdlib.h>

#include "cef_shim.h"

// RTLD_FIRST keeps dlsym from wandering into another image if a second copy
// of libcef is already mapped. It only exists on macOS.
#ifndef RTLD_FIRST
#define RTLD_FIRST 0
#endif
*/
import "C"
import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"unsafe"
)

var (
	libcefOnce  sync.Once
	libcefErr   error
	libcefBound atomic.Bool
)

// Load loads the CEF library from path and binds every CEF entry point
// this package calls. It must run before gomium.Run or ExecuteSubprocess - on
// the browser process and on every CEF helper process (--type=...), since helpers
// are fresh instances of the same executable. Idempotent: the first call
// decides, later calls return that first result even when it was an error.
//
// path names the shared library itself (linux: Release/libcef.so) or, on
// macOS, the framework bundle - both
//
//	Release/Chromium Embedded Framework.framework
//	Release/Chromium Embedded Framework.framework/Chromium Embedded Framework
//
// are accepted.
//
// The library is opened with RTLD_LOCAL: nothing outside this package resolves
// against it, every entry point goes through the shim table.
func Load(path string) error {
	libcefOnce.Do(func() {
		libcefErr = loadLibcef(path)
	})
	return libcefErr
}

func loadLibcef(path string) error {
	if int(C.gomium_cef_slot_count()) != len(cefShimSymbols) {
		return fmt.Errorf("gomium: shim table has %d slots but %d symbols are listed - rerun go generate",
			int(C.gomium_cef_slot_count()), len(cefShimSymbols))
	}

	// An absolute path keeps dlopen from consulting the search paths at all -
	// on a relative name it would try LD_LIBRARY_PATH / DYLD_*_PATH entries
	// first and could bind a different CEF build than the one shipped next to
	// the executable.
	abs, err := filepath.Abs(resolveLibcefPath(path))
	if err != nil {
		return fmt.Errorf("gomium: resolving libcef path %q: %w", path, err)
	}

	cPath := C.CString(abs)
	defer C.free(unsafe.Pointer(cPath))

	handle := C.dlopen(cPath, C.RTLD_LAZY|C.RTLD_LOCAL|C.RTLD_FIRST)
	if handle == nil {
		return fmt.Errorf("gomium: loading %s: %s", abs, dlerror())
	}

	// Resolve everything before binding anything - on a partial failure the
	// table must stay fully unbound instead of half-pointing into a closed
	// library.
	addrs := make([]uintptr, len(cefShimSymbols))
	var missing []string
	for i, name := range cefShimSymbols {
		cName := C.CString(name)
		C.dlerror() // clear a stale error, a NULL symbol value is legal
		addr := C.dlsym(handle, cName)
		failed := addr == nil && C.dlerror() != nil
		C.free(unsafe.Pointer(cName))
		if failed {
			missing = append(missing, name)
			continue
		}
		addrs[i] = uintptr(addr)
	}
	if len(missing) > 0 {
		C.dlclose(handle)
		return fmt.Errorf("gomium: %s does not export: %s", abs, strings.Join(missing, ", "))
	}

	for i, addr := range addrs {
		C.gomium_cef_bind(C.int(i), C.uintptr_t(addr))
	}
	libcefBound.Store(true)
	return nil
}

// resolveLibcefPath turns a macOS framework bundle into the shared library
// inside it: "…/Chromium Embedded Framework.framework" becomes
// "…/Chromium Embedded Framework.framework/Chromium Embedded Framework". A
// path that already names the library - every path on Linux - is returned
// unchanged.
func resolveLibcefPath(path string) string {
	trimmed := strings.TrimRight(path, string(filepath.Separator))
	base := filepath.Base(trimmed)
	if !strings.HasSuffix(base, ".framework") {
		return path
	}
	return filepath.Join(trimmed, strings.TrimSuffix(base, ".framework"))
}

// dlerror reports the pending dynamic linker error, if any.
func dlerror() string {
	if msg := C.dlerror(); msg != nil {
		return C.GoString(msg)
	}
	return "unknown error"
}

// Ready reports whether Load has bound the shim table. It gates every path
// that leads into libcef; the atomic load also publishes the bound table to
// whichever goroutine passes it.
func Ready() error {
	if !libcefBound.Load() {
		return errors.New("gomium: libcef not loaded - call gomium.LoadLibcef(path) before Run or ExecuteSubprocess")
	}
	return nil
}
