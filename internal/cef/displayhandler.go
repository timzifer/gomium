package cef

/*
#include <string.h>
#include "browser_client_bridge.h"
#include "displayhandler_bridge.h"
*/
import "C"
import (
	"net/url"
	"unsafe"
)

type (
	DisplayHandler interface {
		OnTitleChange(instance Browser, title string)
		OnAddressChange(instance Browser, url *url.URL)
		OnFaviconUrlChange(instance Browser, urls []*url.URL)
	}

	ConsoleMessageHandler interface {
		DisplayHandler
		OnConsoleMessage(instance Browser, source string, line int, message string)
	}
)

func displayHandler(handler *C.cef_display_handler_t, cefBrowser *C.cef_browser_t) (DisplayHandler, Browser) {
	app := getInstance()
	if app == nil {
		return nil, nil
	}

	app.clientMutex.Lock()
	var found *client
	for _, c := range app.clients {
		c.Lock()
		match := c.displayHandlerPtr == handler
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
	return found.displayHandler, found.browserByIdentifier(int(C.cBridgeGetBrowserIdentifier(cefBrowser)))
}

//export goGetDisplayHandler
func goGetDisplayHandler(ptr *C.cef_client_t) *C.cef_display_handler_t {

	c := getClient(ptr)
	if c == nil {
		return nil
	}

	c.Lock()
	defer c.Unlock()

	if c.displayHandlerPtr == nil {
		return nil
	}
	addRefBase(unsafe.Pointer(c.displayHandlerPtr))
	return c.displayHandlerPtr
}

//export goDisplayHandlerOnTitleChange
func goDisplayHandlerOnTitleChange(self *C.cef_display_handler_t,
	cefBrowser *C.cef_browser_t,
	title *C.cef_string_t) {
	defer releaseBase(unsafe.Pointer(cefBrowser))

	if handler, browserInstance := displayHandler(self, cefBrowser); handler != nil {
		go handler.OnTitleChange(browserInstance, cefToGoString(title))
	}
}

//export goDisplayHandlerOnAddressChange
func goDisplayHandlerOnAddressChange(self *C.cef_display_handler_t,
	cefBrowser *C.cef_browser_t,
	frame *C.cef_frame_t,
	urlString *C.cef_string_t) {
	defer releaseBase(unsafe.Pointer(cefBrowser))
	defer releaseBase(unsafe.Pointer(frame))

	if handler, browserInstance := displayHandler(self, cefBrowser); handler != nil {
		if theUrl, err := url.ParseRequestURI(cefToGoString(urlString)); err == nil {
			go handler.OnAddressChange(browserInstance, theUrl)
		}
	}
}

//export goDisplayHandlerOnFaviconUrlChange
func goDisplayHandlerOnFaviconUrlChange(self *C.cef_display_handler_t,
	cefBrowser *C.cef_browser_t,
	iconUrls C.cef_string_list_t) {
	defer releaseBase(unsafe.Pointer(cefBrowser))

	if handler, browserInstance := displayHandler(self, cefBrowser); handler != nil {
		var index C.size_t

		var urls []*url.URL
		for index = 0; index < C.cef_string_list_size(iconUrls); index++ {
			var str C.cef_string_t
			if C.cef_string_list_value(iconUrls, index, &str) == 0 {
				continue
			}
			theUrl, err := url.ParseRequestURI(cefToGoString(&str))
			// cef_string_list_value copies into str - release the buffer
			C.cef_string_clear(&str)
			if err == nil {
				urls = append(urls, theUrl)
			}
		}

		go handler.OnFaviconUrlChange(browserInstance, urls)
	}
}

//export goDisplayHandlerOnCursorChange
func goDisplayHandlerOnCursorChange(self *C.cef_display_handler_t,
	cefBrowser *C.cef_browser_t,
	cursorType C.cef_cursor_type_t) C.int {
	defer releaseBase(unsafe.Pointer(cefBrowser))

	// cursor changes arrive on the display handler since CEF 151, but the
	// gomium API keeps them on the render handler
	app := getInstance()
	if app == nil {
		return 0
	}

	app.clientMutex.Lock()
	var found *client
	for _, c := range app.clients {
		c.Lock()
		match := c.displayHandlerPtr == self
		c.Unlock()
		if match {
			found = c
			break
		}
	}
	app.clientMutex.Unlock()

	if found == nil {
		return 0
	}

	found.Lock()
	h := found.renderHandler
	found.Unlock()

	if h != nil {
		h.CursorChanged(CursorType(cursorType))
	}
	return 0
}

//export goDisplayHandlerOnConsoleMessage
func goDisplayHandlerOnConsoleMessage(self *C.cef_display_handler_t,
	cefBrowser *C.cef_browser_t,
	message *C.cef_string_t,
	source *C.cef_string_t,
	line C.int) C.int {
	defer releaseBase(unsafe.Pointer(cefBrowser))

	result := 0

	if handler, browserInstance := displayHandler(self, cefBrowser); handler != nil {
		if consoleHandler, isConsoleHandler := handler.(ConsoleMessageHandler); isConsoleHandler {
			result = 1
			go consoleHandler.OnConsoleMessage(browserInstance, cefToGoString(source), int(line), cefToGoString(message))
		}
	}
	return C.int(result)
}
