//go:build windows

package cefshim

/*
#include "cef_shim.h"
*/
import "C"
import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"golang.org/x/sys/windows"
)

var (
	libcefOnce  sync.Once
	libcefErr   error
	libcefBound atomic.Bool
)

// Load loads libcef.dll from path and binds every CEF entry point this
// package calls. It must run before Run or ExecuteSubprocess - on the
// browser process and on every CEF helper process (--type=...), since
// helpers are fresh instances of the same executable. Idempotent: the first
// call decides, later calls return that first result even when it was an
// error.
//
// The DLL is loaded with LOAD_WITH_ALTERED_SEARCH_PATH, so its dependent
// DLLs (chrome_elf.dll, vulkan-1.dll, dxcompiler.dll, ...) resolve from the
// directory containing libcef.dll - the CEF distribution stays
// self-contained no matter where the executable lives.
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

	// LOAD_WITH_ALTERED_SEARCH_PATH only alters the search order when the
	// given path is absolute.
	abs, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("gomium: resolving libcef path %q: %w", path, err)
	}

	handle, err := windows.LoadLibraryEx(abs, 0, windows.LOAD_WITH_ALTERED_SEARCH_PATH)
	if err != nil {
		return fmt.Errorf("gomium: loading %s: %w", abs, err)
	}

	// Resolve everything before binding anything - on a partial failure the
	// table must stay fully unbound instead of half-pointing into a freed
	// library.
	addrs := make([]uintptr, len(cefShimSymbols))
	var missing []string
	for i, name := range cefShimSymbols {
		addr, err := windows.GetProcAddress(handle, name)
		if err != nil {
			missing = append(missing, name)
			continue
		}
		addrs[i] = addr
	}
	if len(missing) > 0 {
		windows.FreeLibrary(handle)
		return fmt.Errorf("gomium: %s does not export: %s", abs, strings.Join(missing, ", "))
	}

	for i, addr := range addrs {
		C.gomium_cef_bind(C.int(i), C.uintptr_t(addr))
	}
	libcefBound.Store(true)
	return nil
}

// Ready gates every path that leads into libcef; the atomic
// load also publishes the bound table to whichever goroutine passes it.
func Ready() error {
	if !libcefBound.Load() {
		return errors.New("gomium: libcef.dll not loaded - call gomium.LoadLibcef(path) before Run or ExecuteSubprocess")
	}
	return nil
}
