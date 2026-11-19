package cef

/*
#include <stdlib.h>
#include "memory.h"

*/
import "C"
import (
	"fmt"
	"github.com/rs/zerolog"
	"sync"
	"unsafe"
)

var (
	memoryLogger = zerolog.Nop()
	memoryBridge = make(map[unsafe.Pointer]int)
	refCountLock sync.Mutex
)

func SetMemoryLogger(zLogger zerolog.Logger) {
	memoryLogger = zLogger
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
	memoryLogger.Trace().Str("ptr", fmt.Sprintf("0x%x", it)).Int("count", 1).Msg("+++")
}

//export goMemoryAddRef
func goMemoryAddRef(it unsafe.Pointer) C.int {
	refCountLock.Lock()
	defer refCountLock.Unlock()
	m := memoryBridge[it]
	if m == 0 {
		// a struct that was freed (or never ours): use after free
		ledgerFault("add_ref on an unknown or freed pointer", it)
	}
	m++

	memoryLogger.Trace().Str("ptr", fmt.Sprintf("0x%x", it)).Int("count", m).Msg("+++")

	memoryBridge[it] = m
	return C.int(m)
}

// goMemoryRelease implements the cef_base_ref_counted_t release() contract:
// it returns 1 only when the reference count reaches 0 and the object has
// been destroyed.
//
//export goMemoryRelease
func goMemoryRelease(it unsafe.Pointer) C.int {
	refCountLock.Lock()
	defer refCountLock.Unlock()

	if m, ok := memoryBridge[it]; ok {
		m--
		if m <= 0 {
			memoryLogger.Trace().Str("ptr", fmt.Sprintf("0x%x", it)).Msg("###")
			delete(memoryBridge, it)
			ledgerForget(it)
			// drop side registries keyed by the struct pointer before the
			// memory is reused
			forgetResourceServe(it)
			forgetCookiePending(it)
			forgetPdfPending(it)
			C.free(it)
			return 1
		}
		memoryLogger.Trace().Str("ptr", fmt.Sprintf("0x%x", it)).Int("count", m).Msg("---")
		memoryBridge[it] = m
		return 0
	}
	memoryLogger.Warn().Str("ptr", fmt.Sprintf("0x%x", it)).Msg("release on unknown pointer")
	ledgerFault("release on an unknown or freed pointer", it)
	return 0
}

//export goMemoryHasOneRef
func goMemoryHasOneRef(it unsafe.Pointer) C.int {
	refCountLock.Lock()
	defer refCountLock.Unlock()

	cnt := memoryBridge[it]
	memoryLogger.Trace().Str("ptr", fmt.Sprintf("0x%x", it)).Int("count", cnt).Msg("???")
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
