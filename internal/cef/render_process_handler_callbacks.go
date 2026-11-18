package cef

/*
#include "render_process_handler_bridge.h"
#include "browser_client_bridge.h"
*/
import "C"
import (
	"github.com/google/uuid"
	"github.com/rs/zerolog"
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

	// browser and frame references stay alive until the listeners returned
	go func() {
		defer releaseBase(unsafe.Pointer(cBrowser))
		defer releaseBase(unsafe.Pointer(cFrame))
		app.v8ContextCreated(cefBrowser{cBrowser}, cefFrame{cFrame})
	}()
}

//export goOnRenderProcessContextReleased
func goOnRenderProcessContextReleased(ptr *C.cef_render_process_handler_t, cBrowser *C.cef_browser_t, cFrame *C.cef_frame_t, v8ctx *C.cef_v8_context_t) {
	defer releaseBase(unsafe.Pointer(v8ctx))

	frameIdentifier := cefFrame{cFrame}.Identifier()

	v8contextLock.Lock()
	if stored, exists := v8contexts[frameIdentifier]; exists {
		delete(v8contexts, frameIdentifier)
		if stored != nil {
			releaseBase(unsafe.Pointer(stored.cStruct))
		}
	}
	v8contextLock.Unlock()

	app := getInstance()
	if app == nil {
		releaseBase(unsafe.Pointer(cBrowser))
		releaseBase(unsafe.Pointer(cFrame))
		return
	}

	go func() {
		defer releaseBase(unsafe.Pointer(cBrowser))
		defer releaseBase(unsafe.Pointer(cFrame))
		app.v8ContextReleased(cefBrowser{cBrowser}, cefFrame{cFrame})
	}()
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
		logger.Warn().Err(err).Msg("could not decode render-process message")
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
			logger.Warn().Str("frame", payload.Frame).Strs("known-frames", knownFrames).Msg("no v8 context for frame, resolving evaluation with error")
			dispatchEvaluationError(logger, cefB, payload.Id, "no v8 context for frame")
			return 1
		}

		result, exception := ctx.Eval(payload.Code, payload.Url, int(payload.Line))

		theBrowser := ctx.Browser()
		if theBrowser == nil {
			logger.Warn().Msg("v8 context has no browser, cannot dispatch evaluation result")
			return 1
		}

		response := javaScriptEvaluationResponse{Id: payload.Id, Result: result, Exception: exception}
		if success, dispatchErr := response.DispatchWith(theBrowser, PidBrowser); dispatchErr != nil {
			logger.Warn().Err(dispatchErr).Msg("could not dispatch evaluation response")
		} else if !success {
			logger.Warn().Msg("evaluation response was not dispatched")
		}
		theBrowser.Release()
		return 1
	default:
		logger.Debug().Msgf("unhandled render-process message type %T", payload)
		return 0
	}
}

// dispatchEvaluationError sends an error response for an evaluation request
// using the browser the request arrived on.
func dispatchEvaluationError(logger zerolog.Logger, cefB *C.cef_browser_t, id uuid.UUID, message string) {
	response := javaScriptEvaluationResponse{
		Id:        id,
		Exception: &ExceptionError{Message: message},
	}
	if success, err := response.DispatchWith(cefBrowser{cefB}, PidBrowser); err != nil || !success {
		logger.Warn().Msg("could not dispatch evaluation error response")
	}
}
