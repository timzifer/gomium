package cef

import (
	"context"
	"fmt"
	"log/slog"
	"os"
)

// LevelTrace is below slog.LevelDebug: the reference-count and bridge
// tracing of SetMemoryLogger / SetDebugLogger, and the noisiest lifecycle
// steps. Enable it with a handler whose level is LevelTrace or lower.
const LevelTrace = slog.Level(-8)

// discardHandler drops every record. slog.DiscardHandler arrived in Go 1.24;
// the module still builds with 1.23.
type discardHandler struct{}

func (discardHandler) Enabled(context.Context, slog.Level) bool  { return false }
func (discardHandler) Handle(context.Context, slog.Record) error { return nil }
func (d discardHandler) WithAttrs([]slog.Attr) slog.Handler      { return d }
func (d discardHandler) WithGroup(string) slog.Handler           { return d }

// nopLogger is the default of every logger gomium holds.
func nopLogger() *slog.Logger { return slog.New(discardHandler{}) }

// orNop maps a nil logger, which callers may pass to switch logging off, to
// the discarding one.
func orNop(l *slog.Logger) *slog.Logger {
	if l == nil {
		return nopLogger()
	}
	return l
}

func trace(l *slog.Logger, msg string, args ...any) {
	l.Log(context.Background(), LevelTrace, msg, args...)
}

// fatal logs at error level and terminates the process, as zerolog's Fatal
// did before.
func fatal(l *slog.Logger, msg string, args ...any) {
	l.Error(msg, args...)
	os.Exit(1)
}

// hexPtr renders a pointer as 0x... lazily: the formatting runs only when a
// handler actually emits the record.
type hexPtr uintptr

func (p hexPtr) LogValue() slog.Value {
	return slog.StringValue(fmt.Sprintf("0x%x", uintptr(p)))
}
