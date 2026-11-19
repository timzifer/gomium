package cef

/*
#include <stdlib.h>
#include "memory.h"

*/
import "C"
import (
	"context"
	"log/slog"
	"sync"
	"unsafe"
)

var (
	memoryLogger = nopLogger()
	memoryBridge = make(map[unsafe.Pointer]int)
	refCountLock sync.Mutex
)

// SetMemoryLogger sets the logger of the reference counting, which logs at
// LevelTrace. nil switches it off. The goMemory* callbacks read it under
// refCountLock, so it is written under it too.
func SetMemoryLogger(logger *slog.Logger) {
	refCountLock.Lock()
	defer refCountLock.Unlock()
	memoryLogger = orNop(logger)
}

// traceRef logs one reference operation. It runs on every add_ref/release,
// so nothing is built unless the logger takes LevelTrace.
func traceRef(op string, it unsafe.Pointer, count int) {
	if memoryLogger.Enabled(context.Background(), LevelTrace) {
		trace(memoryLogger, op, "ptr", hexPtr(uintptr(it)), "count", count)
	}
}

// addRefBase / releaseBase are the untyped convenience variants used by the
// callback layer, where pointers arrive as concrete CEF struct types.
func addRefBase(p unsafe.Pointer) {
	if p != nil {
		C.base_add_ref((*C.cef_base_ref_counted_t)(p))
	}
}

func releaseBase(p unsafe.Pointer) {
	if p != nil {
		C.base_release_ref((*C.cef_base_ref_counted_t)(p))
	}
}

// goMemoryRegister takes the first reference on a struct that
// initialize_cef_base_ref_counted just set up.
//
//export goMemoryRegister
func goMemoryRegister(it unsafe.Pointer) {
	refCountLock.Lock()
	defer refCountLock.Unlock()
	if memoryBridge[it] != 0 {
		ledgerFault("register of a live pointer", it)
	}
	memoryBridge[it] = 1
	ledgerRegister(it)
	traceRef("+++", it, 1)
}

//export goMemoryAddRef
func goMemoryAddRef(it unsafe.Pointer) C.int {
	refCountLock.Lock()
	defer refCountLock.Unlock()
	m := memoryBridge[it]
	if m == 0 {
		// a struct that was freed (or never ours): use after free. Do not
		// bring it back - a later release would free it a second time.
		memoryLogger.Warn("add_ref on unknown pointer", "ptr", hexPtr(uintptr(it)))
		ledgerFault("add_ref on an unknown or freed pointer", it)
		return 0
	}
	m++

	traceRef("+++", it, m)

	memoryBridge[it] = m
	return C.int(m)
}

// goMemoryRelease implements the cef_base_ref_counted_t release() contract:
// it returns 1 only when the reference count reaches 0 and the object has
// been destroyed.
//
//export goMemoryRelease
func goMemoryRelease(it unsafe.Pointer) C.int {
	if !releaseCounted(it) {
		return 0
	}
	// The last reference is gone. Drop the side registries keyed by the
	// struct pointer outside refCountLock - they take locks of their own,
	// and holding refCountLock across them would fix a lock order that any
	// release under one of those locks would invert. The address cannot be
	// reused before C.free below, so no new entry can collide.
	forgetResourceServe(it)
	forgetCookiePending(it)
	forgetPdfPending(it)
	C.free(it)
	return 1
}

// releaseCounted drops one reference. It reports true when that was the last
// one; the pointer is then gone from memoryBridge and the caller frees it.
func releaseCounted(it unsafe.Pointer) bool {
	refCountLock.Lock()
	defer refCountLock.Unlock()

	m, ok := memoryBridge[it]
	if !ok {
		memoryLogger.Warn("release on unknown pointer", "ptr", hexPtr(uintptr(it)))
		ledgerFault("release on an unknown or freed pointer", it)
		return false
	}
	m--
	if m <= 0 {
		traceRef("###", it, 0)
		delete(memoryBridge, it)
		ledgerForget(it)
		return true
	}
	traceRef("---", it, m)
	memoryBridge[it] = m
	return false
}

//export goMemoryHasOneRef
func goMemoryHasOneRef(it unsafe.Pointer) C.int {
	refCountLock.Lock()
	defer refCountLock.Unlock()

	cnt := memoryBridge[it]
	traceRef("???", it, cnt)
	if cnt == 1 {
		return 1
	}
	return 0
}

//export goMemoryHasAtLeastOneRef
func goMemoryHasAtLeastOneRef(it unsafe.Pointer) C.int {
	refCountLock.Lock()
	defer refCountLock.Unlock()

	if memoryBridge[it] >= 1 {
		return 1
	}
	return 0
}
