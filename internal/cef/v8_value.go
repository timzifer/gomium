package cef

/*
#include "v8.h"
#include "debug.h"
#include "memory.h"
#include <string.h>

#include "include/internal/cef_string_list.h"
#include "include/internal/cef_time.h"

int cef_v8_value_is_valid(struct _cef_v8_value_t* self){
	return self -> is_valid(self);
}
int cef_v8_value_is_same(struct _cef_v8_value_t* self, struct _cef_v8_value_t* that){
	// CEF releases the reference an argument carries (CppToC::Unwrap),
	// so the caller-owned |that| needs one of its own for the call.
	if (that) ((cef_base_ref_counted_t*)that)->add_ref((cef_base_ref_counted_t*)that);
	return self -> is_same(self, that);
}
int cef_v8_value_get_array_length(struct _cef_v8_value_t* self){
	return self -> get_array_length(self);
}

cef_v8_value_t* cef_v8_value_get_value_byindex(cef_v8_value_t* self, int index) {
	return self -> get_value_byindex(self, index);
}

cef_string_userfree_t cef_v8_value_get_function_name(cef_v8_value_t* self) {
	return self -> get_function_name(self);
}

int cef_v8_value_is_user_created(struct _cef_v8_value_t* self) {
	return self -> is_user_created(self);
}

int cef_v8_value_get_keys(struct _cef_v8_value_t* self, cef_string_list_t keys) {
	return self -> get_keys(self, keys);
}

int cef_v8_value_is_undefined(struct _cef_v8_value_t* self) {
	return self -> is_undefined(self);
}

int cef_v8_value_is_null(struct _cef_v8_value_t* self) {
	return self -> is_null(self);
}

int cef_v8_value_is_bool(struct _cef_v8_value_t* self) {
	return self -> is_bool(self);
}

int cef_v8_value_is_int(struct _cef_v8_value_t* self) {
	return self -> is_int(self);
}

int cef_v8_value_is_uint(struct _cef_v8_value_t* self) {
	return self -> is_uint(self);
}

int cef_v8_value_is_double(struct _cef_v8_value_t* self) {
	return self -> is_double(self);
}

int cef_v8_value_is_string(struct _cef_v8_value_t* self) {
	return self -> is_string(self);
}

cef_string_userfree_t cef_v8_value_get_string(struct _cef_v8_value_t* self) {
	return self -> get_string_value(self);
}

int cef_v8_value_is_object(struct _cef_v8_value_t* self) {
	return self -> is_object(self);
}

int cef_v8_value_is_array(struct _cef_v8_value_t* self) {
	return self -> is_array(self);
}

int cef_v8_value_is_date(struct _cef_v8_value_t* self) {
	return self -> is_date(self);
}

int cef_v8_value_is_function(struct _cef_v8_value_t* self) {
	return self -> is_function(self);
}

 int release_v8value(cef_v8_value_t* value) {
         return value->base.release(&value->base);
 }

int setValueByKey(cef_v8_value_t* obj, const char* key, cef_v8_value_t* value) {
    cef_string_t cefKey = {};
    cef_string_utf8_to_utf16(key, strlen(key), &cefKey);
	int result = obj -> set_value_bykey(obj, &cefKey, value,V8_PROPERTY_ATTRIBUTE_NONE);
    cef_string_clear(&cefKey); // Clean up
	return result;
}

cef_v8_value_t* getValueByKey(cef_v8_value_t* obj, const char* key) {
    cef_string_t cefKey = {};
    cef_string_utf8_to_utf16(key, strlen(key), &cefKey);
    cef_v8_value_t* result = obj->get_value_bykey(obj, &cefKey);
    cef_string_clear(&cefKey); // Clean up

    return result;
}

int hasValueByKey(cef_v8_value_t* obj, const char* key) {
    cef_string_t cefKey = {};
    cef_string_utf8_to_utf16(key, strlen(key), &cefKey);
   int result = obj->has_value_bykey(obj, &cefKey);
    cef_string_clear(&cefKey); // Clean up
    return result;
}
*/
import "C"
import (
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unsafe"
)

func init() {
	gob.Register(&Exported{})
}

type (
	Exported struct {
		String    *string
		Int       *int
		UInt      *uint
		Float     *float64
		Array     []*Exported
		Map       map[string]*Exported
		Bool      *bool
		Time      *time.Time
		Function  *string
		Nil       bool
		Undefined bool
		IsObject  bool
		IsArray   bool
		// Cycle marks a value that was already exported earlier in the same
		// graph (circular reference).
		Cycle bool
		// Kind names the scalar field that carries the value. gob leaves out
		// zero values, also behind a pointer: "", 0 and false arrive as a nil
		// pointer and would read as null. Kind is never zero for a scalar, so
		// it survives the trip and Native restores the zero value.
		Kind exportedKind
	}

	exportedKind uint8
)

const (
	exportedOther exportedKind = iota
	exportedString
	exportedInt
	exportedUint
	exportedFloat
	exportedBool
)

func ptr[T any](t T) *T {
	return &t
}

// valueOrZero reads a scalar of Exported; nil is the zero value gob dropped.
func valueOrZero[T any](p *T) T {
	var zero T
	if p == nil {
		return zero
	}
	return *p
}

func (e *Exported) Native() any {
	if e == nil {
		return nil
	}
	switch e.Kind {
	case exportedString:
		return valueOrZero(e.String)
	case exportedInt:
		return valueOrZero(e.Int)
	case exportedUint:
		return valueOrZero(e.UInt)
	case exportedFloat:
		return valueOrZero(e.Float)
	case exportedBool:
		return valueOrZero(e.Bool)
	}
	if nil != e.String {
		return *e.String
	}
	if nil != e.Int {
		return *e.Int
	}
	if nil != e.UInt {
		return *e.UInt
	}
	if nil != e.Float {
		return *e.Float
	}
	if nil != e.Bool {
		return *e.Bool
	}
	if e.IsArray {
		ret := make([]interface{}, len(e.Array))
		for idx := range e.Array {
			ret[idx] = e.Array[idx].Native()
		}
		return ret
	}
	if e.IsObject {
		ret := make(map[string]any)
		for idx := range e.Map {
			ret[idx] = e.Map[idx].Native()
		}
		return ret
	}
	if e.Time != nil {
		return e.Time
	}
	if e.Function != nil {
		return e.Function
	}
	if e.Nil || e.Undefined {
		return nil
	}

	return nil
}

var (
// _ gob.GobEncoder = (*ExportedObjectValue)(nil)
)

// exportedFromNative converts a plain Go value (as returned by extension
// functions) into the wire representation. The inverse of Exported.Native.
func exportedFromNative(val any, ttl int) (*Exported, error) {
	if ttl <= 0 {
		return nil, errors.New("value nesting too deep")
	}

	switch v := val.(type) {
	case nil:
		return &Exported{Nil: true}, nil
	case *Exported:
		return v, nil
	case time.Time:
		return &Exported{Time: &v}, nil
	case string:
		return &Exported{Kind: exportedString, String: &v}, nil
	case bool:
		return &Exported{Kind: exportedBool, Bool: &v}, nil
	case float32:
		return &Exported{Kind: exportedFloat, Float: ptr(float64(v))}, nil
	case float64:
		return &Exported{Kind: exportedFloat, Float: &v}, nil
	case int:
		return &Exported{Kind: exportedInt, Int: &v}, nil
	case int8:
		return &Exported{Kind: exportedInt, Int: ptr(int(v))}, nil
	case int16:
		return &Exported{Kind: exportedInt, Int: ptr(int(v))}, nil
	case int32:
		return &Exported{Kind: exportedInt, Int: ptr(int(v))}, nil
	case int64:
		return &Exported{Kind: exportedInt, Int: ptr(int(v))}, nil
	case uint:
		return &Exported{Kind: exportedUint, UInt: &v}, nil
	case uint8:
		return &Exported{Kind: exportedUint, UInt: ptr(uint(v))}, nil
	case uint16:
		return &Exported{Kind: exportedUint, UInt: ptr(uint(v))}, nil
	case uint32:
		return &Exported{Kind: exportedUint, UInt: ptr(uint(v))}, nil
	case uint64:
		return &Exported{Kind: exportedUint, UInt: ptr(uint(v))}, nil
	}

	reflected := reflect.ValueOf(val)
	switch reflected.Kind() {
	case reflect.Slice, reflect.Array:
		ret := &Exported{IsArray: true, Array: make([]*Exported, reflected.Len())}
		for idx := 0; idx < reflected.Len(); idx++ {
			sub, subErr := exportedFromNative(reflected.Index(idx).Interface(), ttl-1)
			if subErr != nil {
				return nil, subErr
			}
			ret.Array[idx] = sub
		}
		return ret, nil
	case reflect.Map:
		if reflected.Type().Key().Kind() != reflect.String {
			return nil, fmt.Errorf("only string-keyed maps can be exported, got %T", val)
		}
		ret := &Exported{IsObject: true, Map: make(map[string]*Exported)}
		for _, key := range reflected.MapKeys() {
			sub, subErr := exportedFromNative(reflected.MapIndex(key).Interface(), ttl-1)
			if subErr != nil {
				return nil, subErr
			}
			ret.Map[key.String()] = sub
		}
		return ret, nil
	default:
		return nil, fmt.Errorf("unsupported extension result type: %T", val)
	}
}

// valueFromPointer wraps an OWNED cef_v8_value_t reference (as returned by
// eval, get_value_bykey, get_value_byindex, ...). The wrapper takes over that
// reference and releases it on Close. On error the reference is released
// here.
func valueFromPointer(ptr *C.cef_v8_value_t) (Value, error) {
	if nil == ptr {
		return nil, nil
	}
	if C.cef_v8_value_is_valid(ptr) == C.int(0) {
		releaseBase(unsafe.Pointer(ptr))
		return nil, errors.New("value-pointer is invalid")
	}

	if C.cef_v8_value_is_null(ptr) == C.int(1) {
		return &NullValue{
			value{ptr},
		}, nil
	} else if C.cef_v8_value_is_undefined(ptr) == C.int(1) {
		return &UndefinedValue{
			value{ptr},
		}, nil
	} else if C.cef_v8_value_is_bool(ptr) == C.int(1) {
		return &BooleanValue{
			value{ptr},
		}, nil
	} else if C.cef_v8_value_is_int(ptr) == C.int(1) {
		return &IntValue{
			value{ptr},
		}, nil
	} else if C.cef_v8_value_is_uint(ptr) == C.int(1) {
		// checked before is_double: every uint also reports is_double
		return &UintValue{
			value{ptr},
		}, nil
	} else if C.cef_v8_value_is_double(ptr) == C.int(1) {
		return &DoubleValue{
			value{ptr},
		}, nil
	} else if C.cef_v8_value_is_date(ptr) == C.int(1) {
		return &DateValue{
			value{ptr},
		}, nil
	} else if C.cef_v8_value_is_string(ptr) == C.int(1) {
		return &StringValue{
			value{ptr},
		}, nil
	} else if C.cef_v8_value_is_function(ptr) == C.int(1) {
		return &FunctionValue{
			value{ptr},
		}, nil
	} else if C.cef_v8_value_is_array(ptr) == C.int(1) {
		return &ArrayValue{
			value:  value{ptr},
			values: map[int]Value{},
		}, nil
	} else if C.cef_v8_value_is_object(ptr) == C.int(1) {
		return &ObjectValue{
			value:      value{ptr},
			properties: map[string]Value{},
		}, nil
	} else {
		releaseBase(unsafe.Pointer(ptr))
		return nil, errors.New("unknown type")
	}

}

type (
	Value interface {
		fmt.Stringer
		io.Closer

		//	Native() (any, error)
		native() *C.cef_v8_value_t
	}

	ExportedFunction struct {
		FunctionName string
	}

	value struct {
		ptr *C.cef_v8_value_t
	}

	UndefinedValue struct {
		value
	}

	NullValue struct {
		value
	}

	BooleanValue struct {
		value
	}
	IntValue struct {
		value
	}
	UintValue struct {
		value
	}
	DoubleValue struct {
		value
	}
	StringValue struct {
		value
	}
	DateValue struct {
		value
	}

	FunctionValue struct {
		value
	}

	ObjectValue struct {
		value

		properties map[string]Value
	}

	ArrayValue struct {
		value
		values map[int]Value
	}

	Exporter struct {
		rootValue  Value
		repository map[uintptr]struct{}
	}
)

func NewExporter(val Value) *Exporter {
	return &Exporter{
		rootValue:  val,
		repository: map[uintptr]struct{}{},
	}
}

func (exp *Exporter) export(val Value, ttl int) (*Exported, error) {
	if val == nil {
		return &Exported{Undefined: true}, nil
	}

	// a pointer we exported before means a circular reference - emit a
	// marker instead of a nil hole (gob cannot encode nil entries in
	// slices/maps)
	valPtr := uintptr(unsafe.Pointer(val.native()))
	if _, exists := exp.repository[valPtr]; exists {
		return &Exported{Cycle: true}, nil
	}

	exp.repository[valPtr] = struct{}{}
	switch v := val.(type) {
	case *NullValue:
		return &Exported{Nil: true}, nil
	case *UndefinedValue:
		return &Exported{Undefined: true}, nil
	case *DateValue:
		return &Exported{Time: ptr(v.Date())}, nil
	case *StringValue:
		strVal := v.StringValue()
		return &Exported{Kind: exportedString, String: ptr(strVal)}, nil
	case *IntValue:
		return &Exported{Kind: exportedInt, Int: ptr(v.Integer())}, nil
	case *UintValue:
		return &Exported{Kind: exportedUint, UInt: ptr(v.UnsignedInteger())}, nil
	case *BooleanValue:
		return &Exported{Kind: exportedBool, Bool: ptr(v.Boolean())}, nil
	case *DoubleValue:
		return &Exported{Kind: exportedFloat, Float: ptr(v.Float())}, nil
	case *ArrayValue:
		ret := &Exported{IsArray: true}
		if ttl <= 0 {
			return ret, nil
		}
		length := v.Length()
		ret.Array = make([]*Exported, length)
		for idx := range length {
			if subValue, subErr := v.ValueForIndex(idx); subErr != nil {
				return nil, subErr
			} else {
				if exported, exportErr := exp.export(subValue, ttl-1); exportErr != nil {
					return nil, exportErr
				} else {
					ret.Array[idx] = exported
				}
			}
		}
		return ret, nil
	case *FunctionValue:
		return &Exported{Function: ptr(v.Name())}, nil
	case *ObjectValue:
		ret := &Exported{IsObject: true, Map: make(map[string]*Exported)}
		if ttl <= 0 {
			return ret, nil
		}
		for _, key := range v.Keys() {
			if subValue, subErr := v.ValueForKey(key); subErr != nil {
				return nil, subErr
			} else {
				if exported, exportErr := exp.export(subValue, ttl-1); exportErr != nil {
					return nil, exportErr
				} else {
					ret.Map[key] = exported
				}
			}
		}

		return ret, nil
	default:
		return nil, fmt.Errorf("unknown type %T", val)
	}
}

func (exp *Exporter) Export(maxDepth int) (*Exported, error) {
	return exp.export(exp.rootValue, maxDepth)
}

func (v *value) Close() error {
	if v.ptr != nil {
		C.release_v8value(v.ptr)
		v.ptr = nil
	}
	return nil
}

func (v *value) native() *C.cef_v8_value_t {
	return v.ptr
}

func (v *NullValue) String() string {
	return "[NULL]"
}

func (v *NullValue) Native() (any, error) {
	return nil, nil
}

func (v *UndefinedValue) String() string {
	return "[UNDEFINED]"
}

func (v *UndefinedValue) Native() (any, error) {
	return nil, nil
}

func (v *BooleanValue) Boolean() bool {
	return int(C.js_to_bool(v.ptr)) == 1
}

func (v *BooleanValue) Native() (any, error) {
	return v.Boolean(), nil
}

func (v *BooleanValue) String() string {
	return fmt.Sprintf("%t", v.Boolean())
}

func (v *IntValue) Integer() int {
	return int(C.js_to_int(v.ptr))
}

func (v *IntValue) Native() (any, error) {
	return v.Integer(), nil
}

func (v *IntValue) String() string {
	return strconv.Itoa(v.Integer())
}

func (v *UintValue) UnsignedInteger() uint {
	return uint(C.js_to_uint(v.ptr))
}

func (v *UintValue) String() string {
	return strconv.Itoa(int(v.UnsignedInteger()))
}

func (v *UintValue) Native() (any, error) {
	return v.UnsignedInteger(), nil
}
func (v *DoubleValue) Float() float64 {
	return float64(C.js_to_double(v.ptr))
}

func (v *DoubleValue) String() string {
	return fmt.Sprintf("%f", v.Float())
}

func (v *DoubleValue) Native() (any, error) {
	return v.Float(), nil
}

func (v *StringValue) String() string {
	str := v.StringValue()
	return fmt.Sprintf("[String:%s]", str)
}
func (v *StringValue) StringValue() string {
	return cefUserfreeToGoString(C.cef_v8_value_get_string(v.ptr))
}

func (v *StringValue) Native() (any, error) {
	return v.StringValue(), nil
}

func (v *DateValue) Date() time.Time {
	// date
	var (
		d C.double
	)
	t := C.js_to_date(v.ptr)
	C.cef_time_to_doublet(&t, &d)
	floatSeconds := float64(d)

	return time.Unix(int64(floatSeconds), int64((floatSeconds-float64(int64(floatSeconds)))*1e9))
}

func (v *DateValue) String() string {
	return v.Date().String()
}

func (v *DateValue) Native() (any, error) {
	return v.Date(), nil
}

func (v *ObjectValue) Close() error {

	for _, property := range v.properties {
		if property != nil {
			_ = property.Close()
		}
	}
	v.properties = map[string]Value{}

	return v.value.Close()
}

func (v *ObjectValue) ValueForKey(key string) (Value, error) {

	if val, exists := v.properties[key]; exists {
		return val, nil
	}

	cKey := C.CString(key)
	defer C.free(unsafe.Pointer(cKey))

	subValuePtr := C.getValueByKey(v.ptr, cKey)

	if val, err := valueFromPointer(subValuePtr); err != nil {
		return nil, err
	} else {
		v.properties[key] = val
		return val, nil
	}
}

func (v *ObjectValue) SetValueForKey(key string, value Value) error {

	cKey := C.CString(key)
	defer C.free(unsafe.Pointer(cKey))

	// set_value_bykey consumes a reference; add one so the caller's wrapper
	// stays valid. It is consumed even when the assignment fails - CEF
	// unwraps the argument before it calls into V8 - so a failure must not
	// release it a second time.
	addRefBase(unsafe.Pointer(value.native()))
	if success := C.setValueByKey(v.ptr, cKey, value.native()); success == C.int(0) {
		return errors.New("could not assign value to key")
	}

	v.properties[key] = value

	return nil
}

func (v *ObjectValue) HasValueForKey(key string) bool {

	if _, exists := v.properties[key]; exists {
		return true
	}

	cKey := C.CString(key)
	defer C.free(unsafe.Pointer(cKey))

	valueExists := C.hasValueByKey(v.ptr, cKey)
	return valueExists == C.int(1)
}

func (v *ObjectValue) String() string {
	return fmt.Sprintf("[OBJECT %v]", strings.Join(v.Keys(), ","))
}

func (v *ObjectValue) Keys() (keys []string) {
	var keyStringList C.cef_string_list_t
	keyStringList = C.cef_string_list_alloc()
	defer C.cef_string_list_free(keyStringList)

	C.js_get_object_keys(v.ptr, keyStringList)

	var index C.size_t

	for index = 0; index < C.cef_string_list_size(keyStringList); index++ {
		var str C.cef_string_t
		if C.cef_string_list_value(keyStringList, index, &str) == 0 {
			continue
		}
		keys = append(keys, cefToGoString(&str))
		// cef_string_list_value copies - the buffer must be released
		C.cef_string_clear(&str)
	}
	return
}

func (v *FunctionValue) String() string {
	return fmt.Sprintf("[FUNCTION:%s]", v.Name())
}

func (v *FunctionValue) Name() string {
	return cefUserfreeToGoString(C.cef_v8_value_get_function_name(v.ptr))
}

func (v *FunctionValue) Native() (any, error) {
	return nil, nil
}

func (v *ArrayValue) String() string {
	return fmt.Sprintf("[ARRAY:%d]", v.Length())
}

func (v *ArrayValue) Length() int {
	return int(C.cef_v8_value_get_array_length(v.ptr))
}

func (v *ArrayValue) Close() error {
	for _, val := range v.values {
		if val != nil {
			_ = val.Close()
		}
	}
	v.values = map[int]Value{}

	return v.value.Close()
}

func (v *ArrayValue) ValueForIndex(index int) (Value, error) {
	if val, exists := v.values[index]; exists {
		return val, nil
	}

	subValuePtr := C.cef_v8_value_get_value_byindex(v.ptr, C.int(index))

	if val, err := valueFromPointer(subValuePtr); err != nil {
		return nil, err
	} else {
		v.values[index] = val
		return val, nil
	}
}

func (v *ArrayValue) Native() (any, error) {
	return nil, nil
}

func CreateObject() *ObjectValue {
	return &ObjectValue{
		value: value{
			ptr: C.cef_v8_value_create_object(nil, nil),
		},
		properties: map[string]Value{},
	}
}
