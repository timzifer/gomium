package cef

/*
#include "devtools_bridge.h"
*/
import "C"
import "unsafe"

// The browser parameter of every observer callback is a reference the callee
// owns, so each callback releases it (same as the load handler callbacks).

//export goDevToolsMessage
func goDevToolsMessage(self *C.cef_dev_tools_message_observer_t,
	cefBrowser *C.cef_browser_t,
	message unsafe.Pointer,
	messageSize C.size_t) C.int {
	defer releaseBase(unsafe.Pointer(cefBrowser))

	_, s, _ := devToolsLookup(self, cefBrowser)
	if s == nil {
		return 0
	}
	// the message is only valid during the callback - copy it
	s.handleMessage(C.GoBytes(message, C.int(messageSize)))
	return 1
}

//export goDevToolsAgentAttached
func goDevToolsAgentAttached(self *C.cef_dev_tools_message_observer_t, cefBrowser *C.cef_browser_t) {
	defer releaseBase(unsafe.Pointer(cefBrowser))

	receiver, s, instance := devToolsLookup(self, cefBrowser)
	if s == nil || receiver == nil {
		return
	}
	go receiver(instance, s)
}

//export goDevToolsAgentDetached
func goDevToolsAgentDetached(self *C.cef_dev_tools_message_observer_t, cefBrowser *C.cef_browser_t) {
	defer releaseBase(unsafe.Pointer(cefBrowser))

	if _, s, _ := devToolsLookup(self, cefBrowser); s != nil {
		s.onDetached()
	}
}
