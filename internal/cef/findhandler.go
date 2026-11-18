package cef

/*
#include <string.h>
#include "browser_client_bridge.h"
#include "findhandler_bridge.h"
*/
import "C"
import "unsafe"

type (
	// FindResult reports the state of a Browser.Find search. Multiple
	// updates may arrive per search; FinalUpdate marks the last one.
	FindResult struct {
		Identifier         int
		Count              int
		ActiveMatchOrdinal int
		FinalUpdate        bool
	}

	// FindHandler receives find-in-page results. Dispatched on its own
	// goroutine.
	FindHandler interface {
		OnFindResult(instance Browser, result FindResult)
	}

	findHandlerFunc func(Browser, FindResult)
)

func (f findHandlerFunc) OnFindResult(bi Browser, result FindResult) { f(bi, result) }

//export goGetFindHandler
func goGetFindHandler(ptr *C.cef_client_t) *C.cef_find_handler_t {

	c := getClient(ptr)
	if c == nil {
		return nil
	}

	c.Lock()
	defer c.Unlock()

	if c.findHandlerPtr == nil || c.findHandler == nil {
		return nil
	}
	addRefBase(unsafe.Pointer(c.findHandlerPtr))
	return c.findHandlerPtr
}

//export goFindHandlerOnFindResult
func goFindHandlerOnFindResult(self *C.cef_find_handler_t,
	cefBrowser *C.cef_browser_t,
	identifier C.int,
	count C.int,
	activeMatchOrdinal C.int,
	finalUpdate C.int) {
	defer releaseBase(unsafe.Pointer(cefBrowser))

	app := getInstance()
	if app == nil {
		return
	}

	app.clientMutex.Lock()
	var found *client
	for _, c := range app.clients {
		c.Lock()
		match := c.findHandlerPtr == self
		c.Unlock()
		if match {
			found = c
			break
		}
	}
	app.clientMutex.Unlock()

	if found == nil {
		return
	}

	found.Lock()
	handler := found.findHandler
	found.Unlock()

	if handler != nil {
		browserInstance := found.browserByIdentifier(int(C.cBridgeGetBrowserIdentifier(cefBrowser)))
		go handler.OnFindResult(browserInstance, FindResult{
			Identifier:         int(identifier),
			Count:              int(count),
			ActiveMatchOrdinal: int(activeMatchOrdinal),
			FinalUpdate:        finalUpdate != 0,
		})
	}
}
