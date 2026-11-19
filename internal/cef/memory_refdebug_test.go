//go:build refdebug

package cef

import (
	"strings"
	"testing"
	"unsafe"
)

func expectLedgerPanic(t *testing.T, want string, op func()) {
	t.Helper()
	defer func() {
		r := recover()
		msg, _ := r.(string)
		if !strings.Contains(msg, want) {
			t.Fatalf("want panic containing %q, got %v", want, r)
		}
	}()
	op()
}

func TestRefLedgerFaults(t *testing.T) {
	// Go memory stands in for a struct: the ledger never frees it as long as
	// the count stays above zero.
	live := unsafe.Pointer(new([64]byte))
	unknown := unsafe.Pointer(new([64]byte))

	goMemoryRegister(live)
	t.Cleanup(func() {
		refCountLock.Lock()
		delete(memoryBridge, live)
		ledgerForget(live)
		refCountLock.Unlock()
	})

	if n := goMemoryAddRef(live); n != 2 {
		t.Fatalf("add_ref on a live struct: count %d, want 2", n)
	}
	if goMemoryRelease(live) != 0 {
		t.Fatal("release above zero reported the struct as freed")
	}

	expectLedgerPanic(t, "add_ref on an unknown or freed pointer", func() { goMemoryAddRef(unknown) })
	expectLedgerPanic(t, "release on an unknown or freed pointer", func() { goMemoryRelease(unknown) })
	expectLedgerPanic(t, "register of a live pointer", func() { goMemoryRegister(live) })

	err := checkRefLedger()
	if err == nil || !strings.Contains(err.Error(), "refs=1") {
		t.Fatalf("checkRefLedger with a live struct: %v", err)
	}
}
