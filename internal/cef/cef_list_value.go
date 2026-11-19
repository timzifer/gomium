package cef

/*
#include "include/capi/cef_values_capi.h"
#include "memory.h"

int list_value_is_valid(struct _cef_list_value_t* self) {
	return self -> is_valid(self);
}

int list_value_is_owned(struct _cef_list_value_t* self) {
	return self -> is_owned(self);
}

int list_value_is_read_only(struct _cef_list_value_t* self) {
	return self -> is_read_only(self);
}

int list_value_is_same(struct _cef_list_value_t* self, struct _cef_list_value_t* that REF_BORROWED) {
	// CEF releases the reference an argument carries (CppToC::Unwrap),
	// so the caller-owned |that| needs one of its own for the call.
	if (that) ((cef_base_ref_counted_t*)that)->add_ref((cef_base_ref_counted_t*)that);
	return self -> is_same(self, that);
}

int list_value_is_equal(struct _cef_list_value_t* self, struct _cef_list_value_t* that REF_BORROWED) {
	// CEF releases the reference an argument carries (CppToC::Unwrap),
	// so the caller-owned |that| needs one of its own for the call.
	if (that) ((cef_base_ref_counted_t*)that)->add_ref((cef_base_ref_counted_t*)that);
	return self -> is_equal(self, that);
}

struct _cef_list_value_t* list_value_copy(struct _cef_list_value_t* self) {
	return self -> copy(self);
}

int list_value_set_size(struct _cef_list_value_t* self, size_t size) {
	return self -> set_size(self, size);
}

size_t list_value_get_size(struct _cef_list_value_t* self) {
	return self -> get_size(self);
}

int list_value_clear(struct _cef_list_value_t* self) {
	return self -> clear(self);
}

int list_value_remove(struct _cef_list_value_t* self, size_t index) {
	return self -> remove(self, index);
}

cef_value_type_t list_value_get_type(struct _cef_list_value_t* self, size_t index) {
	return self -> get_type(self, index);
}

struct _cef_value_t* list_value_get_value(struct _cef_list_value_t* self, size_t index) {
	return self -> get_value(self, index);
}


int list_value_get_bool(struct _cef_list_value_t* self, size_t index) {
	return self -> get_bool(self, index);
}

int list_value_get_int(struct _cef_list_value_t* self, size_t index) {
	return self -> get_int(self, index);
}

double list_value_get_double(struct _cef_list_value_t* self, size_t index) {
	return self -> get_double(self, index);
}

cef_string_userfree_t list_value_get_string(struct _cef_list_value_t* self, size_t index) {
	return self -> get_string(self, index);
}

struct _cef_binary_value_t* list_value_get_binary(struct _cef_list_value_t* self, size_t index) {
	return self -> get_binary(self, index);
}

struct _cef_dictionary_value_t* list_value_get_dictionary(struct _cef_list_value_t* self, size_t index) {
	return self -> get_dictionary(self, index);
}

struct _cef_list_value_t* list_value_get_list(struct _cef_list_value_t* self, size_t index) {
	return self -> get_list(self, index);
}

int list_value_set_value(struct _cef_list_value_t* self, size_t index, struct _cef_value_t* value REF_CONSUMED) {
	return self -> set_value(self, index, value);
}

int list_value_set_null(struct _cef_list_value_t* self, size_t index) {
	return self -> set_null(self, index);
}

int list_value_set_bool(struct _cef_list_value_t* self, size_t index, int value) {
	return self -> set_bool(self, index, value);
}

int list_value_set_int(struct _cef_list_value_t* self, size_t index, int value) {
	return self -> set_int(self, index, value);
}

int list_value_set_double(struct _cef_list_value_t* self, size_t index, double value) {
	return self -> set_double(self, index, value);
}

int list_value_set_string(struct _cef_list_value_t* self, size_t index, const cef_string_t* value) {
	return self -> set_string(self, index, value);
}

int list_value_set_binary(struct _cef_list_value_t* self, size_t index, struct _cef_binary_value_t* value REF_CONSUMED) {
	return self -> set_binary(self, index, value);
}

int list_value_set_dictionary(struct _cef_list_value_t* self, size_t index, struct _cef_dictionary_value_t* value REF_CONSUMED) {
	return self -> set_dictionary(self, index, value);
}

int list_value_set_list(struct _cef_list_value_t* self, size_t index, struct _cef_list_value_t* value REF_CONSUMED) {
	return self -> set_list(self, index, value);
}

*/
import "C"
import "unsafe"

type (
	cefListValue struct {
		cStruct *C.cef_list_value_t
	}

	CefListValue interface {
		Valid() bool
		Owned() bool
		ReadOnly() bool
		Same(value CefListValue) bool
		Equals(value CefListValue) bool
		// Copy, List and Binary return owned references - Release them when
		// done.
		Copy() CefListValue
		SetSize(int) bool
		Size() int
		Clear() bool
		Remove(index int) bool
		GetType(index int) ValueType

		//Value(index int) any

		Bool(index int) bool
		Int(index int) int
		Double(index int) float64
		String(index int) string
		List(index int) CefListValue
		Binary(index int) CefBinaryValue

		//SetValue(index int, value any) bool

		SetBool(index int, value bool) bool
		SetInt(index int, value int) bool
		SetDouble(index int, value float64) bool
		SetString(index int, value string) bool
		// SetList and SetBinary consume value - it must not be used or
		// released afterwards.
		SetList(index int, value CefListValue) bool
		SetBinary(index int, value CefBinaryValue) bool

		Slice() []any

		// Release drops the reference held by this wrapper.
		Release()

		ptr() *C.cef_list_value_t
	}
)

func (l cefListValue) Slice() []any {
	length := l.Size()
	result := make([]any, length)

	for idx := 0; idx < length; idx++ {
		theType := l.GetType(idx)

		switch theType {
		case ValueTypeBool:
			result[idx] = l.Bool(idx)
		case ValueTypeInt:
			result[idx] = l.Int(idx)
		case ValueTypeDouble:
			result[idx] = l.Double(idx)
		case ValueTypeList:
			result[idx] = l.List(idx)
		case ValueTypeString:
			result[idx] = l.String(idx)
		default:
			// null, invalid and unsupported types (binary, dictionary)
			result[idx] = nil
		}
	}
	return result
}

func (l cefListValue) ptr() *C.cef_list_value_t {
	return l.cStruct
}

func (l cefListValue) Owned() bool {
	return C.list_value_is_owned(l.cStruct) != 0
}

func (l cefListValue) ReadOnly() bool {
	return C.list_value_is_read_only(l.cStruct) != 0
}

func (l cefListValue) Same(value CefListValue) bool {
	return C.list_value_is_same(l.cStruct, value.ptr()) != 0
}

func (l cefListValue) Equals(value CefListValue) bool {
	return C.list_value_is_equal(l.cStruct, value.ptr()) != 0
}

func (l cefListValue) Copy() CefListValue {
	return cefListValue{cStruct: C.list_value_copy(l.cStruct)}
}

func (l cefListValue) SetSize(i int) bool {
	return C.list_value_set_size(l.cStruct, C.size_t(i)) != 0
}

func (l cefListValue) Size() int {
	return int(C.list_value_get_size(l.cStruct))
}

func (l cefListValue) Clear() bool {
	return C.list_value_clear(l.cStruct) != 0
}

func (l cefListValue) Remove(index int) bool {
	return C.list_value_remove(l.cStruct, C.size_t(index)) != 0
}

func (l cefListValue) GetType(index int) ValueType {
	return ValueType(C.list_value_get_type(l.cStruct, C.size_t(index)))
}

func (l cefListValue) Bool(index int) bool {
	return C.list_value_get_bool(l.cStruct, C.size_t(index)) != 0
}

func (l cefListValue) Int(index int) int {
	return int(C.list_value_get_int(l.cStruct, C.size_t(index)))
}

func (l cefListValue) Double(index int) float64 {
	return float64(C.list_value_get_double(l.cStruct, C.size_t(index)))
}

func (l cefListValue) String(index int) string {
	return cefUserfreeToGoString(C.list_value_get_string(l.cStruct, C.size_t(index)))
}

func (l cefListValue) List(index int) CefListValue {
	return cefListValue{C.list_value_get_list(l.cStruct, C.size_t(index))}
}

func (l cefListValue) Binary(index int) CefBinaryValue {
	return cefBinaryValue{C.list_value_get_binary(l.cStruct, C.size_t(index))}
}

func (l cefListValue) SetBool(index int, value bool) bool {
	if value {
		return C.list_value_set_bool(l.cStruct, C.size_t(index), 1) != 0
	} else {
		return C.list_value_set_bool(l.cStruct, C.size_t(index), 0) != 0
	}
}

func (l cefListValue) SetInt(index int, value int) bool {
	return C.list_value_set_int(l.cStruct, C.size_t(index), C.int(value)) != 0
}

func (l cefListValue) SetDouble(index int, value float64) bool {
	return C.list_value_set_double(l.cStruct, C.size_t(index), C.double(value)) != 0
}

func (l cefListValue) SetString(index int, value string) bool {
	valueString := cefString(value)
	defer cefStringFree(valueString)
	return C.list_value_set_string(l.cStruct, C.size_t(index), valueString) != 0
}

func (l cefListValue) SetList(index int, value CefListValue) bool {
	// refcheck:owned the value's reference goes to the container
	return C.list_value_set_list(l.cStruct, C.size_t(index), value.ptr()) != 0
}

func (l cefListValue) SetBinary(index int, value CefBinaryValue) bool {
	// refcheck:owned the value's reference goes to the container
	return C.list_value_set_binary(l.cStruct, C.size_t(index), value.ptr()) != 0
}

func (l cefListValue) Valid() bool {
	return C.list_value_is_valid(l.cStruct) != 0
}

func (l cefListValue) Release() {
	releaseBase(unsafe.Pointer(l.cStruct))
}

var _ CefListValue = (*cefListValue)(nil)
