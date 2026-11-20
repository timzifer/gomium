package cef

/*
#include "app_bridge.h"
*/
import "C"
import (
	"errors"
	"os"
	"runtime"
	"strings"
	"unsafe"
)

// IsSubprocess reports whether the current process was launched by CEF as a
// helper process (renderer, gpu-process, utility, crashpad-handler, ...).
// CEF starts these as new instances of the main executable with a --type=
// switch. Hosts must detect this before any application logic runs -
// single-instance guards or setup work in the main flow would otherwise
// kill or delay the helper and leave the browser hanging.
func IsSubprocess() bool {
	for _, arg := range os.Args[1:] {
		if strings.HasPrefix(arg, "--type=") {
			return true
		}
	}
	return false
}

// ExecuteSubprocess runs the CEF helper-process loop and terminates the
// process with its exit code - it does not return on success. Call it from
// main() when IsSubprocess() is true, before any other initialization.
// Returns an error only when setup fails or the process is not actually a
// CEF helper process.
func ExecuteSubprocess(opts ...Option) error {
	runtime.LockOSThread()

	app, err := New(opts...)
	if err != nil {
		return err
	}
	c := app.(*cefApp)

	release := func() {
		mtx.Lock()
		c.freeNativeStructs()
		instance = nil
		mtx.Unlock()
	}

	// verifies the DLL version AND configures the API version - without the
	// cef_api_hash call CEF aborts cef_execute_process with FATAL
	// "CefApp_0_CToCpp called with invalid version -1"
	if err := c.verifyAndConfigureLibrary(); err != nil {
		release()
		return err
	}

	c.applyProcessSwitches()

	// cef_execute_process consumes a reference on the app struct. For a
	// helper process it runs the process main loop and only returns when
	// the helper is done.
	addRefBase(unsafe.Pointer(c.ptr))
	exitCode := int(C.cef_execute_process(c.mainArgs, c.ptr, nil))
	if exitCode < 0 {
		release()
		return errors.New("cef_execute_process returned -1: this is the browser process, not a helper")
	}
	os.Exit(exitCode)
	return nil
}
