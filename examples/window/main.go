// Command window is the minimal gomium host with a native window: CEF opens a
// top-level window showing one page, and the program ends when the window is
// closed.
//
//	go run github.com/timzifer/gomium/tools/cefdl -dest ./Release   # once
//	go run ./examples/window -url https://example.org
//
// Windows and macOS. The message loop is the external pump, driven from the
// main thread - the same loop on both platforms, and the one a host with a
// main loop of its own (Fyne, a game loop) uses. Each worker also services
// the native event queue (Win32 / AppKit), so the window stays responsive.
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/timzifer/gomium"
)

// work carries the message-pump workers from gomium's scheduler goroutine to
// the main thread, the one that called cef_initialize.
var work = make(chan gomium.Worker, 16)

// options is shared by the browser process and every helper process.
func options() []gomium.Option {
	return append(platformOptions(),
		gomium.WithCachePath(filepath.Join(os.TempDir(), "gomium-example-window")),
		gomium.WithMessagePumpWorker(func(_ int, worker gomium.Worker) {
			select {
			case work <- worker:
			default: // the main loop is busy or gone; the next tick follows
			}
		}),
	)
}

func main() {
	runtime.LockOSThread()

	// Runtime binding first - helper processes come through here too.
	if err := gomium.LoadLibcef(libcefPath(runtimeDir())); err != nil {
		log.Fatal(err)
	}
	if err := beforeSubprocess(); err != nil {
		log.Fatal(err)
	}
	if gomium.IsSubprocess() {
		if err := gomium.ExecuteSubprocess(options()...); err != nil {
			log.Fatal(err)
		}
		return
	}

	// Flags only after the helper branch: helpers carry Chromium switches.
	pageURL := flag.String("url", "https://example.org", "page to show")
	title := flag.String("title", "gomium", "window title")
	width := flag.Int("width", 1200, "window width")
	height := flag.Int("height", 800, "window height")
	flag.Parse()

	app, err := gomium.New(options()...)
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := app.Run(ctx); err != nil {
		_ = app.Close()
		log.Fatal(err)
	}
	afterRun()

	// Closing the window destroys the browser; OnBeforeClose is the signal
	// to leave the loop.
	closed := make(chan struct{})
	client := app.NewClient(
		gomium.WithLifeSpanHandlerFuncs(nil, func() { close(closed) }, nil),
	)

	// NewBrowser waits for CEF to create the browser, which needs the pump -
	// so it runs off the main thread while main keeps pumping.
	go func() {
		if _, err := client.NewBrowser(
			gomium.WithWindow(*title, *width, *height),
			gomium.WithInitialUrl(*pageURL),
		); err != nil {
			log.Printf("creating the browser: %v", err)
			cancel()
		}
	}()

loop:
	for {
		select {
		case worker := <-work:
			worker()
		case <-closed:
			break loop
		case <-ctx.Done():
			break loop
		}
	}

	// Close pumps the message loop itself while the browsers shut down, so it
	// must run here, on the cef_initialize thread.
	if err := app.Close(); err != nil {
		log.Print(err)
	}
}

// runtimeDir finds the CEF runtime: $GOMIUM_CEF_DIR, else Release/ next to
// the executable, else Release/ in the working directory (go run from the
// module root, where tools/cefdl put it).
func runtimeDir() string {
	if dir := os.Getenv("GOMIUM_CEF_DIR"); dir != "" {
		return dir
	}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Join(filepath.Dir(exe), "Release")
		if _, err := os.Stat(dir); err == nil {
			return dir
		}
	}
	return "Release"
}
