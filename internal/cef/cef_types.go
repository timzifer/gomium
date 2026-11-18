package cef

/*
#include "include/capi/cef_values_capi.h"

*/
import "C"

type ValueType C.cef_value_type_t

const (
	ValueTypeInvalid    = ValueType(C.VTYPE_INVALID)
	ValueTypeNull       = ValueType(C.VTYPE_NULL)
	ValueTypeBool       = ValueType(C.VTYPE_BOOL)
	ValueTypeInt        = ValueType(C.VTYPE_INT)
	ValueTypeDouble     = ValueType(C.VTYPE_DOUBLE)
	ValueTypeString     = ValueType(C.VTYPE_STRING)
	ValueTypeBinary     = ValueType(C.VTYPE_BINARY)
	ValueTypeDictionary = ValueType(C.VTYPE_DICTIONARY)
	ValueTypeList       = ValueType(C.VTYPE_LIST)
)

func (ct ValueType) String() string {
	switch ct {
	default:
		return "VTYPE_INVALID"
	case ValueTypeNull:
		return "VTYPE_NULL"
	case ValueTypeBool:
		return "VTYPE_BOOL"
	case ValueTypeInt:
		return "VTYPE_INT"
	case ValueTypeDouble:
		return "VTYPE_DOUBLE"
	case ValueTypeString:
		return "VTYPE_STRING"
	case ValueTypeBinary:
		return "VTYPE_BINARY"
	case ValueTypeList:
		return "VTYPE_LIST"
	case ValueTypeDictionary:
		return "VTYPE_DICTIONARY"

	}
}
