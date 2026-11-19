//go:build darwin || linux

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
	// Dedicated profile directory: without a cache path CEF falls back to
	// the shared default user-data-dir, and any other CEF app using that
	// default holds the Chromium process singleton - cef_initialize then
	// fails with "Opening in existing browser session".
	profileDir := filepath.Join(os.TempDir(), "gomium-test-profile")

	debugPort := 6696
	if fromEnv, err := strconv.Atoi(os.Getenv("GOMIUM_DEBUG_PORT")); err == nil && fromEnv > 0 {
		debugPort = fromEnv
	}

	// No module handle and no multi-threaded message loop here: the first is
	// a Windows concept, the second is not supported on macOS at all - CEF
	// requires its message loop on the main thread. The external pump below
	// is the portable path, and on macOS it is the only one.
	return []gomium.Option{
		gomium.WithLogger(logger),
		gomium.WithLogSeverity(gomium.LOGSEVERITY_VERBOSE),
		gomium.WithCachePath(profileDir),
		gomium.WithRemoteDebuggingPort(debugPort),
		gomium.WithMessagePumpWorker(func(delay int, worker gomium.Worker) {
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
		}),
	}
}
