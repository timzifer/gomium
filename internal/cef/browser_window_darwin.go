//go:build darwin

package cef

/*
#include <stdlib.h>
#include "browser_client_bridge.h"
*/
import "C"
import "unsafe"

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
func WithParentView(view unsafe.Pointer) BrowserOption {
	return func(browser *browser) {
		browser.settings.parentView = view
	}
}

// applyWindowInfo fills the parts of cef_window_info_t that a windowed
// browser needs. cef_window_info_t is an operating-system specific struct -
// the macOS variant has no window style and no window handle, but a
// parent_view and a hidden flag, which is why this lives in a darwin-only
// file.
//
// The runtime style is pinned to Alloy on purpose: Chrome style would bring
// Chromium's own browser interface along, and the point of this window is to
// show one page and nothing else. With a parent_view CEF forces Alloy style
// anyway.
func (b *browser) applyWindowInfo() error {
	bounds := b.settings.windowBounds

	b.windowInfo.bounds = C.cef_rect_t{
		x:      C.int(bounds.Min.X),
		y:      C.int(bounds.Min.Y),
		width:  C.int(bounds.Dx()),
		height: C.int(bounds.Dy()),
	}
	// hidden stays 0: the browser is meant to be visible as soon as it
	// exists, same as the windowed browsers on Windows.
	b.windowInfo.runtime_style = C.CEF_RUNTIME_STYLE_ALLOY

	if b.settings.parentView != nil {
		// A view owned by AppKit, never Go memory - so it passes the cgo
		// pointer check and stays valid without pinning.
		b.windowInfo.parent_view = C.cef_window_handle_t(b.settings.parentView)
	}

	if b.settings.windowTitle != "" {
		// The detour over C memory is not decoration. windowInfo sits in the
		// Go-allocated browser struct, and handing &windowInfo.window_name to
		// a C function means handing an interior Go pointer to cgo's pointer
		// check - which then examines the enclosing object, finds the Go
		// pointers of settings (the URL string among them) and aborts with
		// "cgo argument has Go pointer to unpinned Go pointer". Whether it
		// fires depends on the go directive of the MAIN module, so the same
		// call survives gomium's own test app and kills a host built with a
		// newer toolchain: exactly the kind of bug that only appears at the
		// customer's.
		//
		// cefString allocates the container in C memory and fills it there.
		// What arrives here is a plain struct copy - three fields, one of them
		// the UTF-16 buffer whose ownership moves into windowInfo. Only the
		// container is given back; the buffer belongs to window_name from now
		// on and is released by releaseWindowInfo.
		name := cefString(b.settings.windowTitle)
		b.windowInfo.window_name = *name
		C.free(unsafe.Pointer(name))
	}
	return nil
}

// releaseWindowInfo gives back what applyWindowInfo allocated. CEF copies the
// window info during browser creation, so this runs as soon as the creation
// call returns - successful or not.
//
// The same detour, for the same reason: the string is moved into a C-allocated
// container and cleared there, instead of pointing cef_string_clear at Go
// memory.
func (b *browser) releaseWindowInfo() {
	if b.windowInfo.window_name.str == nil {
		return
	}
	container := (*C.cef_string_t)(C.calloc(1, C.sizeof_cef_string_t))
	*container = b.windowInfo.window_name
	b.windowInfo.window_name = C.cef_string_t{}
	cefStringFree(container)
}
