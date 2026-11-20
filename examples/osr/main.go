// Command osr is the minimal offscreen gomium host: it loads a page without
// any window, waits until the page has loaded and painted, and writes the
// frame to a PNG file.
//
//	go run github.com/timzifer/gomium/tools/cefdl -dest ./Release   # once
//	go run ./examples/osr -url https://example.org -out frame.png
//
// On macOS CEF starts only out of an application bundle; -agent keeps the
// offscreen host out of the Dock:
//
//	go build -o build/osr ./examples/osr
//	go run ./tools/cefbundle -bin build/osr -cef Release -out build/OSR.app -agent
//	build/OSR.app/Contents/MacOS/OSR -url https://example.org -out frame.png
//
// Everything a real host needs is here and nothing else: runtime binding,
// the helper-process branch, the external message pump driven from the main
// thread, and an orderly shutdown.
package main

import (
	"context"
	"flag"
	"image"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/timzifer/gomium"
)

// work carries the message-pump workers from gomium's scheduler goroutine to
// the main thread. CEF must be pumped on the thread that called
// cef_initialize, and that is the one main() locks below.
var work = make(chan gomium.Worker, 16)

// options is shared by the browser process and every helper process: the
// helpers are this same executable started by CEF, and they must see the
// same configuration.
func options() []gomium.Option {
	return append(platformOptions(),
		// A profile directory of our own: the default one is shared with
		// every other CEF application and its process singleton refuses a
		// second user ("Opening in existing browser session").
		gomium.WithCachePath(filepath.Join(os.TempDir(), "gomium-example-osr")),
		gomium.WithMessagePumpWorker(func(_ int, worker gomium.Worker) {
			select {
			case work <- worker:
			default: // the main loop is busy or gone; the next tick follows
			}
		}),
	)
}

func main() {
	// CEF requires all its work on the thread that called cef_initialize.
	runtime.LockOSThread()

	// libcef is bound at runtime. This has to happen before anything else -
	// helper processes come through here too, which is also why the runtime
	// is found without a command-line flag (see runtimeDir).
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

	// Flags only now: helper processes are started with Chromium switches
	// (--type=renderer, ...) that flag.Parse would reject.
	pageURL := flag.String("url", "https://example.org", "page to render")
	out := flag.String("out", "frame.png", "PNG file receiving the frame")
	width := flag.Int("width", 1280, "viewport width")
	height := flag.Int("height", 800, "viewport height")
	timeout := flag.Duration("timeout", 30*time.Second, "give up after this long")
	flag.Parse()

	app, err := gomium.New(options()...)
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	if err := app.Run(ctx); err != nil {
		_ = app.Close()
		log.Fatal(err)
	}
	afterRun()

	// The render handler paints into this image. Every paint replaces the
	// previous frame; the latest one is what gets written once the page has
	// finished loading.
	var (
		mu     sync.Mutex
		latest image.Image
	)
	canvas := image.NewRGBA(image.Rect(0, 0, *width, *height))
	painted := make(chan struct{}, 1)
	loaded := make(chan struct{}, 1)

	client := app.NewClient(
		gomium.WithRenderHandler(gomium.NewImageRenderHandler(canvas, func(frame image.Image) {
			mu.Lock()
			latest = frame
			mu.Unlock()
			select {
			case painted <- struct{}{}:
			default:
			}
		})),
		gomium.WithLoadHandlerFuncs(nil, nil,
			func(_ gomium.Browser, isMainFrame bool, status int) {
				if isMainFrame {
					log.Printf("loaded %s (HTTP %d)", *pageURL, status)
					select {
					case loaded <- struct{}{}:
					default:
					}
				}
			},
			func(_ gomium.Browser, isMainFrame bool, code int, text, failed string) {
				if isMainFrame {
					log.Printf("load error %d (%s) for %s", code, text, failed)
				}
			}),
	)

	// NewBrowser waits for CEF to create the browser, which needs the pump -
	// so it runs off the main thread while main keeps pumping.
	go func() {
		if _, err := client.NewBrowser(
			gomium.WithInitialUrl(*pageURL),
			gomium.WithWindowlessFrameRate(30),
		); err != nil {
			log.Printf("creating the browser: %v", err)
			cancel()
		}
	}()

	// The main loop: run pump workers, and leave once the page loaded and a
	// frame arrived after that (pages often paint again right after load).
	var settle <-chan time.Time
	isLoaded := false
loop:
	for {
		select {
		case worker := <-work:
			worker()
		case <-loaded:
			isLoaded = true
		case <-painted:
			if isLoaded {
				settle = time.After(500 * time.Millisecond)
			}
		case <-settle:
			break loop
		case <-ctx.Done():
			log.Printf("gave up: %v", ctx.Err())
			break loop
		}
	}

	mu.Lock()
	frame := latest
	mu.Unlock()
	if frame != nil {
		if err := writePNG(*out, frame); err != nil {
			log.Print(err)
		} else {
			log.Printf("wrote %s", *out)
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

func writePNG(name string, frame image.Image) error {
	file, err := os.Create(name)
	if err != nil {
		return err
	}
	if err := png.Encode(file, frame); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}
