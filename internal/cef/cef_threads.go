package cef

/*
#include "include/internal/cef_types.h"
#include "include/capi/cef_task_capi.h"


*/
import "C"

type (
	CefThreadId  C.cef_thread_id_t
	CefProcessId C.cef_process_id_t
)

const (
	PidBrowser  CefProcessId = C.PID_BROWSER
	PidRenderer CefProcessId = C.PID_RENDERER

	TidUI               CefThreadId = C.TID_UI
	TidFileBackground   CefThreadId = C.TID_FILE_BACKGROUND
	TidFileUserVisible  CefThreadId = C.TID_FILE_USER_VISIBLE
	TidFileUserBlocking CefThreadId = C.TID_FILE_USER_BLOCKING
	TidProcessLauncher  CefThreadId = C.TID_PROCESS_LAUNCHER
	TidIO               CefThreadId = C.TID_IO
	TidRenderer         CefThreadId = C.TID_RENDERER

	// TidFile is kept as an alias for the closest CEF 151 equivalent of the
	// former TID_FILE thread. TID_DB and TID_CACHE no longer exist.
	TidFile CefThreadId = C.TID_FILE_BACKGROUND
)

func CefCurrentlyOn(id CefThreadId) bool {
	return C.cef_currently_on(C.cef_thread_id_t(id)) == 1
}
