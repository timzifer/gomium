package cef

/*
#include "include/capi/cef_values_capi.h"
#include "memory.h"

void binary_value_free(struct _cef_binary_value_t* self) {
    ((cef_base_ref_counted_t*)self)->release((cef_base_ref_counted_t*)self);
}

int binary_value_is_valid(struct _cef_binary_value_t* self) {
	return self -> is_valid(self);
}

int binary_value_is_owned(struct _cef_binary_value_t* self) {
	return self -> is_owned(self);
}

int binary_value_is_same(struct _cef_binary_value_t* self, struct _cef_binary_value_t* that) {
	// CEF releases the reference an argument carries (CppToC::Unwrap),
	// so the caller-owned |that| needs one of its own for the call.
	if (that) ((cef_base_ref_counted_t*)that)->add_ref((cef_base_ref_counted_t*)that);
	return self -> is_same(self, that);
}

int binary_value_is_equal(struct _cef_binary_value_t* self, struct _cef_binary_value_t* that) {
	// CEF releases the reference an argument carries (CppToC::Unwrap),
	// so the caller-owned |that| needs one of its own for the call.
	if (that) ((cef_base_ref_counted_t*)that)->add_ref((cef_base_ref_counted_t*)that);
	return self -> is_equal(self, that);
}

struct _cef_binary_value_t* binary_value_copy(struct _cef_binary_value_t* self) {
	return self -> copy(self);
}

size_t binary_value_get_size(struct _cef_binary_value_t* self) {
	return self -> get_size(self);
}

size_t binary_value_get_data(struct _cef_binary_value_t* self, void* buffer, size_t buffer_size, size_t data_offset) {
	return self -> get_data(self, buffer, buffer_size, data_offset);
}

*/
import "C"
import (
	"bytes"
	"unsafe"
)

type (
	cefBinaryValue struct {
		cStruct *C.cef_binary_value_t
	}

	CefBinaryValue interface {
		Valid() bool
		Owned() bool
		Same(CefBinaryValue) bool
		Equal(CefBinaryValue) bool
		// Copy returns an owned reference - Release it when done.
		Copy() CefBinaryValue
		Size() int
		Data() *bytes.Buffer
		// Release drops the reference held by this wrapper.
		Release()

		ptr() *C.cef_binary_value_t
	}
)

func (c cefBinaryValue) ptr() *C.cef_binary_value_t {
	return c.cStruct
}

func (c cefBinaryValue) Valid() bool {
	if c.cStruct == nil {
		return false
	}
	return C.binary_value_is_valid(c.cStruct) != 0
}

func (c cefBinaryValue) Owned() bool {
	return C.binary_value_is_owned(c.cStruct) != 0
}

func (c cefBinaryValue) Same(value CefBinaryValue) bool {
	return C.binary_value_is_same(c.cStruct, value.ptr()) != 0
}

func (c cefBinaryValue) Equal(value CefBinaryValue) bool {
	return C.binary_value_is_equal(c.cStruct, value.ptr()) != 0
}

func (c cefBinaryValue) Copy() CefBinaryValue {
	return cefBinaryValue{cStruct: C.binary_value_copy(c.cStruct)}
}

func (c cefBinaryValue) Size() int {
	if c.cStruct == nil {
		return 0
	}
	return int(C.binary_value_get_size(c.cStruct))
}

func (c cefBinaryValue) Data() *bytes.Buffer {
	if c.cStruct == nil {
		return bytes.NewBuffer(nil)
	}

	bufferLength := c.Size()
	if bufferLength <= 0 {
		return bytes.NewBuffer(nil)
	}

	buf := make([]byte, bufferLength)
	read := int(C.binary_value_get_data(c.cStruct, unsafe.Pointer(&buf[0]), C.size_t(bufferLength), C.size_t(0)))
	if read < 0 {
		read = 0
	}

	return bytes.NewBuffer(buf[:read])
}

func (c cefBinaryValue) Release() {
	releaseBase(unsafe.Pointer(c.cStruct))
}

// BinaryValueFromBuffer copies the buffer contents into a new
// cef_binary_value_t. Release it when done (passing it to a Set* function
// transfers the reference instead).
func BinaryValueFromBuffer(buffer *bytes.Buffer) CefBinaryValue {
	buf := buffer.Bytes()
	if len(buf) == 0 {
		return cefBinaryValue{nil}
	}

	return cefBinaryValue{C.cef_binary_value_create(unsafe.Pointer(&buf[0]), C.size_t(len(buf)))}
}
