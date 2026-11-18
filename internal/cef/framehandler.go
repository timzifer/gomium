package cef

/*
#include <string.h>
#include "browser_client_bridge.h"
#include "framehandler_bridge.h"
*/
import "C"
import "unsafe"

type (
	// FrameHandler observes frame lifecycle (iframes, main frame swaps).
	// Callbacks run on the browser process UI thread.
	//
	// The CefFrame arguments are valid only for the duration of the call.
	// Retain nothing; to use a frame later, keep its identifier and look it
	// up again.
	FrameHandler interface {
		OnFrameCreated(instance Browser, frame CefFrame)
		OnFrameDestroyed(instance Browser, frame CefFrame)
		OnFrameAttached(instance Browser, frame CefFrame, reattached bool)
		OnFrameDetached(instance Browser, frame CefFrame)
		OnMainFrameChanged(instance Browser, oldFrame, newFrame CefFrame)
	}
)

func frameHandlerFor(self *C.cef_frame_handler_t, cefBrowser *C.cef_browser_t) (FrameHandler, Browser) {
	c := clientWhere(func(c *client) bool { return c.frameHandlerPtr == self })
	if c == nil || c.frameHandler == nil {
		return nil, nil
	}
	return c.frameHandler, c.browserByIdentifier(int(C.cBridgeGetBrowserIdentifier(cefBrowser)))
}

//export goGetFrameHandler
func goGetFrameHandler(ptr *C.cef_client_t) *C.cef_frame_handler_t {
	c := getClient(ptr)
	if c == nil {
		return nil
	}

	c.Lock()
	defer c.Unlock()

	if c.frameHandlerPtr == nil || c.frameHandler == nil {
		return nil
	}
	addRefBase(unsafe.Pointer(c.frameHandlerPtr))
	return c.frameHandlerPtr
}

//export goFrameHandlerOnFrameCreated
func goFrameHandlerOnFrameCreated(self *C.cef_frame_handler_t, cefBrowser *C.cef_browser_t, frame *C.cef_frame_t) {
	defer releaseBase(unsafe.Pointer(cefBrowser))
	defer releaseBase(unsafe.Pointer(frame))

	if h, b := frameHandlerFor(self, cefBrowser); h != nil {
		h.OnFrameCreated(b, cefFrame{frame})
	}
}

//export goFrameHandlerOnFrameDestroyed
func goFrameHandlerOnFrameDestroyed(self *C.cef_frame_handler_t, cefBrowser *C.cef_browser_t, frame *C.cef_frame_t) {
	defer releaseBase(unsafe.Pointer(cefBrowser))
	defer releaseBase(unsafe.Pointer(frame))

	if h, b := frameHandlerFor(self, cefBrowser); h != nil {
		h.OnFrameDestroyed(b, cefFrame{frame})
	}
}

//export goFrameHandlerOnFrameAttached
func goFrameHandlerOnFrameAttached(self *C.cef_frame_handler_t, cefBrowser *C.cef_browser_t, frame *C.cef_frame_t, reattached C.int) {
	defer releaseBase(unsafe.Pointer(cefBrowser))
	defer releaseBase(unsafe.Pointer(frame))

	if h, b := frameHandlerFor(self, cefBrowser); h != nil {
		h.OnFrameAttached(b, cefFrame{frame}, reattached != 0)
	}
}

//export goFrameHandlerOnFrameDetached
func goFrameHandlerOnFrameDetached(self *C.cef_frame_handler_t, cefBrowser *C.cef_browser_t, frame *C.cef_frame_t) {
	defer releaseBase(unsafe.Pointer(cefBrowser))
	defer releaseBase(unsafe.Pointer(frame))

	if h, b := frameHandlerFor(self, cefBrowser); h != nil {
		h.OnFrameDetached(b, cefFrame{frame})
	}
}

//export goFrameHandlerOnMainFrameChanged
func goFrameHandlerOnMainFrameChanged(self *C.cef_frame_handler_t, cefBrowser *C.cef_browser_t, oldFrame, newFrame *C.cef_frame_t) {
	defer releaseBase(unsafe.Pointer(cefBrowser))
	defer releaseBase(unsafe.Pointer(oldFrame))
	defer releaseBase(unsafe.Pointer(newFrame))

	var o, n CefFrame
	if oldFrame != nil {
		o = cefFrame{oldFrame}
	}
	if newFrame != nil {
		n = cefFrame{newFrame}
	}
	if h, b := frameHandlerFor(self, cefBrowser); h != nil {
		h.OnMainFrameChanged(b, o, n)
	}
}
