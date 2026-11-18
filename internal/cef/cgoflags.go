package cef

// The C include paths for the whole package. cgo unions #cgo directives across
// every cgo file in a package, so stating them once here replaces the identical
// line that used to sit in 38 preambles - a path change is one edit.
//
// -I${SRCDIR} is this directory, where the bridge headers live. It is
// redundant in itself, since `go build` already invokes the compiler as
// `gcc -I .` with the working directory set to the package directory, but it
// anchors the second path and survives a compiler invoked from elsewhere.
//
// -I${SRCDIR}/../.. is the module root, and it is not redundant: it is what
// resolves the `#include "include/capi/..."` lines in the bridge headers
// against the vendored CEF headers in include/. Without it every bridge
// header fails to compile.
//
// -DCEF_API_VERSION pins the CEF client API to 15600 (CEF 156). Without it the
// headers default to CEF_API_VERSION_EXPERIMENTAL, which lets every release
// change the ABI under the binding (see docs/upgrading-cef.md, section 5).

/*
#cgo CFLAGS: -I${SRCDIR} -I${SRCDIR}/../.. -DCEF_API_VERSION=15600
*/
import "C"
