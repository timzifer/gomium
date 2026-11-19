package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"os"
	"sync"
	"sync/atomic"
	"time"

	gomium "github.com/timzifer/gomium"
	"log/slog"
)

// devToolsFlag enables the DevTools probe: `-devtools` on the command line.
// The flag is parsed after the CEF subprocess check, so helper processes never
// see it.
var devToolsFlag = flag.Bool("devtools", false, "run the WithDevTools probe: attach, evaluate, load events, cross-site navigation, log every attach")

// devToolsReattachURL is the second origin the probe navigates to. The live
// run showed that this navigation does not detach the agent: the attach count
// stays at 1 and the load event still reaches the handler.
const devToolsReattachURL = "https://example.org/"

// devToolsProbeEnabled gates the probe: `-devtools` or GOMIUM_DEVTOOLS_TEST=1.
func devToolsProbeEnabled() bool {
	return *devToolsFlag || os.Getenv("GOMIUM_DEVTOOLS_TEST") != ""
}

// devToolsProbeOptions attaches a receiver that logs what the DevTools session
// does. The first attach evaluates the page and, on its first load event,
// navigates cross-site once. Every attach runs the receiver again, so a
// reattach shows up as a second "attached #n" line with the new URL.
func devToolsProbeOptions(logger *slog.Logger) []gomium.ClientOption {
	var attaches atomic.Int32
	var loadCount atomic.Int32
	var navigated sync.Once
	var crashed sync.Once

	return []gomium.ClientOption{
		gomium.WithDevTools(func(b gomium.Browser, dt gomium.DevTools) {
			n := attaches.Add(1)
			logger.Info("devtools: attached", "attach", n)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if _, err := dt.Call(ctx, "Page.enable", nil); err != nil {
				logger.Error("devtools: Page.enable failed", "err", err)
			}
			raw, err := dt.Call(ctx, "Runtime.evaluate", map[string]any{
				"expression":    "location.href + ' | ' + document.title",
				"returnByValue": true,
			})
			if err != nil {
				logger.Error("devtools: Runtime.evaluate failed", "err", err, "attach", n)
			} else {
				logger.Info("devtools: evaluate", "attach", n, "result", string(raw))
			}

			dt.On("Page.loadEventFired", func(p json.RawMessage) {
				logger.Info("devtools: loadEventFired", "attach", n, "params", string(p))
				loads := loadCount.Add(1)
				if loads == 1 {
					navigated.Do(func() {
						go func() {
							nctx, ncancel := context.WithTimeout(context.Background(), 5*time.Second)
							defer ncancel()
							if _, err := dt.Call(nctx, "Page.navigate", map[string]any{"url": devToolsReattachURL}); err != nil {
								logger.Error("devtools: Page.navigate failed", "err", err)
								return
							}
							logger.Info("devtools: navigate issued", "url", devToolsReattachURL)
						}()
					})
					return
				}
				// second load: the page on the second origin. Crash its renderer
				// to force a detach. The pending Page.crash call either returns
				// ErrDevToolsDetached or the renderer dies before any answer.
				crashed.Do(func() {
					go func() {
						cctx, ccancel := context.WithTimeout(context.Background(), 5*time.Second)
						defer ccancel()
						// Page.crash did not detach the agent in the first live run
						// (deadline exceeded, no reattach), so try a renderer crash
						// through navigation instead.
						_, err := dt.Call(cctx, "Page.navigate", map[string]any{"url": "chrome://crash"})
						logger.Info("devtools: chrome://crash navigation returned", "attach", n, "err", err, "detached", errors.Is(err, gomium.ErrDevToolsDetached))
					}()
				})
			})
		}),
	}
}
