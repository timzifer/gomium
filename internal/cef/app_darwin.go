//go:build darwin

package cef

/*
#cgo LDFLAGS: -framework Cocoa

#include <stdlib.h>

int cAppInstallMacApplication(void);
void cAppFinishMacApplicationLaunch(void);
void cAppActivateMacApplication(void);
void cAppInstallMacDefaultMenu(const char* appName);
int cAppPumpMacApplicationEvents(void);
*/
import "C"
import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"unsafe"
)

// InstallMacApplication creates the shared NSApplication CEF needs and makes
// it implement CefAppProtocol. Call it on the main thread (the one that
// called runtime.LockOSThread) before New, and before ExecuteSubprocess in
// helper processes:
//
//	runtime.LockOSThread()
//	if err := gomium.LoadLibcef(frameworkPath); err != nil { ... }
//	if err := gomium.InstallMacApplication(); err != nil { ... }
//
// CEF's macOS message pump asks the shared application object whether it is
// currently dispatching an event; a plain NSApplication cannot answer that
// and CEF aborts. Hosts that already run their own NSApplication - a Cocoa
// application, or a toolkit like Fyne or Ebiten - have to make that class
// implement CefAppProtocol themselves; this function then only reports
// whether it does. It is a no-op on every other platform, where the same
// call does not exist at all, so keep it behind a darwin build tag in
// cross-platform hosts.
func InstallMacApplication() error {
	if C.cAppInstallMacApplication() == 0 {
		return errors.New("gomium: the NSApplication of this process does not implement CefAppProtocol - " +
			"see include/cef_application_mac.h; CEF cannot run under it")
	}
	return nil
}

// FinishMacApplicationLaunch completes the AppKit launch sequence. Call it
// once from the browser process, on the main thread, after Run returned -
// never from a helper process, where the application object exists only to
// satisfy CefAppProtocol.
//
// gomium pumps CEF through WithMessagePumpWorker instead of -[NSApplication
// run], so nothing else calls finishLaunching: without this the process stays
// half-launched and its windows never behave. It is idempotent, and a no-op
// under a host that runs its own Cocoa event loop.
//
// It does not take focus - see ActivateMacApplication for that.
func FinishMacApplicationLaunch() {
	C.cAppFinishMacApplicationLaunch()
}

// ActivateMacApplication brings the process to the front. Call it from the
// browser process on the main thread when the application shows a window: a
// binary started from a terminal is not the active application, so its first
// window comes up behind the terminal and never becomes key.
//
// Do not call it for an offscreen-only host - there is no window to show, and
// pulling focus away from whatever the user is doing is the whole cost with
// none of the benefit.
func ActivateMacApplication() {
	C.cAppActivateMacApplication()
}

// InstallMacDefaultMenu installs a minimal main menu - Hide, Quit, and the
// standard Edit entries (undo/redo/cut/copy/paste/select all).
//
// macOS dispatches keyboard shortcuts through the menu bar, so without a main
// menu cmd+C and cmd+V never reach the page. Call it from the browser process
// on the main thread, before or after Run; it does nothing when a menu is
// already in place, so a host that brings its own menu bar keeps it.
//
// name is what appears in the application menu. Empty falls back to the name
// of the running executable.
func InstallMacDefaultMenu(name string) {
	if name == "" {
		if exe, err := os.Executable(); err == nil {
			name = strings.TrimSuffix(filepath.Base(exe), filepath.Ext(exe))
		}
	}
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))
	C.cAppInstallMacDefaultMenu(cName)
}

// pumpPlatformEvents drains the AppKit event queue, which
// cef_do_message_loop_work does not touch. Called from doMessageWork, so it
// runs wherever the host runs the pump - and does nothing off the main
// thread, where AppKit must not be touched at all.
//
// Reports whether the event budget ran out, so the caller schedules another
// pass right away instead of waiting for the next heartbeat.
func pumpPlatformEvents() (more bool) {
	return C.cAppPumpMacApplicationEvents() != 0
}
