package cef

/*
#include <string.h>
#include "request_bridge.h"
*/
import "C"
import (
	"errors"
	"net/url"
	"sync"
	"unsafe"
)

// Request wraps a cef_request_t. It is only valid during the callback it was
// handed to (for OnBeforeResourceLoad with ResourceLoadContinueAsync: until
// Continue or Cancel is called). After that every method degrades to a safe
// zero value instead of touching freed memory.
type Request struct {
	mu  sync.Mutex
	ptr *C.cef_request_t
}

var RequestInvalidatedErr = errors.New("request is no longer valid")

// invalidateAndRelease detaches the wrapper from the underlying struct and
// drops the reference held for it.
func (r *Request) invalidateAndRelease() {
	r.mu.Lock()
	ptr := r.ptr
	r.ptr = nil
	r.mu.Unlock()

	if ptr != nil {
		releaseBase(unsafe.Pointer(ptr))
	}
}

func (r *Request) IsReadOnly() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.ptr == nil {
		// an invalidated request cannot be modified either way
		return true
	}
	return C.cBridgeRequestIsReadOnly(r.ptr) != 0
}

func (r *Request) GetReferrer() (*url.URL, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.ptr == nil {
		return nil, RequestInvalidatedErr
	}

	utf8string := C.cBridgeRequestGetReferrer(r.ptr)
	defer C.cef_string_userfree_utf8_free(utf8string)
	return url.ParseRequestURI(C.GoString(utf8string.str))
}

func (r *Request) SetUrl(uri *url.URL) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.ptr == nil {
		return RequestInvalidatedErr
	}
	if C.cBridgeRequestIsReadOnly(r.ptr) != 0 {
		return errors.New("request is read-only")
	}

	urlString := cefString(uri.String())
	defer cefStringFree(urlString)
	C.cBridgeRequestSetUrl(r.ptr, urlString)

	return nil
}

func (r *Request) GetUrl() (*url.URL, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.ptr == nil {
		return nil, RequestInvalidatedErr
	}

	utf8string := C.cBridgeRequestGetUrl(r.ptr)
	defer C.cef_string_userfree_utf8_free(utf8string)
	return url.ParseRequestURI(C.GoString(utf8string.str))
}

func (r *Request) GetMethod() string {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.ptr == nil {
		return ""
	}

	utf8string := C.cBridgeRequestGetMethod(r.ptr)
	defer C.cef_string_userfree_utf8_free(utf8string)
	return C.GoString(utf8string.str)
}

// From wraps a cef_request_t and takes over the reference ptr carries;
// invalidateAndRelease gives it back.
//
// refcheck:adopts ptr
func From(ptr unsafe.Pointer) *Request {
	return &Request{ptr: (*C.cef_request_t)(ptr)}
}
