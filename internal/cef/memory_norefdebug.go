//go:build !refdebug

package cef

import "unsafe"

// Without the refdebug tag the ledger only counts; see memory_refdebug.go.

func ledgerRegister(unsafe.Pointer)      {}
func ledgerForget(unsafe.Pointer)        {}
func ledgerFault(string, unsafe.Pointer) {}
func checkRefLedger() error              { return nil }
