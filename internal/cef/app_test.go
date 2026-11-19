package cef

import (
	"log/slog"
	"os"
	"runtime"
	"testing"
)

func TestNew(t *testing.T) {

	runtime.LockOSThread()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: LevelTrace}))

	SetDebugLogger(logger.With("type", "Debug"))
	SetMemoryLogger(logger.With("type", "Memory"))

	if app, err := New(); err != nil {
		t.Fatal(err)
	} else {
		defer func() {
			_ = app.Close()
		}()
	}

}
