//go:build windows

package cef

/*
#cgo LDFLAGS: -luser32
#include "app_bridge.h"

int cAppPumpWindowsMessages(int budget);
*/
import "C"
import (
	"syscall"
	"unsafe"
)

var (
	modkernel32          = syscall.NewLazyDLL("kernel32.dll")
	procGetModuleHandleW = modkernel32.NewProc("GetModuleHandleW")
)

func GetModuleHandle(modname *uint16) (handle syscall.Handle, err error) {
	r0, _, e1 := syscall.Syscall(procGetModuleHandleW.Addr(), 1, uintptr(unsafe.Pointer(modname)), 0, 0)
	handle = syscall.Handle(r0)
	if handle == 0 {
		if e1 != 0 {
			err = error(e1)
		} else {
			err = syscall.EINVAL
		}
	}
	return
}

func WithModuleHandle(handle syscall.Handle) Option {
	return func(c *cefApp) {
		c.options.moduleHandle = uintptr(handle)
	}
}

func WithMultiThreadedMessageLoop(state bool) Option {
	return func(c *cefApp) {
		c.settings.MultiThreadedMessageLoop = state
	}
}

func (c *cefApp) initializeOperatingSystemSpecificMainArgs() {
	C.cAppSetMainArgsInstance(c.mainArgs, C.uintptr_t(c.options.moduleHandle))
	c.options.logger.Trace().Msg("operating-system specific main args initialized")
}

func (c *cefApp) freeOperatingSystemSpecificMainArgs() {
	// nothing to free on windows - the instance handle is not owned by us
}

// pumpPlatformEvents drains the thread's Win32 message queue, which
// cef_do_message_loop_work does not touch (see message_pump_windows.c).
// Bounded like the macOS variant: a page that produces input as fast as it is
// consumed must not keep CEF's own work from running. Reports whether the
// budget ran out, so the caller schedules another pass right away.
func pumpPlatformEvents() (more bool) {
	return C.cAppPumpWindowsMessages(128) != 0
}
