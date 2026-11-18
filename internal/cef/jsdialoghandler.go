package cef

/*
#include <string.h>
#include "browser_client_bridge.h"
#include "jsdialoghandler_bridge.h"
*/
import "C"
import (
	"sync"
	"unsafe"
)

type (
	JsDialogType   int
	JsDialogAction int

	// JsDialogCallback finishes a dialog the handler took over. Continue must
	// be called exactly once; userInput is only meaningful for prompt
	// dialogs. Safe to call from any goroutine.
	JsDialogCallback interface {
		Continue(success bool, userInput string)
	}

	// JsDialogHandler receives JavaScript dialog requests (alert, confirm,
	// prompt, onbeforeunload). Both callbacks run synchronously on the
	// browser process UI thread - decide quickly and return. To show an own
	// (e.g. Fyne) dialog: capture callback, spawn a goroutine, return
	// JsDialogHandled and execute callback once the user decided.
	JsDialogHandler interface {
		// OnJsDialog decides how to treat an alert/confirm/prompt.
		OnJsDialog(instance Browser, originUrl string, dialogType JsDialogType, message, defaultPrompt string, callback JsDialogCallback) JsDialogAction
		// OnBeforeUnloadDialog decides the "leave site?" dialog.
		// JsDialogSuppressed auto-confirms leaving the page.
		OnBeforeUnloadDialog(instance Browser, message string, isReload bool, callback JsDialogCallback) JsDialogAction
	}

	jsDialogCallback struct {
		ptr  *C.cef_jsdialog_callback_t
		once *sync.Once
	}
)

const (
	JsDialogTypeAlert   = JsDialogType(C.JSDIALOGTYPE_ALERT)
	JsDialogTypeConfirm = JsDialogType(C.JSDIALOGTYPE_CONFIRM)
	JsDialogTypePrompt  = JsDialogType(C.JSDIALOGTYPE_PROMPT)
)

const (
	// JsDialogHandled: the handler shows an own dialog and will execute the
	// callback once it is dismissed.
	JsDialogHandled JsDialogAction = iota
	// JsDialogDefault leaves the dialog to CEF's default implementation.
	JsDialogDefault
	// JsDialogSuppressed drops the dialog silently. Preferable to an
	// immediate callback - CEF uses it to detect alert spamming.
	JsDialogSuppressed
)

func (c jsDialogCallback) Continue(success bool, userInput string) {
	c.once.Do(func() {
		defer releaseBase(unsafe.Pointer(c.ptr))

		cSuccess := C.int(0)
		if success {
			cSuccess = 1
		}
		var cInput *C.cef_string_t
		if userInput != "" {
			cInput = cefString(userInput)
			defer cefStringFree(cInput)
		}
		C.cJsDialogCallbackContinue(c.ptr, cSuccess, cInput)
	})
}

// discard marks the callback consumed and drops the reference - used when the
// handler did not take the dialog over, so a stray Continue from user code
// cannot touch freed memory.
func (c jsDialogCallback) discard() {
	c.once.Do(func() {
		releaseBase(unsafe.Pointer(c.ptr))
	})
}

func jsDialogHandler(handler *C.cef_jsdialog_handler_t, cefBrowser *C.cef_browser_t) (JsDialogHandler, Browser) {
	app := getInstance()
	if app == nil {
		return nil, nil
	}

	app.clientMutex.Lock()
	var found *client
	for _, c := range app.clients {
		c.Lock()
		match := c.jsDialogHandlerPtr == handler
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
	return found.jsDialogHandler, found.browserByIdentifier(int(C.cBridgeGetBrowserIdentifier(cefBrowser)))
}

//export goGetJsDialogHandler
func goGetJsDialogHandler(ptr *C.cef_client_t) *C.cef_jsdialog_handler_t {

	c := getClient(ptr)
	if c == nil {
		return nil
	}

	c.Lock()
	defer c.Unlock()

	if c.jsDialogHandlerPtr == nil || c.jsDialogHandler == nil {
		return nil
	}
	addRefBase(unsafe.Pointer(c.jsDialogHandlerPtr))
	return c.jsDialogHandlerPtr
}

//export goJsDialogHandlerOnJsDialog
func goJsDialogHandlerOnJsDialog(self *C.cef_jsdialog_handler_t,
	cefBrowser *C.cef_browser_t,
	originUrl *C.cef_string_t,
	dialogType C.cef_jsdialog_type_t,
	messageText *C.cef_string_t,
	defaultPromptText *C.cef_string_t,
	callback *C.cef_jsdialog_callback_t,
	suppressMessage *C.int) C.int {
	defer releaseBase(unsafe.Pointer(cefBrowser))

	*suppressMessage = 0
	wrapped := jsDialogCallback{ptr: callback, once: &sync.Once{}}

	handler, browserInstance := jsDialogHandler(self, cefBrowser)
	if handler == nil {
		wrapped.discard()
		return 0
	}

	action := handler.OnJsDialog(browserInstance,
		cefToGoString(originUrl), JsDialogType(dialogType),
		cefToGoString(messageText), cefToGoString(defaultPromptText), wrapped)

	switch action {
	case JsDialogHandled:
		// the callback keeps its reference until Continue runs
		return 1
	case JsDialogSuppressed:
		wrapped.discard()
		*suppressMessage = 1
		return 0
	default:
		wrapped.discard()
		return 0
	}
}

//export goJsDialogHandlerOnBeforeUnloadDialog
func goJsDialogHandlerOnBeforeUnloadDialog(self *C.cef_jsdialog_handler_t,
	cefBrowser *C.cef_browser_t,
	messageText *C.cef_string_t,
	isReload C.int,
	callback *C.cef_jsdialog_callback_t) C.int {
	defer releaseBase(unsafe.Pointer(cefBrowser))

	wrapped := jsDialogCallback{ptr: callback, once: &sync.Once{}}

	handler, browserInstance := jsDialogHandler(self, cefBrowser)
	if handler == nil {
		wrapped.discard()
		return 0
	}

	action := handler.OnBeforeUnloadDialog(browserInstance, cefToGoString(messageText), isReload != 0, wrapped)

	switch action {
	case JsDialogHandled:
		return 1
	case JsDialogSuppressed:
		// suppressing an unload dialog means leaving the page unasked
		wrapped.Continue(true, "")
		return 1
	default:
		wrapped.discard()
		return 0
	}
}
