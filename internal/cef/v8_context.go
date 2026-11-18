package cef

/*
#include "debug.h"
#include "v8.h"
*/
import "C"
import (
	"net/url"
	"runtime"
	"unsafe"
)

type (
	v8Context struct {
		cStruct *C.cef_v8_context_t
	}

	V8Context interface {
		IsValid() bool
		// Browser, Frame and TaskRunner return owned references - Release
		// them when done.
		Browser() CefBrowser
		Frame() CefFrame
		Enter() bool
		Exit() bool
		Equals(V8Context) bool
		Eval(code string, sourceUrl *url.URL, startLine int) (*Exported, *ExceptionError)
		TaskRunner() CefTaskRunner

		ptr() *C.cef_v8_context_t
	}
)

func (v v8Context) ptr() *C.cef_v8_context_t {
	return v.cStruct
}

func (v v8Context) TaskRunner() CefTaskRunner {
	return cefTaskRunner{C.v8context_get_task_runner(v.cStruct)}
}

func (v v8Context) IsValid() bool {
	if v.cStruct == nil {
		return false
	}

	return C.v8context_is_valid(v.cStruct) == 1
}

func (v v8Context) Browser() CefBrowser {
	theBrowser := C.v8context_get_browser(v.cStruct)
	if nil != theBrowser {
		return &cefBrowser{theBrowser}
	}
	return nil
}

func (v v8Context) Frame() CefFrame {
	theFrame := C.v8context_get_frame(v.cStruct)
	if nil != theFrame {
		return &cefFrame{theFrame}
	}
	return nil
}

func (v v8Context) Enter() bool {
	return C.v8context_enter(v.cStruct) != 0
}

func (v v8Context) Exit() bool {
	return C.v8context_exit(v.cStruct) != 0
}

func (v v8Context) Equals(that V8Context) bool {
	if that == nil {
		return false
	}
	return C.v8context_is_same(v.cStruct, that.ptr()) != 0
}

func (v v8Context) Eval(source string, sourceUrl *url.URL, startLine int) (*Exported, *ExceptionError) {

	var (
		sourceUrlString *C.cef_string_t
		returnValue     *C.cef_v8_value_t
		exception       *C.cef_v8_exception_t
	)

	if sourceUrl != nil {
		sourceUrlString = cefString(sourceUrl.String())
		defer cefStringFree(sourceUrlString)
	}
	sourceString := cefString(source)
	defer cefStringFree(sourceString)

	runtime.LockOSThread()
	entered := v.Enter()
	defer func() {
		if entered {
			_ = v.Exit()
		}
		runtime.UnlockOSThread()
	}()

	// we are on the renderer, execute directly
	result := C.v8context_eval(v.cStruct, sourceString, sourceUrlString, C.int(startLine), &returnValue, &exception)
	if result != C.int(1) {
		if exception != nil {
			defer releaseBase(unsafe.Pointer(exception))
			return nil, cefV8Exception{exception}.ToError()
		}
		return nil, &ExceptionError{Message: "evaluation failed without exception details"}
	}

	if returnValue == nil {
		return &Exported{Undefined: true}, nil
	}

	val, err := valueFromPointer(returnValue)
	if err != nil {
		return nil, &ExceptionError{Message: err.Error()}
	}
	if val == nil {
		return &Exported{Undefined: true}, nil
	}
	defer func() {
		_ = val.Close()
	}()

	exported, exportErr := NewExporter(val).Export(10)
	if exportErr != nil {
		return nil, &ExceptionError{Message: exportErr.Error()}
	}
	return exported, nil
}

var _ V8Context = (*v8Context)(nil)
