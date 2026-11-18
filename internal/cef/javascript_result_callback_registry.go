package cef

import (
	"github.com/google/uuid"
	"sync"
	"time"
)

type (
	ResultCallback func(result *Exported, err *ExceptionError)
)

// jsEvalTimeout points at the public gomium.JavaScriptEvaluationTimeout
// variable. It is a pointer rather than a copy because that variable is a
// knob callers assign to: re-exporting it from the facade by value would
// snapshot it at init and silently ignore every later assignment.
//
// The default here matches the facade's and only applies if this package is
// ever used without one.
var jsEvalTimeout = func() *time.Duration { d := 30 * time.Second; return &d }()

// BindJavaScriptEvaluationTimeout points this package at the caller-visible
// timeout variable. The facade calls it once, from init.
func BindJavaScriptEvaluationTimeout(d *time.Duration) {
	if d != nil {
		jsEvalTimeout = d
	}
}

var (
	javascriptEvaluationsInFlight = map[uuid.UUID]ResultCallback{}
	javascriptEvaluationsMutex    sync.Mutex
)

func registerJavaScriptEvaluationCallback(id uuid.UUID, callback ResultCallback) {
	javascriptEvaluationsMutex.Lock()
	defer javascriptEvaluationsMutex.Unlock()

	javascriptEvaluationsInFlight[id] = callback

	if timeout := *jsEvalTimeout; timeout > 0 {
		time.AfterFunc(timeout, func() {
			resolveJavaScriptEvaluation(id, nil, &ExceptionError{Message: "javascript evaluation timed out"})
		})
	}
}

func resolveJavaScriptEvaluation(id uuid.UUID, result *Exported, err *ExceptionError) {
	javascriptEvaluationsMutex.Lock()
	defer javascriptEvaluationsMutex.Unlock()

	if cb, exists := javascriptEvaluationsInFlight[id]; exists {
		delete(javascriptEvaluationsInFlight, id)

		go cb(result, err)
	}

}

// failJavaScriptEvaluation resolves a pending evaluation with a local error,
// so failed dispatches do not leave callbacks dangling forever.
func failJavaScriptEvaluation(id uuid.UUID, err error) {
	if err == nil {
		return
	}
	resolveJavaScriptEvaluation(id, nil, &ExceptionError{Message: err.Error()})
}
