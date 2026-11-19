//go:build linux || darwin

package cef

/*
#include <stdlib.h>
*/
import "C"
import (
	"os"
	"unsafe"
)

func (c *cefApp) initializeOperatingSystemSpecificMainArgs() {
	// argv must live in C memory - storing go pointers in the C-allocated
	// cef_main_args_t violates the cgo pointer rules and would dangle as
	// soon as this function returns.
	argc := len(os.Args)
	argv := (**C.char)(C.calloc(C.size_t(argc), C.size_t(unsafe.Sizeof(uintptr(0)))))
	argvSlice := unsafe.Slice(argv, argc)
	for i, arg := range os.Args {
		argvSlice[i] = C.CString(arg)
	}
	c.mainArgs.argc = C.int(argc)
	c.mainArgs.argv = argv
	trace(c.options.logger, "operating-system specific main args initialized", "argc", argc)
}

func (c *cefApp) freeOperatingSystemSpecificMainArgs() {
	if c.mainArgs == nil || c.mainArgs.argv == nil {
		return
	}
	argvSlice := unsafe.Slice(c.mainArgs.argv, int(c.mainArgs.argc))
	for _, arg := range argvSlice {
		C.free(unsafe.Pointer(arg))
	}
	C.free(unsafe.Pointer(c.mainArgs.argv))
	c.mainArgs.argv = nil
	c.mainArgs.argc = 0
}
