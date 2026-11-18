package cef

/*
#include <string.h>
#include "browser_client_bridge.h"
#include "focushandler_bridge.h"
*/
import "C"
import "unsafe"

type (
	// FocusSource tells where a focus change originated (cef_focus_source_t).
	FocusSource int

	// FocusHandler observes keyboard focus of the browser. Callbacks run on
	// the browser process UI thread - keep them short.
	//
	// Use it in OSR to hand focus back to the host: when OnTakeFocus fires
	// with next=false the user tabbed backwards out of the page, and the host
	// should move focus to its previous widget.
	FocusHandler interface {
		// OnTakeFocus fires when the browser gives up focus (Tab past the last
		// or before the first element). next is true for forward Tab.
		OnTakeFocus(instance Browser, next bool)
		// OnSetFocus fires before the browser takes focus. Return true to
		// suppress the focus change, false to accept it.
		OnSetFocus(instance Browser, source FocusSource) bool
		// OnGotFocus fires after the browser received focus.
		OnGotFocus(instance Browser)
	}
)

const (
	FocusSourceNavigation = FocusSource(C.FOCUS_SOURCE_NAVIGATION)
	FocusSourceSystem     = FocusSource(C.FOCUS_SOURCE_SYSTEM)
)

func focusHandlerFor(self *C.cef_focus_handler_t, cefBrowser *C.cef_browser_t) (FocusHandler, Browser) {
	c := clientWhere(func(c *client) bool { return c.focusHandlerPtr == self })
	if c == nil || c.focusHandler == nil {
		return nil, nil
	}
	return c.focusHandler, c.browserByIdentifier(int(C.cBridgeGetBrowserIdentifier(cefBrowser)))
}

//export goGetFocusHandler
func goGetFocusHandler(ptr *C.cef_client_t) *C.cef_focus_handler_t {
	c := getClient(ptr)
	if c == nil {
		return nil
	}

	c.Lock()
	defer c.Unlock()

	if c.focusHandlerPtr == nil || c.focusHandler == nil {
		return nil
	}
	addRefBase(unsafe.Pointer(c.focusHandlerPtr))
	return c.focusHandlerPtr
}

//export goFocusHandlerOnTakeFocus
func goFocusHandlerOnTakeFocus(self *C.cef_focus_handler_t, cefBrowser *C.cef_browser_t, next C.int) {
	defer releaseBase(unsafe.Pointer(cefBrowser))

	if h, b := focusHandlerFor(self, cefBrowser); h != nil {
		h.OnTakeFocus(b, next != 0)
	}
}

//export goFocusHandlerOnSetFocus
func goFocusHandlerOnSetFocus(self *C.cef_focus_handler_t, cefBrowser *C.cef_browser_t, source C.cef_focus_source_t) C.int {
	defer releaseBase(unsafe.Pointer(cefBrowser))

	if h, b := focusHandlerFor(self, cefBrowser); h != nil {
		if h.OnSetFocus(b, FocusSource(source)) {
			return 1
		}
	}
	return 0
}

//export goFocusHandlerOnGotFocus
func goFocusHandlerOnGotFocus(self *C.cef_focus_handler_t, cefBrowser *C.cef_browser_t) {
	defer releaseBase(unsafe.Pointer(cefBrowser))

	if h, b := focusHandlerFor(self, cefBrowser); h != nil {
		h.OnGotFocus(b)
	}
}
