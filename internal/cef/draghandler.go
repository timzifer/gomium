package cef

/*
#include <string.h>
#include "browser_client_bridge.h"
#include "draghandler_bridge.h"
*/
import "C"
import "unsafe"

type (
	// DragHandler is told about drag operations that enter the browser
	// window from outside the browser. Called synchronously on the browser
	// process UI thread, because its return value decides whether the drag
	// continues - a handler that blocks freezes the drag.
	//
	// The callback fires when the pointer ENTERS the window, not when the
	// files are released. It is therefore a source of paths, not a signal
	// that something was dropped: the drop itself arrives in the page as an
	// ordinary HTML5 drop event. A page that wants both has to hold on to
	// the paths until its own drop handler runs.
	DragHandler interface {
		// OnDragEnter reports the native paths of a file drag. Returning
		// true cancels the drag, which means the page never sees a drop
		// event either.
		OnDragEnter(instance Browser, files []string) bool
	}

	dragHandlerFunc func(Browser, []string) bool
)

func (d dragHandlerFunc) OnDragEnter(bi Browser, files []string) bool { return d(bi, files) }

//export goGetDragHandler
func goGetDragHandler(ptr *C.cef_client_t) *C.cef_drag_handler_t {

	c := getClient(ptr)
	if c == nil {
		return nil
	}

	c.Lock()
	defer c.Unlock()

	if c.dragHandlerPtr == nil || c.dragHandler == nil {
		return nil
	}
	addRefBase(unsafe.Pointer(c.dragHandlerPtr))
	return c.dragHandlerPtr
}

//export goDragHandlerOnDragEnter
func goDragHandlerOnDragEnter(self *C.cef_drag_handler_t,
	cefBrowser *C.cef_browser_t,
	dragData *C.cef_drag_data_t,
	mask C.int) C.int {
	defer releaseBase(unsafe.Pointer(cefBrowser))
	defer releaseBase(unsafe.Pointer(dragData))

	app := getInstance()
	if app == nil {
		return 0
	}

	app.clientMutex.Lock()
	var found *client
	for _, c := range app.clients {
		c.Lock()
		match := c.dragHandlerPtr == self
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
	handler := found.dragHandler
	found.Unlock()

	if handler == nil {
		return 0
	}

	// The paths have to be read here and not in a goroutine: dragData is
	// released as soon as this function returns.
	files := dragDataFileNames(dragData)

	browserInstance := found.browserByIdentifier(int(C.cBridgeGetBrowserIdentifier(cefBrowser)))
	if handler.OnDragEnter(browserInstance, files) {
		return 1
	}
	return 0
}

// dragDataFileNames reads the native paths out of a file drag. It returns nil
// for every other kind of drag (text, links, images).
func dragDataFileNames(dragData *C.cef_drag_data_t) []string {
	if dragData == nil || C.cBridgeDragDataIsFile(dragData) == 0 {
		return nil
	}

	list := C.cef_string_list_alloc()
	if list == nil {
		return nil
	}
	defer C.cef_string_list_free(list)

	if C.cBridgeDragDataFileNames(dragData, list) == 0 {
		return nil
	}

	count := int(C.cef_string_list_size(list))
	files := make([]string, 0, count)
	for i := 0; i < count; i++ {
		var value C.cef_string_t
		if C.cef_string_list_value(list, C.size_t(i), &value) == 0 {
			continue
		}
		files = append(files, cefToGoString(&value))
		C.cef_string_clear(&value)
	}
	return files
}
