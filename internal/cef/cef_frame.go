package cef

/*
#include <string.h>
#include "include/capi/cef_frame_capi.h"

cef_string_userfree_t frame_get_name(struct _cef_frame_t* self) {
	return self -> get_name(self);
}

cef_string_userfree_t frame_get_identifier(struct _cef_frame_t* self) {
	return self -> get_identifier(self);
}


int frame_is_main(struct _cef_frame_t* self) {
	return self -> is_main(self);
}

int frame_is_focused(struct _cef_frame_t* self) {
	return self -> is_focused(self);
}

struct _cef_frame_t* frame_get_parent(struct _cef_frame_t* self) {
	return self -> get_parent(self);
}

cef_string_userfree_t frame_get_url(struct _cef_frame_t* self) {
	return self -> get_url(self);
}

struct _cef_browser_t* frame_get_browser(struct _cef_frame_t* self) {
	return self -> get_browser(self);
}

struct _cef_v8_context_t* frame_get_v8context(struct _cef_frame_t* self) {
	return self -> get_v8_context(self);
}

int frame_is_valid(struct _cef_frame_t* self) {
	return self -> is_valid(self);
}

void frame_load_request(struct _cef_frame_t* self, struct _cef_request_t* request) {
	self -> load_request(self, request);
}

void frame_load_url(struct _cef_frame_t* self, const cef_string_t* url) {
	self -> load_url(self, url);
}

void frame_execute_java_script(struct _cef_frame_t* self, const cef_string_t* code, const cef_string_t* script_url, int start_line) {
	self -> execute_java_script(self, code, script_url, start_line);
}

void frame_send_process_message(struct _cef_frame_t* self, cef_process_id_t target_process, struct _cef_process_message_t* message) {
	self -> send_process_message(self, target_process, message);
}
*/
import "C"
import (
	"encoding/base64"
	"errors"
	"github.com/google/uuid"
	"net/url"
	"unsafe"
)

type (
	cefFrame struct {
		cStruct *C.cef_frame_t
	}

	CefFrame interface {
		Name() string
		// Identifier returns the globally unique frame identifier. Since CEF
		// 122 frame identifiers are strings, not int64.
		Identifier() string
		IsMain() bool
		IsFocused() bool
		IsValid() bool
		// Parent and Browser return owned references - call Release on the
		// result when done.
		Parent() CefFrame
		Browser() CefBrowser

		ExecuteJavaScript(code string, sourceUrl *url.URL, line int)
		ExecuteJavaScriptWithResult(code string, sourceUrl *url.URL, line int, callback ResultCallback) error

		LoadString(source string, sourceUrl *url.URL)
		LoadUrl(url url.URL)

		// SendProcessMessage consumes the message - it must not be used
		// afterwards.
		SendProcessMessage(id CefProcessId, message ProcessMessage) bool

		V8() V8Context

		// Release drops the reference held by this wrapper (only for
		// wrappers obtained from owning getters).
		Release()
	}
)

func (f cefFrame) ExecuteJavaScriptWithResult(code string, sourceUrl *url.URL, line int, cb ResultCallback) error {

	theBrowser := f.Browser()
	if theBrowser == nil {
		return errors.New("frame has no browser")
	}
	defer theBrowser.Release()

	id := uuid.New()
	registerJavaScriptEvaluationCallback(id, cb)

	if success, err := (javaScriptEvaluationRequest{
		Id:      id,
		Line:    int64(line),
		Code:    code,
		Url:     sourceUrl,
		Frame:   f.Identifier(),
		Browser: theBrowser.Identifier(),
	}).DispatchWith(f, PidRenderer); err != nil {
		failJavaScriptEvaluation(id, err)
		return err
	} else if !success {
		err := errors.New("could not dispatch execution request")
		failJavaScriptEvaluation(id, err)
		return err
	}
	return nil
}

func (f cefFrame) Browser() CefBrowser {
	theBrowser := C.frame_get_browser(f.cStruct)
	if nil != theBrowser {
		return &cefBrowser{theBrowser}
	}
	return nil
}

func (f cefFrame) V8() V8Context {
	v8contextLock.Lock()
	defer v8contextLock.Unlock()

	// explicit miss check - returning the map's zero value would produce a
	// non-nil interface wrapping a nil pointer
	if ctx, exists := v8contexts[f.Identifier()]; exists && ctx != nil {
		return ctx
	}
	return nil
}

func (f cefFrame) Release() {
	releaseBase(unsafe.Pointer(f.cStruct))
}

func (f cefFrame) IsValid() bool {
	if f.cStruct == nil {
		return false
	}

	return C.frame_is_valid(f.cStruct) == 1
}

func (f cefFrame) Parent() CefFrame {
	parent := C.frame_get_parent(f.cStruct)
	if nil != parent {
		return &cefFrame{parent}
	}
	return nil
}

func (f cefFrame) ExecuteJavaScript(code string, sourceUrl *url.URL, line int) {
	var sourceUrlString *C.cef_string_t
	if sourceUrl != nil {
		sourceUrlString = cefString(sourceUrl.String())
		defer cefStringFree(sourceUrlString)
	}
	codeString := cefString(code)
	defer cefStringFree(codeString)
	C.frame_execute_java_script(f.cStruct, codeString, sourceUrlString, C.int(line))
}

// LoadString loads the given HTML via a data: URL - cef_frame_t lost its
// load_string member. The sourceUrl parameter can no longer be honoured as
// the document origin; it is ignored.
func (f cefFrame) LoadString(source string, sourceUrl *url.URL) {
	dataUrlString := cefString(dataURLForHTML(source))
	defer cefStringFree(dataUrlString)
	C.frame_load_url(f.cStruct, dataUrlString)
}

// dataURLForHTML packs an HTML document into a base64 data: URL.
func dataURLForHTML(source string) string {
	return "data:text/html;base64," + base64.StdEncoding.EncodeToString([]byte(source))
}

func (f cefFrame) SendProcessMessage(id CefProcessId, message ProcessMessage) bool {
	C.frame_send_process_message(f.cStruct, C.cef_process_id_t(id), message.ptr())
	return true
}

func (f cefFrame) LoadUrl(url url.URL) {
	urlString := cefString(url.String())
	defer cefStringFree(urlString)
	C.frame_load_url(f.cStruct, urlString)
}

func (f cefFrame) IsMain() bool {
	return C.frame_is_main(f.cStruct) != 0
}

func (f cefFrame) IsFocused() bool {
	return C.frame_is_focused(f.cStruct) != 0
}

var _ CefFrame = (*cefFrame)(nil)

func (f cefFrame) Name() string {
	return cefUserfreeToGoString(C.frame_get_name(f.cStruct))
}

func (f cefFrame) Identifier() string {
	return cefUserfreeToGoString(C.frame_get_identifier(f.cStruct))
}

// url is the address the frame currently shows. In the browser process CEF
// allows this on any thread.
func (f cefFrame) url() string {
	return cefUserfreeToGoString(C.frame_get_url(f.cStruct))
}
