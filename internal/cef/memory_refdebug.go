//go:build refdebug

package cef

import (
	"errors"
	"fmt"
	"runtime"
	"sort"
	"strings"
	"unsafe"
)

// The refdebug build turns the reference ledger of the structs gomium
// allocates itself (memoryBridge) into a checker: every struct remembers
// where it was allocated, a reference operation on a pointer the ledger does
// not know - a struct that was already freed - panics with both stacks, and
// CefApp.Close fails with the list of structs still alive once CEF has shut
// down and gomium has dropped its own references. Build with -tags refdebug.

// allocStacks and freeStacks are guarded by refCountLock, like memoryBridge.
// freeStacks keeps where a struct was freed until its address is reused.
var (
	allocStacks = map[unsafe.Pointer][]uintptr{}
	freeStacks  = map[unsafe.Pointer][]uintptr{}
)

func callers() []uintptr {
	pcs := make([]uintptr, 32)
	// skip runtime.Callers, callers, the ledger function and its goMemory* caller
	n := runtime.Callers(4, pcs)
	return pcs[:n]
}

func ledgerRegister(p unsafe.Pointer) {
	allocStacks[p] = callers()
	delete(freeStacks, p)
}

func ledgerForget(p unsafe.Pointer) {
	delete(allocStacks, p)
	freeStacks[p] = callers()
}

func ledgerFault(what string, p unsafe.Pointer) {
	msg := fmt.Sprintf("refdebug: %s 0x%x", what, uintptr(p))
	if freed, ok := freeStacks[p]; ok {
		msg += "\nfreed at:\n" + formatStack(freed)
	}
	panic(msg)
}

func formatStack(pcs []uintptr) string {
	var b strings.Builder
	frames := runtime.CallersFrames(pcs)
	for {
		frame, more := frames.Next()
		if !strings.HasPrefix(frame.Function, "runtime.") && !strings.HasPrefix(frame.Function, "_cgo") {
			fmt.Fprintf(&b, "\t%s\n\t\t%s:%d\n", frame.Function, frame.File, frame.Line)
		}
		if !more {
			break
		}
	}
	return b.String()
}

// errRefLedger reports the structs that outlived CEF.
var errRefLedger = errors.New("refdebug: structs still referenced after shutdown")

// checkRefLedger lists every struct the ledger still holds. Called by Close
// after cef_shutdown and freeNativeStructs, when nothing of gomium's may be
// left.
func checkRefLedger() error {
	refCountLock.Lock()
	defer refCountLock.Unlock()
	if len(memoryBridge) == 0 {
		return nil
	}
	leaks := make([]string, 0, len(memoryBridge))
	for p, count := range memoryBridge {
		leaks = append(leaks, fmt.Sprintf("0x%x refs=%d allocated at:\n%s", uintptr(p), count, formatStack(allocStacks[p])))
	}
	sort.Strings(leaks)
	return fmt.Errorf("%w (%d):\n%s", errRefLedger, len(leaks), strings.Join(leaks, "\n"))
}
