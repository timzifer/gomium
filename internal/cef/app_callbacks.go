package cef

/*
   #include "app_bridge.h"
*/
import "C"
import "unsafe"

// The get_*_handler callbacks transfer an owned reference to CEF, so each
// call must add a reference before returning the struct.

//export goGetRenderProcessHandler
func goGetRenderProcessHandler(ptr *C.cef_app_t) *C.cef_render_process_handler_t {
	c := getInstance()
	if c == nil || c.renderProcessHandler == nil {
		return nil
	}
	addRefBase(unsafe.Pointer(c.renderProcessHandler))
	return c.renderProcessHandler
}

//export goGetBrowserProcessHandler
func goGetBrowserProcessHandler(ptr *C.cef_app_t) *C.cef_browser_process_handler_t {
	c := getInstance()
	if c == nil || c.browserProcessHandler == nil {
		return nil
	}
	addRefBase(unsafe.Pointer(c.browserProcessHandler))
	return c.browserProcessHandler
}
