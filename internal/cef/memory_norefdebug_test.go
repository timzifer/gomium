//go:build !refdebug

package cef

import (
	"testing"
	"unsafe"
)

// Without the ledger an add_ref on a pointer gomium does not know must not
// register it: a later release would then free memory that is not (or no
// longer) ours.
func TestAddRefOnUnknownPointerDoesNotRevive(t *testing.T) {
	unknown := unsafe.Pointer(new([64]byte))
	if n := goMemoryAddRef(unknown); n != 0 {
		t.Fatalf("add_ref on an unknown pointer: count %d, want 0", n)
	}
	refCountLock.Lock()
	_, revived := memoryBridge[unknown]
	refCountLock.Unlock()
	if revived {
		t.Fatal("add_ref on an unknown pointer registered it")
	}
}
