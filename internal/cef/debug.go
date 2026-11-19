package cef

import "C"

import (
	"log/slog"
	"unsafe"
)

var (
	debugLogger = nopLogger()
)

// SetDebugLogger sets the logger of the C bridge tracing (debug.h). nil
// switches it off. Set it before Run: the bridge reads it without a lock.
func SetDebugLogger(logger *slog.Logger) {
	debugLogger = orNop(logger)
}

//export goLogFuncCall
func goLogFuncCall(file *C.char, line C.int, funcName *C.char) {
	debugLogger.Debug(">>>", "file", C.GoString(file), "line", (int)(line), "function", C.GoString(funcName))
}

//export goLogFuncExit
func goLogFuncExit(file *C.char, line C.int, funcName *C.char) {
	debugLogger.Debug("<<<", "file", C.GoString(file), "line", (int)(line), "function", C.GoString(funcName))
}

//export goLogMessage
func goLogMessage(file *C.char, line C.int, funcName, message *C.char) {
	debugLogger.Debug(C.GoString(message), "file", C.GoString(file), "line", (int)(line), "function", C.GoString(funcName))
}

//export goLogMessageAndDie
func goLogMessageAndDie(file *C.char, line C.int, funcName, message *C.char) {
	fatal(debugLogger, C.GoString(message), "file", C.GoString(file), "line", (int)(line), "function", C.GoString(funcName))
}

//export goLogPointer
func goLogPointer(file *C.char, line C.int, funcName, message *C.char, ptr unsafe.Pointer) {
	debugLogger.Debug(C.GoString(message), "file", C.GoString(file), "line", (int)(line), "function", C.GoString(funcName), "ptr", hexPtr(uintptr(ptr)))
}
