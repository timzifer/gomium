package cef

/*
#include <string.h>
#include "include/capi/cef_browser_capi.h"
#include "memory.h"
#include "include/internal/cef_types.h"

void browser_reload(struct _cef_browser_t* self) {
	self -> reload(self);
}
void browser_stop_load(struct _cef_browser_t* self) {
	self -> stop_load(self);
}

int browser_get_identifier(struct _cef_browser_t* self) {
	return self -> get_identifier(self);
}

int browser_is_same(struct _cef_browser_t* self, struct _cef_browser_t* that REF_BORROWED) {
	// CEF releases the reference an argument carries (CppToC::Unwrap),
	// so the caller-owned |that| needs one of its own for the call.
	if (that) ((cef_base_ref_counted_t*)that)->add_ref((cef_base_ref_counted_t*)that);
	return self -> is_same(self, that);
}

int browser_is_popup(struct _cef_browser_t* self) {
	return self -> is_popup(self);
}

int browser_has_document(struct _cef_browser_t* self) {
	return self -> has_document(self);
}

struct _cef_frame_t* browser_get_main_frame(struct _cef_browser_t* self) {
	return self -> get_main_frame(self);
}

struct _cef_frame_t* browser_get_focused_frame(struct _cef_browser_t* self) {
	return self -> get_focused_frame(self);
}

struct _cef_frame_t* browser_get_frame_by_identifier(struct _cef_browser_t* self, const cef_string_t* identifier) {
	return self -> get_frame_by_identifier(self, identifier);
}

struct _cef_frame_t* browser_get_frame_by_name(struct _cef_browser_t* self, const cef_string_t* name) {
	return self -> get_frame_by_name(self, name);
}

size_t browser_get_frame_count(struct _cef_browser_t* self) {
	return self -> get_frame_count(self);
}

void browser_get_frame_identifiers(struct _cef_browser_t* self, cef_string_list_t identifiers) {
	return self -> get_frame_identifiers(self, identifiers);
}

void browser_get_frame_names(struct _cef_browser_t* self, cef_string_list_t names) {
	return self -> get_frame_names(self, names);
}

int browser_send_process_message(struct _cef_browser_t* self, cef_process_id_t target_process, struct _cef_process_message_t* message REF_CONSUMED) {
	// cef_browser_t lost send_process_message - route through the main frame
	struct _cef_frame_t* main_frame = self -> get_main_frame(self);
	if (main_frame == NULL) {
		// send_process_message would have consumed the message; without a
		// frame to send it through, this is the last owner
		((cef_base_ref_counted_t*)message)->release((cef_base_ref_counted_t*)message);
		return 0;
	}
	main_frame -> send_process_message(main_frame, target_process, message);
	((cef_base_ref_counted_t*)main_frame)->release((cef_base_ref_counted_t*)main_frame);
	return 1;
}
*/
import "C"
import (
	"unsafe"
)

type (
	cefBrowser struct {
		cStruct *C.cef_browser_t
	}

	CefBrowser interface {
		Reload()
		StopLoad()
		Identifier() int32
		Equals(CefBrowser) bool
		IsPopup() bool
		HasDocument() bool
		// MainFrame, FocusedFrame, GetFrameByIdentifier, GetFrame and Frames
		// return owned references - call Release on the result when done.
		MainFrame() CefFrame
		FocusedFrame() CefFrame
		GetFrameByIdentifier(identifier string) CefFrame
		GetFrame(name string) CefFrame
		Frames() []CefFrame
		// sendProcessMessage consumes the message - it must not be used
		// afterwards.
		sendProcessMessage(id CefProcessId, message ProcessMessage) bool

		// Release drops the reference held by this wrapper (only for
		// wrappers obtained from owning getters).
		Release()
	}
)

func (c cefBrowser) sendProcessMessage(id CefProcessId, message ProcessMessage) bool {
	// refcheck:owned sendProcessMessage consumes the message
	return C.browser_send_process_message(c.cStruct, C.cef_process_id_t(id), message.ptr()) != 0
}

func (c cefBrowser) IsPopup() bool {
	return C.browser_is_popup(c.cStruct) != 0
}

func (c cefBrowser) HasDocument() bool {
	return C.browser_has_document(c.cStruct) != 0
}

func (c cefBrowser) MainFrame() CefFrame {
	return &cefFrame{cStruct: C.browser_get_main_frame(c.cStruct)}
}

func (c cefBrowser) FocusedFrame() CefFrame {
	return &cefFrame{cStruct: C.browser_get_focused_frame(c.cStruct)}
}

func (c cefBrowser) GetFrameByIdentifier(identifier string) CefFrame {
	frameIdentifier := cefString(identifier)
	defer cefStringFree(frameIdentifier)

	fr := C.browser_get_frame_by_identifier(c.cStruct, frameIdentifier)
	if nil == fr {
		return nil
	}
	return &cefFrame{cStruct: fr}
}

func (c cefBrowser) GetFrame(name string) CefFrame {
	frameName := cefString(name)
	defer cefStringFree(frameName)

	fr := C.browser_get_frame_by_name(c.cStruct, frameName)
	if nil == fr {
		return nil
	}
	return &cefFrame{cStruct: fr}
}

func (c cefBrowser) Frames() []CefFrame {
	identifierList := C.cef_string_list_alloc()
	defer C.cef_string_list_free(identifierList)

	C.browser_get_frame_identifiers(c.cStruct, identifierList)

	var frames []CefFrame
	for index := C.size_t(0); index < C.cef_string_list_size(identifierList); index++ {
		var str C.cef_string_t
		if C.cef_string_list_value(identifierList, index, &str) == 0 {
			continue
		}
		identifier := cefToGoString(&str)
		C.cef_string_clear(&str)

		if fr := c.GetFrameByIdentifier(identifier); fr != nil {
			frames = append(frames, fr)
		}
	}
	return frames
}

func (c cefBrowser) Equals(that CefBrowser) bool {
	if cB, isCB := that.(cefBrowser); isCB {
		return C.browser_is_same(c.cStruct, cB.cStruct) != 0
	}
	if cB, isCB := that.(*cefBrowser); isCB {
		return C.browser_is_same(c.cStruct, cB.cStruct) != 0
	}
	return false
}

func (c cefBrowser) Release() {
	releaseBase(unsafe.Pointer(c.cStruct))
}

func (c cefBrowser) Identifier() int32 {
	return int32(C.browser_get_identifier(c.cStruct))
}

func (c cefBrowser) StopLoad() {
	C.browser_stop_load(c.cStruct)
}

func (c cefBrowser) Reload() {
	C.browser_reload(c.cStruct)
}

var _ CefBrowser = (*cefBrowser)(nil)
