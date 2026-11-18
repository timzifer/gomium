package cef

/*
#include "browser_process_handler_bridge.h"
*/
import "C"

//export goOnScheduleMessagePumpWork
func goOnScheduleMessagePumpWork(ptr *C.cef_browser_process_handler_t, cDelay C.int64_t) {
	c := getInstance()
	if c == nil || c.options.pumpWorker == nil {
		return
	}

	c.scheduleWork(int64(cDelay))
}

//export goOnContextInitialized
func goOnContextInitialized(ptr *C.cef_browser_process_handler_t) {
	c := getInstance()
	if c == nil {
		return
	}

	select {
	case c.contextInitializationFinished <- struct{}{}:
	default:
	}
}
