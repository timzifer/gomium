package pkg

/*
#include "bridge.h"

static int preamble_consume(cef_frame_t* self, cef_frame_t* other REF_CONSUMED) {
	return 0;
}
*/
import "C"
import "unsafe"

func releaseBase(unsafe.Pointer) {}
func addRefBase(unsafe.Pointer)  {}

type wrapper struct{ ptr *C.cef_frame_t }

func (w wrapper) native() *C.cef_frame_t { return w.ptr }

// adopt takes the frame over.
//
// refcheck:adopts frame
func adopt(frame *C.cef_frame_t) wrapper { return wrapper{frame} }

// make returns an owned frame.
//
// refcheck:fresh
func makeFrame() *C.cef_frame_t { return C.frame_create() }

//export goReleased
func goReleased(self *C.cef_frame_t, frame *C.cef_frame_t, rect *C.cef_rect_t) {
	defer releaseBase(unsafe.Pointer(frame))
}

//export goLeaked
func goLeaked(self *C.cef_frame_t,
	frame *C.cef_frame_t) { // want: goLeaked: parameter frame
	_ = frame
}

//export goDerivedLeaked
func goDerivedLeaked(self *C.cef_frame_t,
	context *C.struct__cef_request_context_t) { // want: goDerivedLeaked: parameter context
}

//export goWrapped
func goWrapped(self *C.cef_frame_t, frame *C.cef_frame_t, other *C.cef_frame_t) {
	w := wrapper{ptr: frame}
	_ = w
	_ = adopt(other)
}

//export goHandedOff
func goHandedOff(self *C.cef_frame_t, frame *C.cef_frame_t, browser *C.cef_browser_t) {
	// refcheck:handoff frame the registry releases it
	keep(unsafe.Pointer(frame))
	C.frame_adopt(self, (*C.cef_frame_t)(unsafe.Pointer(browser)))
}

func keep(unsafe.Pointer) {}

func notExported(self *C.cef_frame_t, frame *C.cef_frame_t) {}

func consumers(f *C.cef_frame_t, w wrapper, ctx *C.cef_request_context_t) {
	C.frame_is_same(f, w.native()) // borrowed: fine
	C.frame_set_rect(f, nil)

	C.frame_adopt(f, w.native()) // want: frame_adopt consumes a reference to w.native()

	addRefBase(unsafe.Pointer(w.ptr))
	C.frame_adopt(f, w.ptr)
	C.frame_adopt(f, w.ptr) // want: frame_adopt consumes a reference to w.ptr

	C.frame_adopt(f, C.frame_create())
	C.frame_adopt(f, makeFrame())
	fresh := C.frame_create()
	C.frame_adopt(f, fresh)
	var owned = makeFrame()
	C.preamble_consume(f, owned)
	C.frame_adopt(f, nil)

	// refcheck:owned the caller gave its reference away
	C.frame_adopt(f, w.ptr)

	C.cef_post_frame(f, nil) // want: cef_post_frame consumes a reference to f
	C.context_adopt(f, ctx)  // want: context_adopt consumes a reference to ctx
}
