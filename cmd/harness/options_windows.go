//go:build windows

package main

import (
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/timzifer/gomium"
	"log/slog"
)

func options(logger *slog.Logger) []gomium.Option {
	handle, err := gomium.GetModuleHandle(nil)
	if err != nil {
		panic(err)
	}

	// Dedicated profile directory: without a cache path CEF falls back to
	// the shared default user-data-dir, and any other CEF app using that
	// default (e.g. the terminal app) holds the Chromium process singleton -
	// cef_initialize then fails with "Opening in existing browser session".
	profileDir := filepath.Join(os.TempDir(), "gomium-test-profile")

	debugPort := 6696
	if fromEnv, err := strconv.Atoi(os.Getenv("GOMIUM_DEBUG_PORT")); err == nil && fromEnv > 0 {
		debugPort = fromEnv
	}

	opts := []gomium.Option{
		gomium.WithLogger(logger),
		gomium.WithModuleHandle(handle),
		gomium.WithLogSeverity(gomium.LOGSEVERITY_VERBOSE),
		gomium.WithCachePath(profileDir),
		gomium.WithRemoteDebuggingPort(debugPort),
	}

	// GOMIUM_MT_LOOP=1 mirrors the terminal app: CEF runs its own message
	// loop thread instead of the external pump. Without it the external pump
	// below drives OSR and windowed browsers (GOMIUM_WINDOW_TEST=1) alike -
	// each worker also drains this thread's Win32 queue.
	if os.Getenv("GOMIUM_MT_LOOP") != "" {
		opts = append(opts, gomium.WithMultiThreadedMessageLoop(true))
	} else {
		opts = append(opts, gomium.WithMessagePumpWorker(func(delay int, worker gomium.Worker) {
			go func() {
				if delay > 0 {
					time.Sleep(time.Duration(delay) * time.Millisecond)
				}
				// non-blocking: once the main loop stopped draining, a missed
				// pump tick is preferable to a leaked goroutine
				select {
				case workerChannel <- worker:
				default:
				}
			}()
		}))
	}
	return opts
}
