package gomium

import "github.com/timzifer/gomium/internal/cefshim"

// LoadLibcef loads the CEF library from path and binds every CEF entry point
// this package calls. It must run before Run or ExecuteSubprocess - on the
// browser process and on every CEF helper process (--type=...), since helpers
// are fresh instances of the same executable. Idempotent: the first call
// decides, later calls return that first result even when it was an error.
//
// path names the shared library itself (linux: Release/libcef.so, windows:
// Release\libcef.dll) or, on macOS, the framework bundle - both
//
//	Release/Chromium Embedded Framework.framework
//	Release/Chromium Embedded Framework.framework/Chromium Embedded Framework
//
// are accepted.
func LoadLibcef(path string) error { return cefshim.Load(path) }
