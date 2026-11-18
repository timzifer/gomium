package cefdl

import (
	_ "embed"
	"strings"
)

// pinnedVersionFile is the CEF version this gomium module version is built
// against. It lives next to this file because go:embed cannot reach outside
// the package directory - and embedding is the point: tools/cefdl runs from a
// consumer's module just as well as from a gomium checkout, where a path
// relative to the working directory would point into the consumer's tree.
//
//go:embed cef.version
var pinnedVersionFile string

// PinnedVersion returns the CEF version pinned in cefdl/cef.version of the
// gomium module this package was compiled from. It must match the vendored
// headers in include/, so it is also what gomium.CEFVersion() reports - but
// unlike that, it is available without cgo.
func PinnedVersion() string {
	return strings.TrimSpace(pinnedVersionFile)
}
