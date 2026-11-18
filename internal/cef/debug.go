package cef

import "C"

import (
	"fmt"
	"github.com/rs/zerolog"
	"unsafe"
)

var (
	debugLogger = zerolog.Nop()
)

func SetDebugLogger(zLogger zerolog.Logger) {
	debugLogger = zLogger
}

//export goLogFuncCall
func goLogFuncCall(file *C.char, line C.int, funcName *C.char) {
	debugLogger.Debug().Str("file", C.GoString(file)).Int("line", (int)(line)).Str("function", C.GoString(funcName)).Msg(">>>")
}

//export goLogFuncExit
func goLogFuncExit(file *C.char, line C.int, funcName *C.char) {
	debugLogger.Debug().Str("file", C.GoString(file)).Int("line", (int)(line)).Str("function", C.GoString(funcName)).Msg("<<<")
}

//export goLogMessage
func goLogMessage(file *C.char, line C.int, funcName, message *C.char) {
	debugLogger.Debug().Str("file", C.GoString(file)).Int("line", (int)(line)).Str("function", C.GoString(funcName)).Msg(C.GoString(message))
}

//export goLogMessageAndDie
func goLogMessageAndDie(file *C.char, line C.int, funcName, message *C.char) {
	debugLogger.Fatal().Str("file", C.GoString(file)).Int("line", (int)(line)).Str("function", C.GoString(funcName)).Msg(C.GoString(message))
}

//export goLogPointer
func goLogPointer(file *C.char, line C.int, funcName, message *C.char, ptr unsafe.Pointer) {
	debugLogger.Debug().Str("file", C.GoString(file)).Int("line", (int)(line)).Str("function", C.GoString(funcName)).Str("ptr", fmt.Sprintf("0x%x", ptr)).Msg(C.GoString(message))
}
