package cef

/*
#include <stdlib.h>
#include "binding_bridge.h"
#include "v8.h"
#include "memory.h"
*/
import "C"

import (
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"unsafe"

	"github.com/google/uuid"
)

// Bindings expose Go functions to page JavaScript as <root>.<name>.
//
// Lifecycle:
//   - WithBinding registers a function on the app. The option runs in both
//     processes, so the browser and the renderer agree on the list and on the
//     index that identifies each function.
//   - Each render-process V8 context gets the functions when it is created
//     (OnContextCreated), before any page script runs.
//   - A call from page JavaScript sends a request to the browser process. The
//     BindingGuard of the calling browser decides whether it may run. The
//     result returns to the renderer, which settles the promise of the call.
//   - Guard and function run on a goroutine of their own, never on a CEF
//     thread: a slow function must not stall the browser UI thread, and one
//     that itself waits for the UI thread (CookieManager, DevTools Call,
//     PrintToPdf) would otherwise deadlock until its timeout. Calls therefore
//     run concurrently - a bound function must be safe for that.
//
// Argument and result values travel as *Exported, the same wire form that
// JavaScript evaluation uses.

const (
	processMessageBindingCallRequest  = "pm.binding.Request"
	processMessageBindingCallResponse = "pm.binding.Response"

	bindingMaxDepth = 10
)

type (
	// BindingCall describes one call of a binding from page JavaScript, as the
	// browser process sees it when the call arrives.
	BindingCall struct {
		// Path is the binding's path, e.g. "app.getItems".
		Path string

		// FrameURL is the address the calling frame shows. nil when it does not
		// parse. For an iframe this is the iframe's own address.
		FrameURL *url.URL

		// MainFrame reports whether the call came from the top-level frame.
		MainFrame bool
	}

	// BindingGuard decides whether a binding call may run. false rejects it:
	// the function is not executed and the promise in the page is rejected.
	// The guard runs off the CEF threads, on the goroutine of the call, and
	// may be called concurrently. FrameURL and MainFrame are taken when the
	// call arrives, so a navigation in between does not change what the
	// guard sees.
	BindingGuard func(call BindingCall) bool

	bindingFunc struct {
		path string
		fn   reflect.Value
		typ  reflect.Type
	}

	bindingCallRequest struct {
		Id   uuid.UUID
		Path string
		Args []*Exported
	}

	bindingCallResponse struct {
		Id     uuid.UUID
		Result *Exported
		Error  string
	}

	pendingBindingCall struct {
		promise *C.cef_v8_value_t
		context *C.cef_v8_context_t
	}
)

var (
	errorInterface = reflect.TypeOf((*error)(nil)).Elem()

	// pendingBindingCalls holds calls that wait for the browser process. Each
	// entry owns one reference to its promise and to its context.
	pendingBindingCallsLock sync.Mutex
	pendingBindingCalls     = map[uuid.UUID]pendingBindingCall{}
)

func init() {
	_ = RegisterMessageIdentifier(processMessageBindingCallRequest, func(message ProcessMessage) (any, error) {
		val := &bindingCallRequest{}
		if err := decodeBinaryPayload(message, processMessagePayloadArgumentIndex, val); err != nil {
			return nil, err
		}
		return val, nil
	})
	_ = RegisterMessageIdentifier(processMessageBindingCallResponse, func(message ProcessMessage) (any, error) {
		val := &bindingCallResponse{}
		if err := decodeBinaryPayload(message, processMessagePayloadArgumentIndex, val); err != nil {
			return nil, err
		}
		return val, nil
	})
}

func (e bindingCallRequest) DispatchWith(dispatcher Dispatcher, target CefProcessId) (bool, error) {
	return dispatchGobPayload(processMessageBindingCallRequest, e, dispatcher, target)
}

func (e bindingCallResponse) DispatchWith(dispatcher Dispatcher, target CefProcessId) (bool, error) {
	return dispatchGobPayload(processMessageBindingCallResponse, e, dispatcher, target)
}

// WithBinding exposes fn to page JavaScript at path, e.g. "app.getItems".
//
// fn has at most one result besides an error: func(...) error, func(...) T or
// func(...) (T, error). A variadic fn takes the page's surplus arguments as
// its last parameter, so app.open("A", "B") reaches func(names ...string). Arguments arrive converted from JavaScript values
// (numbers as float64, objects as map[string]any, arrays as []any, null as the
// zero value). The result is returned to the page as a promise value.
//
// fn runs on a goroutine of its own, never on a CEF thread, and calls from
// the page may run concurrently.
//
// A bad path or signature is reported by New.
func WithBinding(path string, fn any) Option {
	return func(a *cefApp) {
		entry, err := newBindingFunc(path, fn)
		if err != nil {
			a.options.optionErrors = append(a.options.optionErrors, err)
			return
		}
		for _, existing := range a.options.bindings {
			if existing.path == path {
				a.options.optionErrors = append(a.options.optionErrors,
					fmt.Errorf("WithBinding(%q): registered twice", path))
				return
			}
		}
		a.options.bindings = append(a.options.bindings, entry)
	}
}

// WithBindingGuard sets the guard for the binding calls of this browser's
// frames. A browser without a guard of its own falls back to the default guard
// (WithDefaultBindingGuard). Without either, every binding call is rejected.
func WithBindingGuard(guard BindingGuard) ClientOption {
	return func(c *client) {
		c.bindingGuard = guard
	}
}

// WithDefaultBindingGuard sets the guard for every client without one of its
// own, popup browsers included. A popup that shares its opener's client gets
// the opener's guard.
func WithDefaultBindingGuard(guard BindingGuard) Option {
	return func(a *cefApp) {
		a.options.defaultBindingGuard = guard
	}
}

// bindingGuardFor picks the guard that decides a call of client c.
func bindingGuardFor(app *cefApp, c *client) BindingGuard {
	if c != nil && c.bindingGuard != nil {
		return c.bindingGuard
	}
	if app != nil {
		return app.options.defaultBindingGuard
	}
	return nil
}

// SameOrigin reports whether a and b share scheme, host and effective port.
// A nil address matches nothing.
func SameOrigin(a, b *url.URL) bool {
	if a == nil || b == nil || a.Host == "" || b.Host == "" {
		return false
	}
	return strings.EqualFold(a.Scheme, b.Scheme) &&
		strings.EqualFold(a.Hostname(), b.Hostname()) &&
		effectivePort(a) == effectivePort(b)
}

func effectivePort(u *url.URL) string {
	if port := u.Port(); port != "" {
		return port
	}
	switch strings.ToLower(u.Scheme) {
	case "https":
		return "443"
	case "http":
		return "80"
	}
	return ""
}

func newBindingFunc(path string, fn any) (bindingFunc, error) {
	if _, _, ok := splitBindingPath(path); !ok {
		return bindingFunc{}, fmt.Errorf("WithBinding(%q): the path must be <root>.<name>", path)
	}
	value := reflect.ValueOf(fn)
	if value.Kind() != reflect.Func || value.IsNil() {
		return bindingFunc{}, fmt.Errorf("WithBinding(%q): %T is not a function", path, fn)
	}
	typ := value.Type()
	switch typ.NumOut() {
	case 0:
	case 1:
	case 2:
		if typ.Out(1) != errorInterface {
			return bindingFunc{}, fmt.Errorf("WithBinding(%q): the second result must be error", path)
		}
	default:
		return bindingFunc{}, fmt.Errorf("WithBinding(%q): at most a result and an error, got %d results", path, typ.NumOut())
	}
	return bindingFunc{path: path, fn: value, typ: typ}, nil
}

// invoke converts the JavaScript arguments, runs the function and returns its
// result. A panic becomes an error.
func (b bindingFunc) invoke(args []*Exported) (result any, err error) {
	fixed := b.typ.NumIn()
	if b.typ.IsVariadic() {
		fixed--
		if len(args) < fixed {
			return nil, fmt.Errorf("%s takes at least %d arguments, got %d", b.path, fixed, len(args))
		}
	} else if len(args) != fixed {
		return nil, fmt.Errorf("%s takes %d arguments, got %d", b.path, fixed, len(args))
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			result, err = nil, fmt.Errorf("%s: panic: %v", b.path, recovered)
		}
	}()

	in := make([]reflect.Value, 0, fixed+1)
	for i := 0; i < fixed; i++ {
		value, convErr := b.argument(i, args[i], b.typ.In(i))
		if convErr != nil {
			return nil, convErr
		}
		in = append(in, value)
	}
	if b.typ.IsVariadic() {
		elem := b.typ.In(fixed).Elem()
		rest := reflect.MakeSlice(b.typ.In(fixed), 0, len(args)-fixed)
		for i := fixed; i < len(args); i++ {
			value, convErr := b.argument(i, args[i], elem)
			if convErr != nil {
				return nil, convErr
			}
			rest = reflect.Append(rest, value)
		}
		in = append(in, rest)
	}

	var out []reflect.Value
	if b.typ.IsVariadic() {
		out = b.fn.CallSlice(in)
	} else {
		out = b.fn.Call(in)
	}
	switch {
	case len(out) == 0:
		return nil, nil
	case b.typ.NumOut() == 1 && b.typ.Out(0) == errorInterface:
		if !out[0].IsNil() {
			return nil, out[0].Interface().(error)
		}
		return nil, nil
	case b.typ.NumOut() == 2:
		if !out[1].IsNil() {
			return nil, out[1].Interface().(error)
		}
		return out[0].Interface(), nil
	default:
		return out[0].Interface(), nil
	}
}

// argument converts the i-th JavaScript argument to want.
func (b bindingFunc) argument(i int, arg *Exported, want reflect.Type) (reflect.Value, error) {
	value := callArgument(b.typ, i, arg)
	switch {
	case value.Type().AssignableTo(want):
		return value, nil
	case value.Type().ConvertibleTo(want):
		return value.Convert(want), nil
	}
	return reflect.Value{}, fmt.Errorf("%s: argument %d is %s, want %s", b.path, i, value.Type(), want)
}

// splitBindingPath checks the <root>.<name> form and returns both parts.
func splitBindingPath(path string) (root, leaf string, ok bool) {
	root, leaf, ok = strings.Cut(path, ".")
	if !ok || root == "" || leaf == "" || strings.Contains(leaf, ".") {
		return "", "", false
	}
	return root, leaf, true
}

// installBindings runs in the render process when a V8 context is created. It
// attaches every binding to the context's global object, one root object per
// first path segment.
func installBindings(v8ctx *C.cef_v8_context_t, bindings []bindingFunc) {
	if len(bindings) == 0 {
		return
	}

	// V8 handles are only valid inside their context
	C.v8context_enter(v8ctx)
	defer C.v8context_exit(v8ctx)

	global := C.v8context_get_global(v8ctx)
	if global == nil {
		return
	}
	defer releaseBase(unsafe.Pointer(global))

	roots := map[string]*C.cef_v8_value_t{}
	defer func() {
		for _, root := range roots {
			releaseBase(unsafe.Pointer(root))
		}
	}()

	for index, entry := range bindings {
		root, leaf, _ := splitBindingPath(entry.path)

		object, known := roots[root]
		if !known {
			object = C.cef_v8_value_create_object(nil, nil)
			if object == nil {
				continue
			}
			// set_value_bykey consumes one reference: keep ours for the leaves
			addRefBase(unsafe.Pointer(object))
			rootKey := cefString(root)
			C.cBindingSetProperty(global, rootKey, object)
			cefStringFree(rootKey)
			roots[root] = object
		}

		leafKey := cefString(leaf)
		function := C.cBindingFunctionCreate(leafKey, C.int64_t(index))
		if function != nil {
			// consumed by set_value_bykey
			C.cBindingSetProperty(object, leafKey, function)
		}
		cefStringFree(leafKey)
	}
}

// goBindingExecute runs in the render process when page JavaScript calls a
// binding. It returns a promise and sends the call to the browser process.
//
//export goBindingExecute
func goBindingExecute(id int64, argc int64, argv unsafe.Pointer, retval unsafe.Pointer) C.int {
	app := getInstance()
	if app == nil || id < 0 || int(id) >= len(app.options.bindings) {
		return 0
	}
	entry := app.options.bindings[id]

	currentContext := v8Context{cStruct: C.cef_v8_context_get_current_context()}
	if currentContext.cStruct == nil {
		return 0
	}
	defer releaseBase(unsafe.Pointer(currentContext.cStruct))

	promise := C.cef_v8_value_create_promise()
	if promise == nil {
		return 0
	}

	callId := uuid.New()
	// one reference for the pending map, one for the caller via retval
	addRefBase(unsafe.Pointer(promise))
	// the context stays alive until the promise is settled
	addRefBase(unsafe.Pointer(currentContext.cStruct))
	pendingBindingCallsLock.Lock()
	pendingBindingCalls[callId] = pendingBindingCall{promise: promise, context: currentContext.cStruct}
	pendingBindingCallsLock.Unlock()

	*(**C.cef_v8_value_t)(retval) = promise

	args, err := exportArguments(argc, argv)
	if err != nil {
		rejectBindingCall(callId, err)
		return 1
	}

	// the call goes through the frame that made it: the browser's own
	// SendProcessMessage addresses the main frame, which would hide iframes
	// from the guard
	theFrame := currentContext.Frame()
	if theFrame == nil {
		rejectBindingCall(callId, errors.New("binding call without frame"))
		return 1
	}
	defer theFrame.Release()

	request := bindingCallRequest{Id: callId, Path: entry.path, Args: args}
	if sent, err := request.DispatchWith(theFrame, PidBrowser); err != nil || !sent {
		rejectBindingCall(callId, fmt.Errorf("binding call not dispatched: %v", err))
	}
	return 1
}

// exportArguments converts the V8 arguments of a call to their wire form.
func exportArguments(argc int64, argv unsafe.Pointer) ([]*Exported, error) {
	if argc == 0 {
		return nil, nil
	}
	pointers := unsafe.Slice((**C.cef_v8_value_t)(argv), int(argc))
	args := make([]*Exported, 0, len(pointers))
	for _, pointer := range pointers {
		// the value wrapper takes a reference of its own; the arguments are
		// borrowed - binding_execute releases CEF's reference after the call
		addRefBase(unsafe.Pointer(pointer))
		value, err := valueFromPointer(pointer) // releases it on error
		if err != nil {
			return nil, err
		}
		if value == nil {
			args = append(args, &Exported{Undefined: true})
			continue
		}
		exported, err := NewExporter(value).Export(bindingMaxDepth)
		_ = value.Close()
		if err != nil {
			return nil, err
		}
		args = append(args, exported)
	}
	return args, nil
}

// handleBindingCallRequest runs in the browser process. It applies the guard
// of the client, runs the function and answers to the renderer that asked.
// handleBindingCallRequest runs in the browser process, on the UI thread
// inside on_process_message_received. It takes only what has to be read
// there - the frame as it is now - and hands guard, function and answer to a
// goroutine.
func handleBindingCallRequest(c *client, request *bindingCallRequest, frame cefFrame) {
	app := getInstance()
	if app == nil {
		return
	}

	call := BindingCall{Path: request.Path, MainFrame: frame.IsMain()}
	if parsed, err := url.Parse(frame.url()); err == nil {
		call.FrameURL = parsed
	}

	// The caller releases its frame reference when the callback returns; the
	// goroutine answers through one of its own. Close releases it if the
	// call is still running then (#18).
	held := app.holdRef(unsafe.Pointer(frame.cStruct))
	if held == nil {
		return
	}
	go func() {
		response := runBindingCall(app, c, request, call)
		app.withCEF(func() {
			_, _ = response.DispatchWith(frame, PidRenderer)
			app.releaseHeld(held)
		})
	}()
}

// runBindingCall checks the guard and runs the function. It never touches
// CEF, so it is free to block.
func runBindingCall(app *cefApp, c *client, request *bindingCallRequest, call BindingCall) (response bindingCallResponse) {
	response.Id = request.Id

	entry, ok := bindingByPath(app, request.Path)
	if !ok {
		response.Error = "unknown binding " + request.Path
		return
	}

	guard := bindingGuardFor(app, c)
	if guard == nil || !guard(call) {
		if c != nil {
			c.logger.Warn().Str("binding", request.Path).Str("origin", originOf(call.FrameURL)).Msg("binding call rejected")
		}
		response.Error = "binding call not permitted: " + request.Path
		return
	}

	result, err := entry.invoke(request.Args)
	if err != nil {
		response.Error = err.Error()
		return
	}
	exported, err := exportedFromNative(result, bindingMaxDepth)
	if err != nil {
		response.Error = fmt.Sprintf("%s: result: %v", request.Path, err)
		return
	}
	response.Result = exported
	return
}

// originOf names only the origin of a URL for log lines: paths and queries can
// carry data.
func originOf(u *url.URL) string {
	if u == nil || u.Host == "" {
		return ""
	}
	return u.Scheme + "://" + u.Host
}

// handleBindingCallResponse runs in the render process and settles the
// promise of the call.
func handleBindingCallResponse(response *bindingCallResponse) {
	pending, ok := takeBindingCall(response.Id)
	if !ok {
		return
	}
	defer releaseBase(unsafe.Pointer(pending.promise))
	defer releaseBase(unsafe.Pointer(pending.context))

	// promises are settled inside their context
	C.v8context_enter(pending.context)
	defer C.v8context_exit(pending.context)

	if response.Error != "" {
		rejectPromise(pending.promise, response.Error)
		return
	}

	value, err := castValue(response.Result.Native())
	if err != nil {
		rejectPromise(pending.promise, err.Error())
		return
	}
	// refcheck:owned resolve consumes the reference to the value, like set_value_bykey does
	C.js_resolve_promise(pending.promise, value.native())
}

func rejectBindingCall(id uuid.UUID, err error) {
	pending, ok := takeBindingCall(id)
	if !ok {
		return
	}
	defer releaseBase(unsafe.Pointer(pending.promise))
	defer releaseBase(unsafe.Pointer(pending.context))

	C.v8context_enter(pending.context)
	defer C.v8context_exit(pending.context)

	rejectPromise(pending.promise, err.Error())
}

func rejectPromise(promise *C.cef_v8_value_t, message string) {
	text := cefString(message)
	C.js_reject_promise(promise, text)
	cefStringFree(text)
}

// rejectBindingCallsOf rejects and drops the pending binding calls made in
// context. Runs on the renderer main thread while context is still valid.
func rejectBindingCallsOf(context *C.cef_v8_context_t) {
	pendingBindingCallsLock.Lock()
	ids := make([]uuid.UUID, 0)
	for id, pending := range pendingBindingCalls {
		if C.v8context_is_same(pending.context, context) != 0 {
			ids = append(ids, id)
		}
	}
	pendingBindingCallsLock.Unlock()

	for _, id := range ids {
		rejectBindingCall(id, errors.New("binding call abandoned: the context was released"))
	}
}

func takeBindingCall(id uuid.UUID) (pendingBindingCall, bool) {
	pendingBindingCallsLock.Lock()
	defer pendingBindingCallsLock.Unlock()
	pending, ok := pendingBindingCalls[id]
	delete(pendingBindingCalls, id)
	return pending, ok
}

func bindingByPath(app *cefApp, path string) (bindingFunc, bool) {
	if app == nil {
		return bindingFunc{}, false
	}
	for _, entry := range app.options.bindings {
		if entry.path == path {
			return entry, true
		}
	}
	return bindingFunc{}, false
}
