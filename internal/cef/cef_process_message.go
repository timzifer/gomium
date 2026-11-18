package cef

/*
#include "include/capi/cef_process_message_capi.h"

int process_message_is_valid(struct _cef_process_message_t* self) {
    return self -> is_valid(self);
}

int process_message_is_read_only(struct _cef_process_message_t* self) {
	return self -> is_read_only(self);
}

struct _cef_process_message_t* process_message_copy(struct _cef_process_message_t* self) {
	return self -> copy(self);
}

cef_string_userfree_t process_message_get_name(struct _cef_process_message_t* self) {
    return self->get_name(self);
}

struct _cef_list_value_t* process_message_get_argument_list(struct _cef_process_message_t* self) {
	return self -> get_argument_list(self);
}
*/
import "C"
import "unsafe"

type (
	processMessage struct {
		cStruct *C.cef_process_message_t
	}

	// ProcessMessage is a message received from the other process or about to
	// be sent to it. The CEF reference is managed by this package; callers
	// only read the name and decode the payload.
	ProcessMessage interface {
		Valid() bool
		Name() string
		ReadOnly() bool
		// DecodePayload gob-decodes the payload into target.
		DecodePayload(target any) error

		ptr() *C.cef_process_message_t
		// argumentList returns an owned reference - release it when done.
		argumentList() CefListValue
		// release drops the reference held by this wrapper.
		release()
	}
)

func (p processMessage) ptr() *C.cef_process_message_t {
	return p.cStruct
}

func (p processMessage) argumentList() CefListValue {
	return cefListValue{C.process_message_get_argument_list(p.cStruct)}
}

func (p processMessage) ReadOnly() bool {
	return C.process_message_is_read_only(p.cStruct) != 0
}

func (p processMessage) Valid() bool {
	return C.process_message_is_valid(p.cStruct) != 0
}

func (p processMessage) Name() string {
	return cefUserfreeToGoString(C.process_message_get_name(p.cStruct))
}

func (p processMessage) DecodePayload(target any) error {
	return decodeBinaryPayload(p, processMessagePayloadArgumentIndex, target)
}

func (p processMessage) release() {
	releaseBase(unsafe.Pointer(p.cStruct))
}

func newProcessMessage(name string) ProcessMessage {
	nameString := cefString(name)
	defer cefStringFree(nameString)

	return processMessage{
		cStruct: C.cef_process_message_create(nameString),
	}
}
