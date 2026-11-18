package cef

import (
	"github.com/rs/zerolog"
	"os"
	"runtime"
	"testing"
	"time"
)

func TestNew(t *testing.T) {

	runtime.LockOSThread()

	writer := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	logger := zerolog.New(writer).With().Timestamp().Stack().Logger()

	SetDebugLogger(logger.With().Str("type", "Debug").Logger())
	SetMemoryLogger(logger.With().Str("type", "Memory").Logger())

	if app, err := New(); err != nil {
		t.Fatal(err)
	} else {
		defer func() {
			_ = app.Close()
		}()
	}

}
