package cef

/*
#include <string.h>
#include "resourcehandler_bridge.h"
*/
import "C"
import (
	"net/url"
	"strings"
	"sync"
	"unsafe"
)

type (
	// ResourceContent is a complete in-memory response served for an
	// intercepted resource request.
	ResourceContent struct {
		Data []byte
		// MimeType is the media type, e.g. "text/html". A charset
		// parameter is allowed ("text/html; charset=utf-8") and is passed
		// on separately; other parameters are dropped, because CEF has no
		// place for them. Empty means application/octet-stream.
		MimeType string
		// Status is the HTTP status code; 0 means 200.
		Status int
	}

	// ResourceContentFn produces the response content. It runs on its own
	// goroutine and may block (downloads, rendering). Returning nil aborts
	// the request with ERR_ABORTED.
	ResourceContentFn func() *ResourceContent

	// ResourceInterceptor is optionally implemented by RequestHandler
	// implementations. InterceptResource is called on the CEF IO thread for
	// every resource request and must return quickly: either nil to let CEF
	// load the resource normally, or a ResourceContentFn that produces the
	// response. Unlike rewriting the request URL, serving content this way
	// is not a redirect - the original URL is preserved, so CSP and
	// same-origin checks are unaffected.
	ResourceInterceptor interface {
		InterceptResource(b Browser, requestUrl *url.URL, method string) ResourceContentFn
	}

	resourceServe struct {
		mu       sync.Mutex
		fn       ResourceContentFn
		content  *ResourceContent
		offset   int
		canceled bool
	}
)

var (
	resourceServeLock     sync.Mutex
	resourceServeRegistry = map[unsafe.Pointer]*resourceServe{}
)

func registerResourceServe(handler *C.cef_resource_handler_t, fn ResourceContentFn) {
	resourceServeLock.Lock()
	defer resourceServeLock.Unlock()
	resourceServeRegistry[unsafe.Pointer(handler)] = &resourceServe{fn: fn}
}

func lookupResourceServe(handler *C.cef_resource_handler_t) *resourceServe {
	resourceServeLock.Lock()
	defer resourceServeLock.Unlock()
	return resourceServeRegistry[unsafe.Pointer(handler)]
}

// forgetResourceServe drops the registry entry once the underlying struct is
// destroyed. Called from goMemoryRelease on final release; a no-op for
// pointers that are not resource handlers.
func forgetResourceServe(it unsafe.Pointer) {
	resourceServeLock.Lock()
	defer resourceServeLock.Unlock()
	delete(resourceServeRegistry, it)
}

//export goResourceHandlerOpen
func goResourceHandlerOpen(self *C.cef_resource_handler_t, request *C.cef_request_t, handleRequest *C.int, callback *C.cef_callback_t) C.int {
	defer releaseBase(unsafe.Pointer(request))

	s := lookupResourceServe(self)
	if s == nil {
		// no registry entry - cancel the request immediately
		*handleRequest = C.int(1)
		releaseBase(unsafe.Pointer(callback))
		return 0
	}

	// decide later: the content function may block, so it runs on its own
	// goroutine and continues via the callback (valid on any thread)
	*handleRequest = C.int(0)
	go func() {
		content := runResourceContentFn(s.fn)

		s.mu.Lock()
		s.content = content
		canceled := s.canceled
		s.mu.Unlock()

		if content == nil || canceled {
			C.cBridgeCallbackCancel(callback)
		} else {
			C.cBridgeCallbackCont(callback)
		}
		releaseBase(unsafe.Pointer(callback))
	}()
	return 1
}

// runResourceContentFn shields the process from panics in embedder code - a
// panic escaping into the cgo callback chain would abort the process.
func runResourceContentFn(fn ResourceContentFn) (content *ResourceContent) {
	defer func() {
		if r := recover(); r != nil {
			if app := getInstance(); app != nil {
				app.options.logger.Warn().Any("panic", r).Msg("resource content function panicked")
			}
			content = nil
		}
	}()
	if fn == nil {
		return nil
	}
	return fn()
}

//export goResourceHandlerGetResponseHeaders
func goResourceHandlerGetResponseHeaders(self *C.cef_resource_handler_t, response *C.cef_response_t, responseLength *C.int64_t) {
	defer releaseBase(unsafe.Pointer(response))

	s := lookupResourceServe(self)

	var content *ResourceContent
	if s != nil {
		s.mu.Lock()
		content = s.content
		s.mu.Unlock()
	}

	if content == nil {
		// should not happen - open only continues with content present
		*responseLength = 0
		return
	}

	// CEF wants the bare type in set_mime_type: a "text/html; charset=utf-8"
	// matches nothing, and Chromium then treats the response as an unknown
	// type - the page arrives with status 200 and stays blank. The charset
	// has its own setter, so a MimeType carrying one is split rather than
	// rejected.
	mimeType, charset := splitMediaType(content.MimeType)
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	status := content.Status
	if status == 0 {
		status = 200
	}

	mimeString := cefString(mimeType)
	// No charset given: leave CEF's own default alone rather than setting an
	// empty one.
	var charsetString *C.cef_string_t
	if charset != "" {
		charsetString = cefString(charset)
	}
	C.cBridgeResponseSet(response, mimeString, charsetString, C.int(status))
	cefStringFree(mimeString)
	if charsetString != nil {
		cefStringFree(charsetString)
	}

	*responseLength = C.int64_t(len(content.Data))
}

//export goResourceHandlerSkip
func goResourceHandlerSkip(self *C.cef_resource_handler_t, bytesToSkip C.int64_t, bytesSkipped *C.int64_t, callback *C.cef_resource_skip_callback_t) C.int {
	defer releaseBase(unsafe.Pointer(callback))

	s := lookupResourceServe(self)
	if s == nil {
		*bytesSkipped = -2 // ERR_FAILED
		return 0
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	remaining := 0
	if s.content != nil {
		remaining = len(s.content.Data) - s.offset
	}
	n := int(bytesToSkip)
	if n > remaining {
		n = remaining
	}
	s.offset += n
	*bytesSkipped = C.int64_t(n)
	return 1
}

//export goResourceHandlerRead
func goResourceHandlerRead(self *C.cef_resource_handler_t, dataOut unsafe.Pointer, bytesToRead C.int, bytesRead *C.int, callback *C.cef_resource_read_callback_t) C.int {
	defer releaseBase(unsafe.Pointer(callback))

	s := lookupResourceServe(self)
	if s == nil {
		*bytesRead = -2 // ERR_FAILED
		return 0
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.content == nil || s.offset >= len(s.content.Data) {
		*bytesRead = 0
		return 0 // response complete
	}

	n := len(s.content.Data) - s.offset
	if n > int(bytesToRead) {
		n = int(bytesToRead)
	}
	C.memcpy(dataOut, unsafe.Pointer(&s.content.Data[s.offset]), C.size_t(n))
	s.offset += n
	*bytesRead = C.int(n)
	return 1
}

//export goResourceHandlerCancel
func goResourceHandlerCancel(self *C.cef_resource_handler_t) {
	if s := lookupResourceServe(self); s != nil {
		s.mu.Lock()
		s.canceled = true
		s.mu.Unlock()
	}
}

// splitMediaType separates "text/html; charset=utf-8" into type and charset.
// Parameters other than charset are dropped - CEF has no place to put them.
func splitMediaType(value string) (mimeType string, charset string) {
	mimeType, parameters, found := strings.Cut(value, ";")
	mimeType = strings.TrimSpace(mimeType)
	if !found {
		return mimeType, ""
	}

	for _, parameter := range strings.Split(parameters, ";") {
		name, value, hasValue := strings.Cut(parameter, "=")
		if !hasValue || !strings.EqualFold(strings.TrimSpace(name), "charset") {
			continue
		}
		return mimeType, strings.Trim(strings.TrimSpace(value), `"`)
	}
	return mimeType, ""
}
