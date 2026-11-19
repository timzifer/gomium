package cef

/*
#include <string.h>
#include "browser_client_bridge.h"
#include "requesthandler_bridge.h"
*/
import "C"
import (
	"net/url"
	"sync"
	"unsafe"
)

type (
	resourceLoadCallback struct {
		mu  sync.Mutex
		cb  *C.cef_callback_t
		req *Request
	}
	RequestHandler interface {
		OnProtocolExecution(b Browser, url *url.URL) (stopLoad bool)
		OnBeforeBrowse(b Browser, url *url.URL) (stopLoad bool)

		OnBeforeResourceLoad(b Browser, request *Request, cb ResourceLoadAsyncResult) ResourceLoadResult
	}

	ResourceLoadAsyncResult interface {
		Continue(allow bool)
		Cancel()
	}
	ResourceLoadResult int
)

const (
	ResourceLoadContinue ResourceLoadResult = iota
	ResourceLoadContinueAsync
	ResourceLoadCancel
)

type (
	CertificateAction int

	// CertificateErrorCallback finishes an async certificate decision.
	// Continue must be called exactly once; safe from any goroutine.
	CertificateErrorCallback interface {
		Continue(allow bool)
	}

	// CertificateErrorHandler decides per request whether a load with an
	// invalid TLS certificate proceeds. certError is a net error code (e.g.
	// -202 ERR_CERT_AUTHORITY_INVALID). Runs synchronously on the browser
	// process UI thread - decide quickly or return CertificateDecideAsync
	// and finish via the callback. Never called when the app runs with
	// WithIgnoreCertificateErrors(true) - CEF accepts everything then.
	CertificateErrorHandler func(instance Browser, certError int, requestUrl string, callback CertificateErrorCallback) CertificateAction

	certErrorCallback struct {
		ptr  *C.cef_callback_t
		once *sync.Once
	}
)

const (
	// CertificateDeny cancels the request (the CEF default).
	CertificateDeny CertificateAction = iota
	// CertificateAccept continues the load despite the invalid certificate.
	CertificateAccept
	// CertificateDecideAsync defers to a later callback.Continue call.
	CertificateDecideAsync
)

func (c certErrorCallback) Continue(allow bool) {
	c.once.Do(func() {
		defer releaseBase(unsafe.Pointer(c.ptr))
		if allow {
			C.cBridgeContinueResourceLoad(c.ptr)
		} else {
			C.cBridgeCancelResourceLoad(c.ptr)
		}
	})
}

// discard marks the callback consumed and drops the reference without
// answering - used for the synchronous return paths.
func (c certErrorCallback) discard() {
	c.once.Do(func() {
		releaseBase(unsafe.Pointer(c.ptr))
	})
}

//export goOnCertificateError
func goOnCertificateError(self *C.cef_request_handler_t,
	cefBrowser *C.cef_browser_t,
	certError C.cef_errorcode_t,
	requestUrl *C.cef_string_t,
	sslInfo *C.struct__cef_sslinfo_t,
	callback *C.cef_callback_t) C.int {
	defer releaseBase(unsafe.Pointer(cefBrowser))
	defer releaseBase(unsafe.Pointer(sslInfo))

	wrapped := certErrorCallback{ptr: callback, once: &sync.Once{}}

	app := getInstance()
	if app == nil {
		wrapped.discard()
		return 0
	}

	app.clientMutex.Lock()
	var found *client
	for _, c := range app.clients {
		c.Lock()
		match := c.requestHandlerPtr == self
		c.Unlock()
		if match {
			found = c
			break
		}
	}
	app.clientMutex.Unlock()

	if found == nil {
		wrapped.discard()
		return 0
	}

	found.Lock()
	handler := found.certificateErrorHandler
	found.Unlock()
	if handler == nil {
		wrapped.discard()
		return 0
	}

	browserInstance := found.browserByIdentifier(int(C.cBridgeGetBrowserIdentifier(cefBrowser)))

	switch handler(browserInstance, int(certError), cefToGoString(requestUrl), wrapped) {
	case CertificateAccept:
		wrapped.Continue(true)
		return 1
	case CertificateDecideAsync:
		// the callback keeps its reference until Continue runs
		return 1
	default:
		wrapped.discard()
		return 0
	}
}

func (r *resourceLoadCallback) Continue(allow bool) {
	r.mu.Lock()
	if r.cb == nil {
		r.mu.Unlock()
		return
	}
	// cef_callback_t replaced cef_request_callback_t; cont() no longer takes
	// an allow flag - disallowing is now an explicit cancel
	if allow {
		C.cBridgeContinueResourceLoad(r.cb)
	} else {
		C.cBridgeCancelResourceLoad(r.cb)
	}
	releaseBase(unsafe.Pointer(r.cb))
	r.cb = nil
	req := r.req
	r.req = nil
	r.mu.Unlock()

	if req != nil {
		req.invalidateAndRelease()
	}
}

func (r *resourceLoadCallback) Cancel() {
	r.mu.Lock()
	if r.cb == nil {
		r.mu.Unlock()
		return
	}
	C.cBridgeCancelResourceLoad(r.cb)
	releaseBase(unsafe.Pointer(r.cb))
	r.cb = nil
	req := r.req
	r.req = nil
	r.mu.Unlock()

	if req != nil {
		req.invalidateAndRelease()
	}
}

func requestHandler(handler *C.cef_request_handler_t, cefBrowser *C.cef_browser_t) (RequestHandler, Browser) {
	return requestHandlerByMatch(cefBrowser, func(c *client) bool {
		return c.requestHandlerPtr == handler
	})
}

// resourceRequestHandler resolves the client owning a
// cef_resource_request_handler_t - resource callbacks arrive on that struct
// since CEF split it out of cef_request_handler_t.
func resourceRequestHandler(handler *C.cef_resource_request_handler_t, cefBrowser *C.cef_browser_t) (RequestHandler, Browser) {
	return requestHandlerByMatch(cefBrowser, func(c *client) bool {
		return c.resourceRequestHandlerPtr == handler
	})
}

func requestHandlerByMatch(cefBrowser *C.cef_browser_t, match func(*client) bool) (RequestHandler, Browser) {
	app := getInstance()
	if app == nil {
		return nil, nil
	}

	app.clientMutex.Lock()
	var found *client
	for _, c := range app.clients {
		c.Lock()
		matched := match(c)
		c.Unlock()
		if matched {
			found = c
			break
		}
	}
	app.clientMutex.Unlock()

	if found == nil {
		return nil, nil
	}
	return found.requestHandler, found.browserByIdentifier(int(C.cBridgeGetBrowserIdentifier(cefBrowser)))
}

//export goGetResourceRequestHandler
func goGetResourceRequestHandler(
	self *C.cef_request_handler_t,
	cefBrowser *C.cef_browser_t,
	frame *C.cef_frame_t,
	cefRequest *C.cef_request_t,
) *C.cef_resource_request_handler_t {
	defer releaseBase(unsafe.Pointer(cefBrowser))
	defer releaseBase(unsafe.Pointer(frame))
	defer releaseBase(unsafe.Pointer(cefRequest))

	app := getInstance()
	if app == nil {
		return nil
	}

	app.clientMutex.Lock()
	defer app.clientMutex.Unlock()

	for _, c := range app.clients {
		c.Lock()
		matched := c.requestHandlerPtr == self
		ptr := c.resourceRequestHandlerPtr
		c.Unlock()
		if matched {
			if ptr == nil {
				return nil
			}
			addRefBase(unsafe.Pointer(ptr))
			return ptr
		}
	}
	return nil
}

//export goGetRequestHandler
func goGetRequestHandler(
	cefClient *C.cef_client_t,
) *C.cef_request_handler_t {
	c := getClient(cefClient)
	if c == nil {
		return nil
	}

	c.Lock()
	defer c.Unlock()

	if c.requestHandlerPtr == nil {
		return nil
	}
	addRefBase(unsafe.Pointer(c.requestHandlerPtr))
	return c.requestHandlerPtr
}

//export goResourceRequestGetResourceHandler
func goResourceRequestGetResourceHandler(
	self *C.cef_resource_request_handler_t,
	cefBrowser *C.cef_browser_t,
	frame *C.cef_frame_t,
	cefRequest *C.cef_request_t,
) *C.cef_resource_handler_t {
	defer releaseBase(unsafe.Pointer(cefBrowser))
	defer releaseBase(unsafe.Pointer(frame))

	// the request wrapper owns the parameter reference; URL and method are
	// extracted here so the interceptor never touches the request after the
	// callback returns
	r := requestFrom(unsafe.Pointer(cefRequest))
	defer r.invalidateAndRelease()

	handler, browserInstance := resourceRequestHandler(self, cefBrowser)
	if handler == nil {
		return nil
	}
	interceptor, isInterceptor := handler.(ResourceInterceptor)
	if !isInterceptor {
		return nil
	}

	requestUrl, urlErr := r.GetUrl()
	if urlErr != nil {
		return nil
	}

	fn := interceptor.InterceptResource(browserInstance, requestUrl, r.GetMethod())
	if fn == nil {
		return nil
	}

	// the create reference transfers to CEF - lookup happens via the registry
	resourceHandler := C.cResourceHandlerCreate()
	registerResourceServe(resourceHandler, fn)
	return resourceHandler
}

//export goOnBeforeResourceLoad
func goOnBeforeResourceLoad(
	self *C.cef_resource_request_handler_t,
	cefBrowser *C.cef_browser_t,
	frame *C.cef_frame_t,
	cefRequest *C.cef_request_t,
	callback *C.cef_callback_t) (result C.cef_return_value_t) {
	defer releaseBase(unsafe.Pointer(cefBrowser))
	defer releaseBase(unsafe.Pointer(frame))

	// the request wrapper owns the reference carried by the parameter; it is
	// invalidated below (or by Continue/Cancel in the async case), so late
	// accesses from user code degrade safely instead of touching freed memory
	r := requestFrom(unsafe.Pointer(cefRequest))

	// callback and request are kept only for the async case; Continue or
	// Cancel release them
	async := false

	asyncResult := &resourceLoadCallback{
		cb:  callback,
		req: r,
	}
	if handler, browserInstance := resourceRequestHandler(self, cefBrowser); handler != nil {
		switch handler.OnBeforeResourceLoad(browserInstance, r, asyncResult) {
		case ResourceLoadContinue:
			result = C.RV_CONTINUE
		case ResourceLoadContinueAsync:
			result = C.RV_CONTINUE_ASYNC
			async = true
		case ResourceLoadCancel:
			result = C.RV_CANCEL
		}
	} else {
		result = C.RV_CONTINUE
	}

	if !async {
		// Disarm the callback the handler was given: a synchronous answer
		// means CEF ignores it, and a handler that kept it must find it
		// spent instead of calling into a released cef_callback_t later. A
		// handler that already called Continue/Cancel before returning has
		// released both itself - releasing again would be one too many.
		asyncResult.mu.Lock()
		cb, req := asyncResult.cb, asyncResult.req
		asyncResult.cb, asyncResult.req = nil, nil
		asyncResult.mu.Unlock()
		if cb != nil {
			releaseBase(unsafe.Pointer(cb))
		}
		if req != nil {
			req.invalidateAndRelease()
		}
	}

	return
}

//export goOnBeforeBrowse
func goOnBeforeBrowse(
	self *C.cef_request_handler_t,
	cefBrowser *C.cef_browser_t,
	frame *C.cef_frame_t,
	cefRequest *C.cef_request_t,
	isRedirect C.int,
) C.int {
	defer releaseBase(unsafe.Pointer(cefBrowser))
	defer releaseBase(unsafe.Pointer(frame))

	cefRequestInstance := requestFrom(unsafe.Pointer(cefRequest))
	defer cefRequestInstance.invalidateAndRelease()

	if requestUrl, err := cefRequestInstance.GetUrl(); err != nil {
		return C.int(0) // we could not parse the uri, allow since we might not have understood it
	} else {

		result := 0
		if handler, browserInstance := requestHandler(self, cefBrowser); handler != nil {
			if handler.OnBeforeBrowse(browserInstance, requestUrl) {
				result = 1
			} else {
				result = 0
			}
		}

		return C.int(result)
	}
}

//export goOnProtocolExecution
func goOnProtocolExecution(
	self *C.cef_resource_request_handler_t,
	cefBrowser *C.cef_browser_t,
	frame *C.cef_frame_t,
	cefRequest *C.cef_request_t,
	allowOsExecution *C.int,
) {
	defer releaseBase(unsafe.Pointer(cefBrowser))
	defer releaseBase(unsafe.Pointer(frame))

	// on_protocol_execution now hands over the request instead of a URL string
	cefRequestInstance := requestFrom(unsafe.Pointer(cefRequest))
	defer cefRequestInstance.invalidateAndRelease()

	if requestUrl, err := cefRequestInstance.GetUrl(); err != nil {
		// we could not parse the uri, do not allow
		C.cBridgeBrowserStopLoad(cefBrowser)
		*allowOsExecution = C.int(0)
		return
	} else {
		if handler, browserInstance := resourceRequestHandler(self, cefBrowser); handler != nil {
			if handler.OnProtocolExecution(browserInstance, requestUrl) {
				C.cBridgeBrowserStopLoad(cefBrowser)
				*allowOsExecution = C.int(0)
				return
			}
		}

		// no client found... forbid mailto just to be sure
		if requestUrl.Scheme == "mailto" {
			C.cBridgeBrowserStopLoad(cefBrowser)
			*allowOsExecution = C.int(0)
			return
		}
	}
	*allowOsExecution = C.int(1)
}

// cefToGoString converts a borrowed cef_string_t. It does NOT free the
// source - use cefUserfreeToGoString for cef_string_userfree_t results.
func cefToGoString(source *C.cef_string_t) string {
	if source == nil {
		return ""
	}
	utf8string := C.cBridgeCefSourceToString(source)
	defer C.cef_string_userfree_utf8_free(utf8string)
	if utf8string.str == nil {
		return ""
	}
	return C.GoString(utf8string.str)
}
