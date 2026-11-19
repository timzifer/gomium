package cef

/*
#include "render_process_handler_bridge.h"
#include "browser_client_bridge.h"
#include "v8.h"
*/
import "C"
import (
	"fmt"
	"github.com/google/uuid"
	"log/slog"
	"sync"
	"unsafe"
)

var (
	v8contextLock = sync.Mutex{}
	v8contexts    = map[string]*v8Context{}
)

//export goOnWebKitInitialized
func goOnWebKitInitialized(ptr *C.cef_render_process_handler_t) {
	c := getInstance()
	if c == nil {
		return
	}

	c.onWebKitInitialized()
}

//export goOnRenderProcessContextCreated
func goOnRenderProcessContextCreated(ptr *C.cef_render_process_handler_t, cBrowser *C.cef_browser_t, cFrame *C.cef_frame_t, v8ctx *C.cef_v8_context_t) {
	theFrame := cefFrame{cFrame}
	frameIdentifier := theFrame.Identifier()

	// the context reference carried by the callback parameter is retained in
	// the map; it is released in goOnRenderProcessContextReleased
	theNewV8Context := &v8Context{v8ctx}

	v8contextLock.Lock()
	if previous, exists := v8contexts[frameIdentifier]; exists && previous != nil {
		releaseBase(unsafe.Pointer(previous.cStruct))
	}
	v8contexts[frameIdentifier] = theNewV8Context
	v8contextLock.Unlock()

	app := getInstance()
	if app == nil {
		releaseBase(unsafe.Pointer(cBrowser))
		releaseBase(unsafe.Pointer(cFrame))
		return
	}

	// bindings are in place before any page script of this context runs
	installBindings(v8ctx, app.options.bindings)

	// The listeners run right here, on the renderer main thread, before any
	// page script of the context: in the render process CEF accepts frame
	// and browser calls only on that thread, so a listener on a goroutine
	// could not even inject the script it is there for. Browser and frame
	// are released once the listeners returned.
	defer releaseBase(unsafe.Pointer(cBrowser))
	defer releaseBase(unsafe.Pointer(cFrame))
	app.v8ContextCreated(cefBrowser{cBrowser}, cefFrame{cFrame})
}

//export goOnRenderProcessContextReleased
func goOnRenderProcessContextReleased(ptr *C.cef_render_process_handler_t, cBrowser *C.cef_browser_t, cFrame *C.cef_frame_t, v8ctx *C.cef_v8_context_t) {
	defer releaseBase(unsafe.Pointer(v8ctx))

	frameIdentifier := cefFrame{cFrame}.Identifier()

	// A frame's new context can be created before its old one is released;
	// the entry then already holds the new context and must stay.
	v8contextLock.Lock()
	if stored, exists := v8contexts[frameIdentifier]; exists {
		if stored == nil {
			delete(v8contexts, frameIdentifier)
		} else if C.v8context_is_same(stored.cStruct, v8ctx) != 0 {
			delete(v8contexts, frameIdentifier)
			releaseBase(unsafe.Pointer(stored.cStruct))
		}
	}
	v8contextLock.Unlock()

	// binding calls of this context that the browser never answered would
	// hold their promise and the context forever
	rejectBindingCallsOf(v8ctx)

	app := getInstance()
	if app == nil {
		releaseBase(unsafe.Pointer(cBrowser))
		releaseBase(unsafe.Pointer(cFrame))
		return
	}

	// on the renderer main thread, like the created listeners
	defer releaseBase(unsafe.Pointer(cBrowser))
	defer releaseBase(unsafe.Pointer(cFrame))
	app.v8ContextReleased(cefBrowser{cBrowser}, cefFrame{cFrame})
}

//export goOnRenderProcessMessageReceived
func goOnRenderProcessMessageReceived(ptr *C.cef_render_process_handler_t, cefB *C.cef_browser_t, cFrame *C.cef_frame_t, sourceProcess C.cef_process_id_t, message *C.cef_process_message_t) C.int {
	defer releaseBase(unsafe.Pointer(cefB))
	defer releaseBase(unsafe.Pointer(cFrame))
	defer releaseBase(unsafe.Pointer(message))

	app := getInstance()
	if app == nil {
		return 0
	}
	logger := app.options.logger

	msg, err := getProcessMessageInstance(processMessage{message})
	if err != nil {
		logger.Warn("could not decode render-process message", "err", err)
		return 0
	}

	switch payload := msg.(type) {
	case *bindingCallResponse:
		handleBindingCallResponse(payload)
		return 1
	case *javaScriptEvaluationRequest:
		v8contextLock.Lock()
		ctx, exists := v8contexts[payload.Frame]
		knownFrames := make([]string, 0, len(v8contexts))
		for frameId := range v8contexts {
			knownFrames = append(knownFrames, frameId)
		}
		v8contextLock.Unlock()

		if !exists || ctx == nil {
			logger.Warn("no v8 context for frame, resolving evaluation with error", "frame", payload.Frame, "known-frames", knownFrames)
			dispatchEvaluationError(logger, cefB, payload.Id, "no v8 context for frame")
			return 1
		}

		result, exception := ctx.Eval(payload.Code, payload.Url, int(payload.Line))

		theBrowser := ctx.Browser()
		if theBrowser == nil {
			logger.Warn("v8 context has no browser, cannot dispatch evaluation result")
			return 1
		}

		response := javaScriptEvaluationResponse{Id: payload.Id, Result: result, Exception: exception}
		if success, dispatchErr := response.DispatchWith(theBrowser, PidBrowser); dispatchErr != nil {
			logger.Warn("could not dispatch evaluation response", "err", dispatchErr)
		} else if !success {
			logger.Warn("evaluation response was not dispatched")
		}
		theBrowser.Release()
		return 1
	default:
		logger.Debug(fmt.Sprintf("unhandled render-process message type %T", payload))
		return 0
	}
}

// dispatchEvaluationError sends an error response for an evaluation request
// using the browser the request arrived on.
func dispatchEvaluationError(logger *slog.Logger, cefB *C.cef_browser_t, id uuid.UUID, message string) {
	response := javaScriptEvaluationResponse{
		Id:        id,
		Exception: &ExceptionError{Message: message},
	}
	if success, err := response.DispatchWith(cefBrowser{cefB}, PidBrowser); err != nil || !success {
		logger.Warn("could not dispatch evaluation error response")
	}
}
