package cef

/*
#include "include/capi/cef_values_capi.h"
#include "memory.h"

int value_is_valid(struct _cef_value_t* self) {
	return self -> is_valid(self);
}

int value_is_owned(struct _cef_value_t* self) {
	return self -> is_owned(self);
}

int value_is_read_only(struct _cef_value_t* self) {
	return self -> is_read_only(self);
}

int value_is_same(struct _cef_value_t* self, struct _cef_value_t* that REF_BORROWED) {
	// CEF releases the reference an argument carries (CppToC::Unwrap),
	// so the caller-owned |that| needs one of its own for the call.
	if (that) ((cef_base_ref_counted_t*)that)->add_ref((cef_base_ref_counted_t*)that);
	return self -> is_same(self, that);
}

int value_is_equal(struct _cef_value_t* self, struct _cef_value_t* that REF_BORROWED) {
	// CEF releases the reference an argument carries (CppToC::Unwrap),
	// so the caller-owned |that| needs one of its own for the call.
	if (that) ((cef_base_ref_counted_t*)that)->add_ref((cef_base_ref_counted_t*)that);
	return self -> is_equal(self, that);
}

struct _cef_value_t* value_copy(struct _cef_value_t* self) {
	return self -> copy(self);
}

cef_value_type_t value_get_type(struct _cef_value_t* self) {
	return self -> get_type(self);
}

int value_get_bool(struct _cef_value_t* self) {
	return self -> get_bool(self);
}

int value_get_int(struct _cef_value_t* self) {
	return self -> get_int(self);
}

double value_get_double(struct _cef_value_t* self) {
	return self -> get_double(self);
}

cef_string_userfree_t value_get_string(struct _cef_value_t* self) {
	return self -> get_string(self);
}

struct _cef_binary_value_t* value_get_binary(struct _cef_value_t* self) {
	return self -> get_binary(self);
}

struct _cef_dictionary_value_t* value_get_dictionary(struct _cef_value_t* self) {
	return self -> get_dictionary(self);
}

struct _cef_list_value_t* value_get_list(struct _cef_value_t* self) {
	return self -> get_list(self);
}

int value_set_null(struct _cef_value_t* self) {
	return self -> set_null(self);
}

int value_set_bool(struct _cef_value_t* self, int value) {
	return self -> set_bool(self, value);
}

int value_set_int(struct _cef_value_t* self, int value) {
	return self -> set_int(self, value);
}

int value_set_double(struct _cef_value_t* self, double value) {
	return self -> set_double(self, value);
}

int value_set_string(struct _cef_value_t* self, const cef_string_t* value) {
	return self -> set_string(self, value);
}

int value_set_binary(struct _cef_value_t* self, struct _cef_binary_value_t* value REF_CONSUMED) {
	return self -> set_binary(self, value);
}

int value_set_dictionary(struct _cef_value_t* self, struct _cef_dictionary_value_t* value REF_CONSUMED) {
	return self -> set_dictionary(self, value);
}

int value_set_list(struct _cef_value_t* self, struct _cef_list_value_t* value REF_CONSUMED) {
	return self -> set_list(self, value);
}

*/
import "C"
import "unsafe"

type (
	cefValue struct {
		cStruct *C.cef_value_t
	}

	CefValue interface {
		Valid() bool
		Owned() bool
		ReadOnly() bool

		// Copy, Binary and List return owned references - Release them when
		// done.
		Copy() CefValue
		Equals(CefValue) bool
		Same(CefValue) bool

		Type() ValueType

		Bool() bool
		Int() int
		Double() float64
		String() string
		Binary() CefBinaryValue

		List() CefListValue

		SetNull() bool
		SetBool(bool) bool
		SetInt(int) bool
		SetDouble(float64) bool
		SetString(string) bool
		// SetBinary and SetList consume their argument - it must not be
		// used or released afterwards.
		SetBinary(CefBinaryValue) bool
		SetList(value CefListValue) bool

		// Release drops the reference held by this wrapper.
		Release()

		ptr() *C.cef_value_t
	}
)

func (c cefValue) Valid() bool {
	return C.value_is_valid(c.cStruct) != 0
}

func (c cefValue) Owned() bool {
	return C.value_is_owned(c.cStruct) != 0
}

func (c cefValue) ReadOnly() bool {
	return C.value_is_read_only(c.cStruct) != 0
}

func (c cefValue) Copy() CefValue {
	return cefValue{C.value_copy(c.cStruct)}
}

func (c cefValue) Equals(value CefValue) bool {
	if value == nil {
		return false
	}
	return C.value_is_equal(c.cStruct, value.ptr()) != 0
}

func (c cefValue) Same(value CefValue) bool {
	if value == nil {
		return false
	}
	return C.value_is_same(c.cStruct, value.ptr()) != 0
}

func (c cefValue) Type() ValueType {
	return ValueType(C.value_get_type(c.cStruct))
}

func (c cefValue) Bool() bool {
	return C.value_get_bool(c.cStruct) != 0
}

func (c cefValue) Int() int {
	return int(C.value_get_int(c.cStruct))
}

func (c cefValue) Double() float64 {
	return float64(C.value_get_double(c.cStruct))
}

func (c cefValue) String() string {
	return cefUserfreeToGoString(C.value_get_string(c.cStruct))
}

func (c cefValue) Binary() CefBinaryValue {
	return cefBinaryValue{C.value_get_binary(c.cStruct)}
}

func (c cefValue) List() CefListValue {
	return cefListValue{C.value_get_list(c.cStruct)}
}

func (c cefValue) SetNull() bool {
	return C.value_set_null(c.cStruct) != 0
}

func (c cefValue) SetBool(b bool) bool {
	if b {
		return C.value_set_bool(c.cStruct, C.int(1)) != 0
	}
	return C.value_set_bool(c.cStruct, C.int(0)) != 0
}

func (c cefValue) SetInt(i int) bool {
	return C.value_set_int(c.cStruct, C.int(i)) != 0
}

func (c cefValue) SetDouble(f float64) bool {
	return C.value_set_double(c.cStruct, C.double(f)) != 0
}

func (c cefValue) SetString(s string) bool {
	valueString := cefString(s)
	defer cefStringFree(valueString)
	return C.value_set_string(c.cStruct, valueString) != 0
}

func (c cefValue) SetBinary(b CefBinaryValue) bool {
	if b == nil {
		return false
	}
	// refcheck:owned the value's reference goes to the container
	return C.value_set_binary(c.cStruct, b.ptr()) != 0
}

func (c cefValue) SetList(value CefListValue) bool {
	// refcheck:owned the value's reference goes to the container
	return C.value_set_list(c.cStruct, value.ptr()) != 0
}

func (c cefValue) Release() {
	releaseBase(unsafe.Pointer(c.cStruct))
}

func (c cefValue) ptr() *C.cef_value_t {
	return c.cStruct
}

var _ CefValue = (*cefValue)(nil)
