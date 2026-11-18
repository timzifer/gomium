package cef

/*
#include <string.h>
#include "browser_client_bridge.h"
#include "loadhandler_bridge.h"
*/
import "C"
import "unsafe"

// LoadHandler receives browser load progress events. Callbacks originate on
// the browser process UI thread and are dispatched on their own goroutines.
type LoadHandler interface {
	// OnLoadingStateChange fires once when loading starts and once when it
	// ends (completion, cancellation or failure). It brackets all
	// OnLoadStart/OnLoadEnd/OnLoadError calls and carries the current
	// navigation state.
	OnLoadingStateChange(instance Browser, isLoading, canGoBack, canGoForward bool)
	// OnLoadStart fires after a navigation committed, before content loads.
	// Not called for same-page navigations (fragments, history state).
	OnLoadStart(instance Browser, isMainFrame bool)
	// OnLoadEnd fires when a frame finished loading.
	OnLoadEnd(instance Browser, isMainFrame bool, httpStatusCode int)
	// OnLoadError fires when a navigation fails or is canceled. errorCode is
	// a net error code from net/base/net_error_list.h.
	OnLoadError(instance Browser, isMainFrame bool, errorCode int, errorText string, failedUrl string)
}

func loadHandler(handler *C.cef_load_handler_t, cefBrowser *C.cef_browser_t) (LoadHandler, Browser) {
	app := getInstance()
	if app == nil {
		return nil, nil
	}

	app.clientMutex.Lock()
	var found *client
	for _, c := range app.clients {
		c.Lock()
		match := c.loadHandlerPtr == handler
		c.Unlock()
		if match {
			found = c
			break
		}
	}
	app.clientMutex.Unlock()

	if found == nil {
		return nil, nil
	}
	// the browser map is guarded by the client mutex, not the clientMutex
	return found.loadHandler, found.browserByIdentifier(int(C.cBridgeGetBrowserIdentifier(cefBrowser)))
}

//export goGetLoadHandler
func goGetLoadHandler(ptr *C.cef_client_t) *C.cef_load_handler_t {

	c := getClient(ptr)
	if c == nil {
		return nil
	}

	c.Lock()
	defer c.Unlock()

	if c.loadHandlerPtr == nil || c.loadHandler == nil {
		return nil
	}
	addRefBase(unsafe.Pointer(c.loadHandlerPtr))
	return c.loadHandlerPtr
}

//export goLoadHandlerOnLoadingStateChange
func goLoadHandlerOnLoadingStateChange(self *C.cef_load_handler_t,
	cefBrowser *C.cef_browser_t,
	isLoading C.int,
	canGoBack C.int,
	canGoForward C.int) {
	defer releaseBase(unsafe.Pointer(cefBrowser))

	if handler, browserInstance := loadHandler(self, cefBrowser); handler != nil {
		go handler.OnLoadingStateChange(browserInstance, isLoading != 0, canGoBack != 0, canGoForward != 0)
	}
}

//export goLoadHandlerOnLoadStart
func goLoadHandlerOnLoadStart(self *C.cef_load_handler_t,
	cefBrowser *C.cef_browser_t,
	frame *C.cef_frame_t) {
	defer releaseBase(unsafe.Pointer(cefBrowser))

	theFrame := cefFrame{frame}
	isMain := theFrame.IsMain()
	theFrame.Release()

	if handler, browserInstance := loadHandler(self, cefBrowser); handler != nil {
		go handler.OnLoadStart(browserInstance, isMain)
	}
}

//export goLoadHandlerOnLoadEnd
func goLoadHandlerOnLoadEnd(self *C.cef_load_handler_t,
	cefBrowser *C.cef_browser_t,
	frame *C.cef_frame_t,
	httpStatusCode C.int) {
	defer releaseBase(unsafe.Pointer(cefBrowser))

	theFrame := cefFrame{frame}
	isMain := theFrame.IsMain()
	theFrame.Release()

	if handler, browserInstance := loadHandler(self, cefBrowser); handler != nil {
		go handler.OnLoadEnd(browserInstance, isMain, int(httpStatusCode))
	}
}

//export goLoadHandlerOnLoadError
func goLoadHandlerOnLoadError(self *C.cef_load_handler_t,
	cefBrowser *C.cef_browser_t,
	frame *C.cef_frame_t,
	errorCode C.cef_errorcode_t,
	errorText *C.cef_string_t,
	failedUrl *C.cef_string_t) {
	defer releaseBase(unsafe.Pointer(cefBrowser))

	theFrame := cefFrame{frame}
	isMain := theFrame.IsMain()
	theFrame.Release()

	if handler, browserInstance := loadHandler(self, cefBrowser); handler != nil {
		go handler.OnLoadError(browserInstance, isMain, int(errorCode), cefToGoString(errorText), cefToGoString(failedUrl))
	}
}
