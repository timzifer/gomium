package cef

/*
#include <string.h>
#include "include/capi/cef_v8_capi.h"

cef_string_userfree_t v8exception_get_message(struct _cef_v8_exception_t* self) {
	return self -> get_message(self);
}

cef_string_userfree_t v8exception_get_source_line(struct _cef_v8_exception_t* self) {
	return self -> get_source_line(self);
}

cef_string_userfree_t v8exception_get_script_resource_name(struct _cef_v8_exception_t* self) {
	return self -> get_script_resource_name(self);
}

int v8exception_get_line_number(struct _cef_v8_exception_t* self) {
	return self -> get_line_number(self);
}

int v8exception_get_start_position(struct _cef_v8_exception_t* self) {
	return self -> get_start_position(self);
}

int v8exception_get_end_position(struct _cef_v8_exception_t* self) {
	return self -> get_end_position(self);
}

int v8exception_get_start_column(struct _cef_v8_exception_t* self) {
	return self -> get_start_column(self);
}

int v8exception_get_end_column(struct _cef_v8_exception_t* self) {
	return self -> get_end_column(self);
}
*/
import "C"
import "fmt"

type (
	cefV8Exception struct {
		cStruct *C.cef_v8_exception_t
	}

	CefV8Exception interface {
		Message() string
		SourceLine() string
		ScriptResourceName() string

		LineNumber() int
		StartPosition() int
		EndPosition() int
		StartColumn() int
		EndColumn() int

		ToError() *ExceptionError
	}

	ExceptionError struct {
		Message            string
		SourceLine         string
		ScriptResourceName string

		LineNumber    int
		StartPosition int
		EndPosition   int
		StartColumn   int
		EndColumn     int
	}
)

func (e ExceptionError) Error() string {
	return fmt.Sprintf("[%d:%d] %s", e.StartPosition, e.StartColumn, e.Message)
}

func (c cefV8Exception) ToError() *ExceptionError {
	return &ExceptionError{
		Message:            c.Message(),
		SourceLine:         c.SourceLine(),
		ScriptResourceName: c.ScriptResourceName(),
		LineNumber:         c.LineNumber(),
		StartPosition:      c.StartPosition(),
		EndPosition:        c.EndPosition(),
		StartColumn:        c.StartColumn(),
		EndColumn:          c.EndColumn(),
	}
}

func (c cefV8Exception) Message() string {
	return cefUserfreeToGoString(C.v8exception_get_message(c.cStruct))
}

func (c cefV8Exception) SourceLine() string {
	return cefUserfreeToGoString(C.v8exception_get_source_line(c.cStruct))
}

func (c cefV8Exception) ScriptResourceName() string {
	return cefUserfreeToGoString(C.v8exception_get_script_resource_name(c.cStruct))
}

func (c cefV8Exception) LineNumber() int {
	return int(C.v8exception_get_line_number(c.cStruct))
}

func (c cefV8Exception) StartPosition() int {
	return int(C.v8exception_get_start_position(c.cStruct))
}

func (c cefV8Exception) EndPosition() int {
	return int(C.v8exception_get_end_position(c.cStruct))
}

func (c cefV8Exception) StartColumn() int {
	return int(C.v8exception_get_start_column(c.cStruct))
}

func (c cefV8Exception) EndColumn() int {
	return int(C.v8exception_get_end_column(c.cStruct))
}

var (
	_ CefV8Exception = (*cefV8Exception)(nil)
	_ error          = (*ExceptionError)(nil)
)
