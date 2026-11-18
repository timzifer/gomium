package gomium

import (
	"time"

	cef "github.com/timzifer/gomium/internal/cef"
)

// Sentinel errors of the binding. Re-exported by value: they are read-only,
// so the alias keeps error identity and errors.Is comparisons intact.

var CefAppAlreadyExistsErr = cef.CefAppAlreadyExistsErr

var MessageIdentifierAlreadyRegisteredError = cef.MessageIdentifierAlreadyRegisteredError

var MessageIdentifierNotRegisteredError = cef.MessageIdentifierNotRegisteredError

var RequestInvalidatedErr = cef.RequestInvalidatedErr

// JavaScriptEvaluationTimeout bounds how long an ExecuteJavaScriptWithResult
// callback may stay pending. If no response arrives in time (e.g. the render
// process died), the callback is resolved with an error. Zero disables the
// timeout.
//
// Unlike everything else here this is a knob callers assign to, so it is
// declared here rather than re-exported: `var X = cef.X` would copy the value
// once at init and quietly drop every later assignment. The binding reads it
// through the pointer bound below.
var JavaScriptEvaluationTimeout = 30 * time.Second

func init() { cef.BindJavaScriptEvaluationTimeout(&JavaScriptEvaluationTimeout) }
