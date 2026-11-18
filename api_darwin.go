//go:build darwin

package gomium

import (
	cef "github.com/timzifer/gomium/internal/cef"
	"unsafe"
)

// macOS-only surface.

// MacFrameworkName is the base name of the CEF framework bundle, both in the
// downloaded distribution (Release/) and inside an application bundle.
const MacFrameworkName = cef.MacFrameworkName

// ActivateMacApplication brings the process to the front. Call it from the
// browser process on the main thread when the application shows a window: a
// binary started from a terminal is not the active application, so its first
// window comes up behind the terminal and never becomes key.
//
// Do not call it for an offscreen-only host - there is no window to show, and
// pulling focus away from whatever the user is doing is the whole cost with
// none of the benefit.
func ActivateMacApplication() { cef.ActivateMacApplication() }

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
func FinishMacApplicationLaunch() { cef.FinishMacApplicationLaunch() }

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
func InstallMacApplication() error { return cef.InstallMacApplication() }

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
func InstallMacDefaultMenu(name string) { cef.InstallMacDefaultMenu(name) }

// MacBundlePaths derives the CEF paths of the running process from its own
// executable. It answers for both layouts CEF launches:
//
//	<App>.app/Contents/MacOS/<App>                                        browser
//	<App>.app/Contents/Frameworks/<App> Helper.app/Contents/MacOS/<App> Helper
//
// bundle is the top-level .app, framework the CEF framework bundle inside it
// (the path LoadLibcef takes), and helper the default helper executable - the
// one CEF derives itself when browser_subprocess_path stays empty.
//
// It reports an error when the executable does not sit inside an application
// bundle, which is the normal case during development: callers fall back to
// the downloaded ./Release distribution then. Nothing here touches the file
// system beyond resolving the executable, so a returned path is a claim about
// the layout, not proof that the framework is there.
func MacBundlePaths() (string, string, string, error) { return cef.MacBundlePaths() }

// WithFrameworkDirPath points CEF at the framework bundle. path must be
// absolute; a relative one is resolved against the working directory.
//
// Leave it unset for an application bundle built by tools/cefbundle - CEF then
// finds the framework at its conventional place,
// "Contents/Frameworks/Chromium Embedded Framework.framework". Setting it is
// for hosts that keep the framework somewhere else, and for running a bare
// executable against ./Release during development.
//
// This is macOS only; the same setting does not exist on other platforms,
// so keep the call behind a darwin build tag in cross-platform hosts.
func WithFrameworkDirPath(path string) Option { return cef.WithFrameworkDirPath(path) }

// WithMainBundlePath names the .app bundle CEF should treat as the main
// bundle. path must be absolute.
//
// Only needed when the running executable is not inside the bundle it belongs
// to - CEF otherwise takes the top-level bundle of the current executable,
// which is what tools/cefbundle produces.
func WithMainBundlePath(path string) Option { return cef.WithMainBundlePath(path) }

// WithParentView creates the browser inside an existing NSView instead of a
// window of its own. view is an NSView pointer - what a host already running
// its own Cocoa window (Fyne, Ebiten, an Objective-C application) hands out
// as its content view.
//
// Combine it with WithWindow / WithWindowBounds: the bounds are then read as
// the browser's frame inside the parent view instead of screen coordinates,
// and the window title is ignored because the window belongs to the host.
//
// Without a parent view CEF creates and owns the NSWindow itself.
func WithParentView(view unsafe.Pointer) BrowserOption { return cef.WithParentView(view) }
