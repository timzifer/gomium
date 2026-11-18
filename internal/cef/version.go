package cef

/*
#include "app_bridge.h"
*/
import "C"

// CEFVersion returns the full CEF version string this binary was compiled
// against, e.g. "151.3.14+g5d67476+chromium-151.0.7922.72". It reflects the
// headers used at build time and does not touch the loaded library, so it is
// safe to call at any point - including before Run and against a mismatched
// libcef.
func CEFVersion() string {
	return C.GoString(C.cAppCompiledCefVersion())
}
