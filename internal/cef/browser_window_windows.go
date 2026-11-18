//go:build windows

package cef

/*
#include <stdlib.h>
#include "browser_client_bridge.h"
*/
import "C"
import "unsafe"

// The CreateWindowEx styles a hosted browser window needs. They are spelled
// out instead of taken from windows.h because cgo would have to parse the
// whole header for four constants that have not moved since Windows 3.
const (
	wsOverlappedWindow = 0x00CF0000
	wsClipChildren     = 0x02000000
	wsClipSiblings     = 0x04000000
	wsVisible          = 0x10000000
)

// applyWindowInfo fills the parts of cef_window_info_t that a windowed
// browser needs. cef_window_info_t is an operating-system specific struct -
// the window style below exists on Windows only, and the handles are an
// HWND here, an X11 window id on Linux and an NSView on macOS - which is why
// this lives in a windows-only file.
//
// The runtime style is pinned to Alloy on purpose: Chrome style would bring
// Chromium's own browser interface along, and the point of this window is to
// show one page and nothing else.
func (b *browser) applyWindowInfo() error {
	bounds := b.settings.windowBounds

	b.windowInfo.style = C.DWORD(wsOverlappedWindow | wsClipChildren | wsClipSiblings | wsVisible)
	b.windowInfo.bounds = C.cef_rect_t{
		x:      C.int(bounds.Min.X),
		y:      C.int(bounds.Min.Y),
		width:  C.int(bounds.Dx()),
		height: C.int(bounds.Dy()),
	}
	b.windowInfo.runtime_style = C.CEF_RUNTIME_STYLE_ALLOY

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
