//go:build windows

package gomium

import (
	"syscall"

	cef "github.com/timzifer/gomium/internal/cef"
)

// Windows-only surface.

func GetModuleHandle(modname *uint16) (syscall.Handle, error) { return cef.GetModuleHandle(modname) }

func WithModuleHandle(handle syscall.Handle) Option { return cef.WithModuleHandle(handle) }

func WithMultiThreadedMessageLoop(state bool) Option { return cef.WithMultiThreadedMessageLoop(state) }
