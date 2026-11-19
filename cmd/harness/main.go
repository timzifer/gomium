package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/rs/zerolog"
	"github.com/timzifer/gomium"
	"image"
	"image/jpeg"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"
)

// softNavPage is a minimal SPA: every click on the full-screen button does
// history.pushState + a DOM mutation. Blink's SoftNavigationHeuristics
// reports that as a soft navigation to the browser process - the trigger for
// the CEF-151 ImmersiveReadAnything crash (TabInterface null-deref).
const softNavPage = `<!doctype html><html><head><title>softnav</title></head><body>
<button id="b" style="position:fixed;inset:0;font-size:60px">click me</button>
<script>
let n = 0;
document.getElementById('b').addEventListener('click', () => {
  n++;
  history.pushState({n}, '', '/soft-' + n);
  document.title = 'soft-' + n;
  const d = document.createElement('div');
  d.textContent = 'nav ' + n;
  d.style.cssText = 'position:fixed;left:10px;top:10px;font-size:40px;color:red;z-index:9';
  document.body.appendChild(d);
});
</script>
</body></html>`

// popupPage carries a full-screen target="_blank" link. A trusted click
// (user gesture) passes the popup blocker, so on_before_popup fires.
const popupPage = `<!doctype html><html><head><title>popup-test</title></head><body>
<a href="https://example.com/" target="_blank"
   style="position:fixed;inset:0;font-size:60px">open in new window</a>
</body></html>`

// downloadPage carries a full-screen link to an attachment response - a
// trusted click on it starts a download.
const downloadPage = `<!doctype html><html><head><title>download-test</title></head><body>
<a href="/file.bin" style="position:fixed;inset:0;font-size:60px">download</a>
</body></html>`

// dropPage takes files by drag and drop. It prints two lists: what the
// browser hands to JavaScript (names only) and what Go pushed into
// window.__droppedFiles from the drag handler (the native paths).
const dropPage = `<!doctype html><html><head><title>drop-test</title></head><body
 style="font:16px sans-serif;margin:0">
<div id="zone" style="position:fixed;inset:0;padding:24px">
  <h1>drop files here</h1>
  <pre id="out">waiting…</pre>
</div>
<script>
const out = document.getElementById("out");
addEventListener("dragover", ev => ev.preventDefault());
addEventListener("drop", ev => {
  ev.preventDefault();
  const names = Array.from(ev.dataTransfer.files, f => f.name);
  const paths = window.__droppedFiles || [];
  window.__droppedFiles = null;
  out.textContent = "names from the browser:\n  " + names.join("\n  ") +
                    "\n\npaths from go:\n  " + paths.join("\n  ");
  console.log("drop", JSON.stringify({names, paths}));
});
</script>
</body></html>`

// testDialogHandler answers confirm with OK and prompt with "gomium" so the
// page-side results prove the round trip.
type testDialogHandler struct{ logger zerolog.Logger }

func (h testDialogHandler) OnJsDialog(_ gomium.Browser, originUrl string, dialogType gomium.JsDialogType, message, defaultPrompt string, callback gomium.JsDialogCallback) gomium.JsDialogAction {
	h.logger.Info().Str("origin", originUrl).Int("type", int(dialogType)).Str("message", message).Str("default", defaultPrompt).Msg("dialog-test: js dialog")
	go func() {
		switch dialogType {
		case gomium.JsDialogTypePrompt:
			callback.Continue(true, "gomium")
		default:
			callback.Continue(true, "")
		}
	}()
	return gomium.JsDialogHandled
}

func (h testDialogHandler) OnBeforeUnloadDialog(_ gomium.Browser, message string, isReload bool, _ gomium.JsDialogCallback) gomium.JsDialogAction {
	h.logger.Info().Str("message", message).Bool("reload", isReload).Msg("dialog-test: unload dialog suppressed")
	return gomium.JsDialogSuppressed
}

// testMenuHandler logs the right-click context and swallows the menu.
type testMenuHandler struct{ logger zerolog.Logger }

func (h testMenuHandler) RunContextMenu(_ gomium.Browser, params gomium.ContextMenuParams, model []gomium.ContextMenuItem, callback gomium.ContextMenuCallback) bool {
	labels := make([]string, 0, len(model))
	for _, item := range model {
		if item.Type != gomium.MenuItemTypeSeparator {
			labels = append(labels, item.Label)
		}
	}
	h.logger.Info().Int("x", params.X).Int("y", params.Y).Str("page", params.PageUrl).
		Bool("editable", params.IsEditable).Int("items", len(model)).Strs("labels", labels).
		Msg("menu-test: context menu intercepted")
	callback.Cancel()
	return true
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// libcefPathIn names the CEF library inside a distribution directory. The
// file the loader wants differs per platform: a DLL on Windows, a shared
// object on Linux and the binary inside the framework bundle on macOS.
func libcefPathIn(dir string) string {
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(dir, "libcef.dll")
	case "darwin":
		const framework = "Chromium Embedded Framework"
		return filepath.Join(dir, framework+".framework", framework)
	default:
		return filepath.Join(dir, "libcef.so")
	}
}

func main() {

	runtime.LockOSThread()

	writer := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	logger := zerolog.New(writer).With().Timestamp().Stack().Logger()

	// libcef is bound at runtime - this must happen before Run AND before
	// ExecuteSubprocess, since helpers are new instances of this
	// executable. Prefer the library next to the executable, fall back to
	// the working directory (go run from the repo root).
	// Inside a macOS .app the framework is the only copy that exists, and a
	// helper process has no other way to find it - it is started from within
	// the bundle with a working directory of its own. Outside a bundle this
	// is empty and the Release/ lookup below applies.
	libcefPath := libcefPathIn("Release")
	if p := libcefFrameworkInBundle(); p != "" && fileExists(p) {
		libcefPath = p
	} else if exe, err := os.Executable(); err == nil {
		if p := libcefPathIn(filepath.Join(filepath.Dir(exe), "Release")); fileExists(p) {
			libcefPath = p
		}
	}
	if err := gomium.LoadLibcef(libcefPath); err != nil {
		logger.Fatal().Err(err).Send()
	}

	// macOS only: CEF needs an NSApplication that speaks CefAppProtocol, and
	// it has to exist before cef_initialize and before cef_execute_process.
	if err := installMacApplication(); err != nil {
		logger.Fatal().Err(err).Send()
	}

	// CEF helper processes are new instances of this executable - hand them
	// to CEF before any application logic runs. This mirrors how embedding
	// applications must structure their main().
	if gomium.IsSubprocess() {
		if err := gomium.ExecuteSubprocess(append(append(options(logger), schemeTestOptions()...), bindingTestOptions()...)...); err != nil {
			logger.Fatal().Err(err).Send()
		}
		return
	}

	// command-line switches; helper processes returned above and never get here
	flag.Parse()

	//	gomium.SetDebugLogger(logger.With().Str("type", "Debug").Logger())
	//	gomium.SetMemoryLogger(logger.With().Str("type", "Memory").Logger())

	if app, err := gomium.New(append(append(options(logger), schemeTestOptions()...), bindingTestOptions()...)...); err != nil {
		logger.Fatal().Err(err).Send()
	} else {
		ctx, cancel := context.WithCancel(context.Background())

		if runErr := app.Run(ctx); runErr != nil {
			logger.Error().Err(runErr).Send()
			_ = app.Close()
		} else {
			logger.Info().Msg("going...")

			// macOS only: cef_initialize is done, so this is where cefclient
			// would enter [NSApp run]. gomium pumps the loop itself, which
			// means nobody else finishes the AppKit launch or installs a
			// menu bar - and without either, a windowed browser swallows
			// cmd+C / cmd+V. Taking focus is left to the window test; an
			// offscreen run has no window to show.
			installMacDefaultMenu()
			finishMacApplicationLaunch()
			if os.Getenv("GOMIUM_WINDOW_TEST") != "" {
				activateMacApplication()
			}

			renderHandler := gomium.NewImageRenderHandler(image.NewRGBA(image.Rect(0, 0, 1024, 786)), func(result image.Image) {
				outputFile, err := os.Create(fmt.Sprintf("shots/%d.jpg", time.Now().UnixMicro()))
				if err != nil {
					fmt.Println("Error creating output file:", err)
					return
				}
				defer outputFile.Close()

				if err = jpeg.Encode(outputFile, result, nil); err != nil {
					fmt.Println("Error encoding output image:", err)
					return
				}
			})

			// popupAttached hands the popup browser from Attach to the close step
			popupAttached := make(chan gomium.Browser, 1)
			clientOptions := []gomium.ClientOption{
				gomium.WithBindingGuard(bindingTestGuard),
				gomium.WithLifeSpanHandlerFuncs(
					func() {
						spew.Dump("afterCreated")
					}, func() {
						spew.Dump("onBeforeClose")
					}, func() bool {
						spew.Dump("doClose")
						// false lets CEF proceed with destroying the browser
						return false
					}), gomium.WithDisplayHandlerFuncs(func(b gomium.Browser, s string) {
					logger.Info().Str("title", s).Msg("title changed")
				}, func(b gomium.Browser, theUrl *url.URL) {
					logger.Info().Str("url", theUrl.String()).Msg("url changed")
				}, func(b gomium.Browser, file string, line int, message string) {
					logger.Info().Str("file", file).Int("line", line).Msg(message)
				}, func(b gomium.Browser, strings []*url.URL) {
					logger.Info().Interface("urls", strings).Msg("favicon changed")
				}),
				gomium.WithLoadHandlerFuncs(
					func(b gomium.Browser, isLoading, canGoBack, canGoForward bool) {
						logger.Info().Bool("loading", isLoading).Bool("back", canGoBack).Bool("forward", canGoForward).Msg("loading state changed")
					},
					func(b gomium.Browser, isMainFrame bool) {
						logger.Info().Bool("main", isMainFrame).Msg("load start")
					},
					func(b gomium.Browser, isMainFrame bool, httpStatusCode int) {
						logger.Info().Bool("main", isMainFrame).Int("status", httpStatusCode).Msg("load end")
					},
					func(b gomium.Browser, isMainFrame bool, errorCode int, errorText, failedUrl string) {
						logger.Warn().Bool("main", isMainFrame).Int("code", errorCode).Str("text", errorText).Str("url", failedUrl).Msg("load error")
					}),
				gomium.WithPopupsInOpener(),
				gomium.WithDownloadDirectory(os.TempDir(), func(b gomium.Browser, item gomium.DownloadItem) {
					logger.Info().Uint32("id", item.Id).Str("path", item.FullPath).Int64("received", item.ReceivedBytes).Bool("complete", item.Complete).Msg("download updated")
				}),
			}

			// GOMIUM_WINDOW_TEST=1: no render handler, so CEF creates a real
			// window instead of painting offscreen. The drop handler is what
			// gives the page the paths behind dropped files - JavaScript only
			// ever sees the names.
			windowTest := os.Getenv("GOMIUM_WINDOW_TEST") != ""
			if windowTest {
				clientOptions = append(clientOptions, gomium.WithFileDropHandler(func(bi gomium.Browser, files []string) {
					logger.Info().Strs("files", files).Msg("window-test: files dragged in")
					encoded, encodeErr := json.Marshal(files)
					if encodeErr != nil {
						logger.Error().Err(encodeErr).Send()
						return
					}
					bi.ExecuteJavascript("window.__droppedFiles = "+string(encoded), "", 0)
				}))
			} else {
				clientOptions = append(clientOptions, gomium.WithRenderHandler(renderHandler))
			}

			// GOMIUM_HANDLER_TEST=1: focus, keyboard, file dialog, frame and
			// auth handlers that only log (see handlertest.go)
			if handlerTestEnabled() {
				clientOptions = append(clientOptions,
					gomium.WithFocusHandler(testFocusHandler{logger: logger}),
					gomium.WithKeyboardHandler(testKeyboardHandler{logger: logger}),
					gomium.WithFileDialogHandler(testFileDialogHandler{logger: logger}),
					gomium.WithFrameHandler(testFrameHandler{logger: logger}),
					gomium.WithAuthHandler(testAuthHandler{logger: logger}),
				)
			}

			// GOMIUM_INTERCEPT_TEST=1: serve .pdf requests from the
			// ResourceInterceptor instead of the network
			if os.Getenv("GOMIUM_INTERCEPT_TEST") != "" {
				clientOptions = append(clientOptions, gomium.WithRequestHandler(&testInterceptor{logger: logger}))
			}

			// GOMIUM_SCHEME_TEST=1: serve a custom scheme from Go
			if schemeTestEnabled() {
				clientOptions = append(clientOptions, gomium.WithRequestHandler(schemeTestHandler{logger: logger}))
			}

			// GOMIUM_DIALOG_TEST=1: answer js dialogs from Go and intercept
			// the context menu
			if os.Getenv("GOMIUM_DIALOG_TEST") != "" {
				clientOptions = append(clientOptions,
					gomium.WithJsDialogHandler(testDialogHandler{logger: logger}),
					gomium.WithContextMenuHandler(testMenuHandler{logger: logger}),
				)
			}

			// GOMIUM_DEVTOOLS_TEST=1: smoke test for WithDevTools
			if devToolsProbeEnabled() {
				clientOptions = append(clientOptions, devToolsProbeOptions(logger)...)
			}

			// GOMIUM_BINDING_POPUP_TEST=1: let target="_blank" open a real popup
			// browser (false = not cancelled) instead of navigating the opener
			// The popup gets an offscreen view with its own client and guard;
			// its frames are written to shots/popup-*.jpg so the paint can be checked.
			if os.Getenv("GOMIUM_BINDING_POPUP_TEST") != "" {
				clientOptions = append(clientOptions,
					gomium.WithPopupHandler(func(gomium.Browser, gomium.PopupRequest) bool { return false }),
					gomium.WithPopupView(func(opener gomium.Browser, _ gomium.PopupRequest) (gomium.PopupView, bool) {
						// GOMIUM_POPUP_ABORT=1: close the opener while CEF builds the
						// popup, so CEF aborts it - the prepared client must go away
						if os.Getenv("GOMIUM_POPUP_ABORT") != "" && opener != nil {
							logger.Info().Msg("binding-popup-test: closing opener during popup creation")
							go func() { _ = opener.Close() }()
						}
						popupImage := gomium.NewImageRenderHandler(image.NewRGBA(image.Rect(0, 0, 1024, 786)), func(result image.Image) {
							outputFile, err := os.Create(fmt.Sprintf("shots/popup-%d.jpg", time.Now().UnixMicro()))
							if err != nil {
								fmt.Println("Error creating popup output file:", err)
								return
							}
							defer outputFile.Close()
							if err = jpeg.Encode(outputFile, result, nil); err != nil {
								fmt.Println("Error encoding popup output image:", err)
							}
						})
						options := []gomium.ClientOption{gomium.WithRenderHandler(popupImage)}
						// GOMIUM_POPUP_NO_GUARD=1: the popup client gets no guard of its
						// own, so its binding call must fall back to the default (deny)
						if os.Getenv("GOMIUM_POPUP_NO_GUARD") == "" {
							options = append(options, gomium.WithBindingGuard(bindingTestGuard))
						}
						return gomium.PopupView{
							Options: append(options,
								gomium.WithDisplayHandlerFuncs(nil, nil, func(_ gomium.Browser, _ string, _ int, message string) {
									logger.Info().Str("console", message).Msg("binding-popup-test: popup console")
								}, nil),
							),
							Attach: func(b gomium.Browser) {
								logger.Info().Msg("binding-popup-test: popup browser attached")
								popupAttached <- b
							},
						}, true
					}),
				)
			}

			browserClient := app.NewClient(clientOptions...)

			closeChan := make(chan bool, 2)
			go func() {
				// GOMIUM_RUN_SECONDS overrides the default lifetime - long
				// runs are needed to catch the ~90s profiler crash window
				runSeconds := 20
				if fromEnv, err := strconv.Atoi(os.Getenv("GOMIUM_RUN_SECONDS")); err == nil && fromEnv > 0 {
					runSeconds = fromEnv
				}
				time.Sleep(time.Duration(runSeconds) * time.Second)
				closeChan <- true
				cancel()
			}()

			go func() {

				initialUrl := "https://example.org"
				if fromEnv := os.Getenv("GOMIUM_TEST_URL"); fromEnv != "" {
					initialUrl = fromEnv
				}

				// GOMIUM_SCHEME_TEST=1: the page lives behind the custom
				// scheme, so it is also the initial URL
				if schemeTestEnabled() {
					initialUrl = schemeTestURL
				}

				// GOMIUM_POPUP_TEST=1: serve the target="_blank" page locally;
				// the trusted click below routes the popup into the opener
				if os.Getenv("GOMIUM_POPUP_TEST") != "" {
					mux := http.NewServeMux()
					mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
						w.Header().Set("Content-Type", "text/html")
						_, _ = w.Write([]byte(popupPage))
					})
					go func() { _ = http.ListenAndServe("127.0.0.1:18778", mux) }()
					initialUrl = "http://127.0.0.1:18778/"
				}

				// GOMIUM_DOWNLOAD_TEST=1: serve a page whose link answers with
				// Content-Disposition: attachment
				if os.Getenv("GOMIUM_DOWNLOAD_TEST") != "" {
					mux := http.NewServeMux()
					mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
						w.Header().Set("Content-Type", "text/html")
						_, _ = w.Write([]byte(downloadPage))
					})
					mux.HandleFunc("/file.bin", func(w http.ResponseWriter, _ *http.Request) {
						w.Header().Set("Content-Type", "application/octet-stream")
						w.Header().Set("Content-Disposition", `attachment; filename="gomium-download-test.bin"`)
						_, _ = w.Write(make([]byte, 256*1024))
					})
					go func() { _ = http.ListenAndServe("127.0.0.1:18779", mux) }()
					initialUrl = "http://127.0.0.1:18779/"
				}

				// GOMIUM_SOFTNAV_TEST=1: serve the SPA page locally; clicks
				// are injected further below once the browser exists
				if os.Getenv("GOMIUM_SOFTNAV_TEST") != "" {
					mux := http.NewServeMux()
					mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
						w.Header().Set("Content-Type", "text/html")
						_, _ = w.Write([]byte(softNavPage))
					})
					go func() { _ = http.ListenAndServe("127.0.0.1:18777", mux) }()
					initialUrl = "http://127.0.0.1:18777/"
				}

				// GOMIUM_WINDOW_TEST=1: serve the drop page locally - a
				// file:// origin would let Chromium navigate to the dropped
				// file before the page sees anything
				if windowTest {
					mux := http.NewServeMux()
					mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
						w.Header().Set("Content-Type", "text/html")
						_, _ = w.Write([]byte(dropPage))
					})
					go func() { _ = http.ListenAndServe("127.0.0.1:18776", mux) }()
					initialUrl = "http://127.0.0.1:18776/"
				}

				// GOMIUM_BINDING_POPUP_TEST=1: the opener has a target="_blank" link
				// at 200,200 (clicked by the trusted click below); the popup calls
				// app.ping too. Both report to the log as BINDING_MAIN/POPUP lines.
				if os.Getenv("GOMIUM_BINDING_POPUP_TEST") != "" {
					mux := http.NewServeMux()
					mux.HandleFunc("/popup", func(w http.ResponseWriter, _ *http.Request) {
						w.Header().Set("Content-Type", "text/html")
						_, _ = w.Write([]byte(bindingPopupTarget))
					})
					mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
						w.Header().Set("Content-Type", "text/html")
						_, _ = w.Write([]byte(bindingPopupOpener))
					})
					go func() { _ = http.ListenAndServe("127.0.0.1:18779", mux) }()
					initialUrl = "http://127.0.0.1:18779/"
				}

				// GOMIUM_BINDING_TEST=1: the page calls the binding app.ping and
				// logs the answer as console line BINDING_RESULT / BINDING_ERR
				if os.Getenv("GOMIUM_BINDING_TEST") != "" {
					mux := http.NewServeMux()
					mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
						w.Header().Set("Content-Type", "text/html")
						_, _ = w.Write([]byte(bindingPage))
					})
					go func() { _ = http.ListenAndServe("127.0.0.1:18779", mux) }()
					initialUrl = "http://127.0.0.1:18779/"
				}

				browserOptions := []gomium.BrowserOption{gomium.WithInitialUrl(initialUrl)}
				if windowTest {
					browserOptions = append(browserOptions, gomium.WithWindow("gomium window test", 1024, 768))
				} else {
					browserOptions = append(browserOptions, gomium.WithWindowlessFrameRate(30))
				}

				br, browserCreateErr := browserClient.NewBrowser(browserOptions...)
				if browserCreateErr != nil {
					logger.Error().Err(browserCreateErr).Send()
					closeChan <- true
					cancel()
				} else {
					// GOMIUM_BINDING_POPUP_TEST=1: one trusted click on the _blank link
					if os.Getenv("GOMIUM_BINDING_POPUP_TEST") != "" {
						go func() {
							time.Sleep(5 * time.Second)
							logger.Info().Msg("binding-popup-test: trusted click on _blank link")
							// the popup is closed again after a while: its client must go with it
							go func() {
								popup := <-popupAttached
								time.Sleep(5 * time.Second)
								logger.Info().Msg("binding-popup-test: closing popup")
								_ = popup.Close()
							}()
							br.MouseMoved(200, 200, false)
							time.Sleep(100 * time.Millisecond)
							br.MouseClicked(200, 200, gomium.MouseButtonLeft, gomium.MouseButtonDown, 1)
							time.Sleep(80 * time.Millisecond)
							br.MouseClicked(200, 200, gomium.MouseButtonLeft, gomium.MouseButtonUp, 1)
						}()
					}

					// GOMIUM_POPUP_TEST=1: one trusted click on the _blank link
					if os.Getenv("GOMIUM_POPUP_TEST") != "" {
						go func() {
							time.Sleep(5 * time.Second)
							logger.Info().Msg("popup-test: trusted click on _blank link")
							br.MouseMoved(200, 200, false)
							time.Sleep(100 * time.Millisecond)
							br.MouseClicked(200, 200, gomium.MouseButtonLeft, gomium.MouseButtonDown, 1)
							time.Sleep(80 * time.Millisecond)
							br.MouseClicked(200, 200, gomium.MouseButtonLeft, gomium.MouseButtonUp, 1)
						}()
					}

					// GOMIUM_DOWNLOAD_TEST=1: one trusted click on the link
					if os.Getenv("GOMIUM_DOWNLOAD_TEST") != "" {
						go func() {
							time.Sleep(5 * time.Second)
							logger.Info().Msg("download-test: trusted click on attachment link")
							br.MouseMoved(200, 200, false)
							time.Sleep(100 * time.Millisecond)
							br.MouseClicked(200, 200, gomium.MouseButtonLeft, gomium.MouseButtonDown, 1)
							time.Sleep(80 * time.Millisecond)
							br.MouseClicked(200, 200, gomium.MouseButtonLeft, gomium.MouseButtonUp, 1)
						}()
					}

					// GOMIUM_SOFTNAV_TEST=1: real (trusted) mouse clicks on the
					// SPA button - each one produces a soft navigation
					if os.Getenv("GOMIUM_SOFTNAV_TEST") != "" {
						go func() {
							for i := 1; i <= 3; i++ {
								time.Sleep(5 * time.Second)
								logger.Info().Int("click", i).Msg("softnav-test: trusted click")
								br.MouseMoved(200, 200, false)
								time.Sleep(100 * time.Millisecond)
								br.MouseClicked(200, 200, gomium.MouseButtonLeft, gomium.MouseButtonDown, 1)
								time.Sleep(80 * time.Millisecond)
								br.MouseClicked(200, 200, gomium.MouseButtonLeft, gomium.MouseButtonUp, 1)
							}
						}()
					}

					// GOMIUM_COOKIE_TEST=1: exercise the cookie manager - set,
					// read back, delete, read again
					if os.Getenv("GOMIUM_COOKIE_TEST") != "" {
						go func() {
							time.Sleep(4 * time.Second)
							cookies := gomium.GlobalCookieManager()

							expires := time.Now().Add(24 * time.Hour)
							if err := cookies.SetCookie("https://example.org/", gomium.Cookie{
								Name:    "gomium_test",
								Value:   "hello-from-go",
								Path:    "/",
								Secure:  true,
								Expires: &expires,
							}); err != nil {
								logger.Error().Err(err).Msg("cookie-test: set failed")
								return
							}
							logger.Info().Msg("cookie-test: set ok")

							if all, err := cookies.CookiesForUrl("https://example.org/", true); err != nil {
								logger.Error().Err(err).Msg("cookie-test: read failed")
							} else {
								for _, c := range all {
									logger.Info().Str("name", c.Name).Str("value", c.Value).Str("domain", c.Domain).Time("created", c.Creation).Msg("cookie-test: got cookie")
								}
							}

							if count, err := cookies.DeleteCookies("https://example.org/", "gomium_test"); err != nil {
								logger.Error().Err(err).Msg("cookie-test: delete failed")
							} else {
								logger.Info().Int("deleted", count).Msg("cookie-test: delete ok")
							}

							if all, err := cookies.Cookies(); err != nil {
								logger.Error().Err(err).Msg("cookie-test: full read failed")
							} else {
								logger.Info().Int("total", len(all)).Msg("cookie-test: cookies remaining in store")
							}
						}()
					}

					// GOMIUM_CONTEXT_TEST=1: two incognito request contexts -
					// cookie isolation plus a browser living in context A
					if os.Getenv("GOMIUM_CONTEXT_TEST") != "" {
						go func() {
							time.Sleep(4 * time.Second)

							ctxA, err := gomium.NewRequestContext()
							if err != nil {
								logger.Error().Err(err).Msg("context-test: create A failed")
								return
							}
							ctxB, err := gomium.NewRequestContext()
							if err != nil {
								logger.Error().Err(err).Msg("context-test: create B failed")
								return
							}
							global, err := gomium.GlobalRequestContext()
							if err != nil {
								logger.Error().Err(err).Msg("context-test: global failed")
								return
							}
							logger.Info().Bool("a-global", ctxA.IsGlobal()).Bool("global-global", global.IsGlobal()).Msg("context-test: contexts created")

							expires := time.Now().Add(time.Hour)
							if err := ctxA.CookieManager().SetCookie("https://ctx.example/", gomium.Cookie{
								Name: "tenant", Value: "context-a", Path: "/", Secure: true, Expires: &expires,
							}); err != nil {
								logger.Error().Err(err).Msg("context-test: set in A failed")
								return
							}

							countIn := func(label string, rc gomium.RequestContext) {
								if all, err := rc.CookieManager().Cookies(); err != nil {
									logger.Error().Err(err).Str("ctx", label).Msg("context-test: read failed")
								} else {
									logger.Info().Str("ctx", label).Int("cookies", len(all)).Msg("context-test: store size")
								}
							}
							countIn("A", ctxA)
							countIn("B", ctxB)
							countIn("global", global)

							// browser inside context A - shares A's storage
							br2, err := browserClient.NewBrowser(
								gomium.WithInitialUrl("https://example.org"),
								gomium.WithWindowlessFrameRate(5),
								gomium.WithRequestContext(ctxA),
							)
							if err != nil {
								logger.Error().Err(err).Msg("context-test: browser in context failed")
								return
							}
							logger.Info().Int("id", br2.Identifier()).Msg("context-test: browser in context A created")

							// site cookies land in A, the global store stays clean
							time.Sleep(6 * time.Second)
							countIn("A-after-load", ctxA)
							countIn("global-after-load", global)
						}()
					}

					// GOMIUM_DIALOG_TEST=1: fire alert/confirm/prompt, read the
					// answers back, then right-click for the context menu
					if os.Getenv("GOMIUM_DIALOG_TEST") != "" {
						go func() {
							time.Sleep(4 * time.Second)
							br.ExecuteJavascript(`
								window.__d = {};
								alert('alert-test');
								window.__d.confirmed = confirm('confirm-test');
								window.__d.prompted = prompt('prompt-test', 'unset');`, "", 0)

							time.Sleep(2 * time.Second)
							_ = br.Cef().MainFrame().ExecuteJavaScriptWithResult(`window.__d`, nil, 0, func(result *gomium.Exported, err *gomium.ExceptionError) {
								if err != nil {
									logger.Error().Err(err).Msg("dialog-test: result read failed")
								} else {
									logger.Info().Any("result", result.Native()).Msg("dialog-test: page-side answers")
								}
							})

							time.Sleep(1 * time.Second)
							logger.Info().Msg("menu-test: right click")
							br.MouseMoved(300, 300, false)
							time.Sleep(100 * time.Millisecond)
							br.MouseClicked(300, 300, gomium.MouseButtonRight, gomium.MouseButtonDown, 1)
							time.Sleep(80 * time.Millisecond)
							br.MouseClicked(300, 300, gomium.MouseButtonRight, gomium.MouseButtonUp, 1)
						}()
					}

					// GOMIUM_NAV_TEST=1: simulate link-style navigations
					if os.Getenv("GOMIUM_NAV_TEST") != "" {
						go func() {
							time.Sleep(6 * time.Second)
							logger.Info().Msg("nav-test: location.href")
							br.ExecuteJavascript(`location.href='https://example.com/';`, "", 0)
							time.Sleep(6 * time.Second)
							logger.Info().Msg("nav-test: click first link")
							br.ExecuteJavascript(`var a=document.querySelector('a[href]'); if(a){a.click();}`, "", 0)
							time.Sleep(6 * time.Second)
							logger.Info().Msg("nav-test: window.open")
							br.ExecuteJavascript(`window.open('https://example.org/');`, "", 0)
						}()
					}

					// GOMIUM_RESIZE_STRESS=1: continuously resize the OSR
					// surface like a parent window being dragged
					if os.Getenv("GOMIUM_RESIZE_STRESS") != "" {
						go func() {
							time.Sleep(4 * time.Second)
							sizes := [][2]int{{800, 600}, {1280, 1024}, {640, 480}, {1920, 1080}, {1024, 786}, {333, 777}}
							for i := 0; ; i++ {
								s := sizes[i%len(sizes)]
								if rh, ok := renderHandler.(gomium.ResizableRenderHandler); ok {
									rh.Resize(s[0], s[1])
								}
								if err := br.Redraw(); err != nil {
									logger.Warn().Err(err).Msg("redraw failed, stopping resize stress")
									return
								}
								time.Sleep(300 * time.Millisecond)
							}
						}()
					}
					go func() {

						time.Sleep(3 * time.Second)
						/*
							v8 := br.Cef().MainFrame().V8()
							if nil != v8 {
								spew.Dump(gomium.CefCurrentlyOn(gomium.TidRenderer))
								spew.Dump(gomium.CefCurrentlyOn(gomium.TidUI))
							}*/

						/*_ = br.Cef().MainFrame().ExecuteJavaScriptWithResult("navigator.appName", nil, 99, func(result gomium.Value, err *gomium.ExceptionError) {

							if err != nil {
								logger.Error().Err(err).Send()
							} else {
								logger.Info().Any("result", result).Msg("result received")
							}
						})*/
						/*
							_ = br.Cef().MainFrame().ExecuteJavaScriptWithResult(`const person = {
								  firstName: "John",
								  lastName: "Doe",
								  age: 50,
								  eyeColor: "blue"
								}; person`, nil, 101, func(result gomium.Value, err *gomium.ExceptionError) {
								/*/
						//		}

					}()
				}

			}()
		loop:
			for {
				select {
				case <-ctx.Done():
					break loop
				case <-closeChan:
					break loop
				case f := <-workerChannel:
					f()
				}
			}

			// Close pumps the message loop itself while the browsers shut
			// down - it must run on this (the cef_initialize) thread. In a
			// -tags refdebug build it also reports structs that outlived CEF.
			if closeErr := app.Close(); closeErr != nil {
				logger.Error().Err(closeErr).Msg("close failed")
				os.Exit(3)
			}
		}
	}

}

const bindingPopupOpener = `<!doctype html><html><body style="margin:0">
<a href="/popup" target="_blank" style="position:absolute;left:180px;top:180px;font-size:40px">popup</a>
<script>
app.ping().then(function (v) { console.log("BINDING_MAIN_RESULT " + v); },
  function (e) { console.log("BINDING_MAIN_ERR " + e); });
</script></body></html>`

const bindingPopupTarget = `<!doctype html><html><body style="margin:0;font-size:80px;font-family:sans-serif"><h1>POPUP VISIBLE</h1><script>
console.log("POPUP_PROBE url=" + location.href + " app=" + typeof app);
app.ping().then(function (v) { console.log("BINDING_POPUP_RESULT " + v); },
  function (e) { console.log("BINDING_POPUP_ERR " + e); });
</script></body></html>`

const bindingPage = `<!doctype html><html><body><script>
var isFrame = window.top !== window;
var tag = isFrame ? "FRAME" : "MAIN";
var t0 = performance.now();
app.ping().then(function (v) {
  console.log("BINDING_" + tag + "_RESULT " + v + " ms=" + (performance.now() - t0).toFixed(1));
}, function (e) {
  console.log("BINDING_" + tag + "_ERR " + e);
});
if (!isFrame) {
  app.add(2, 3).then(function (v) { console.log("BINDING_ADD " + v); }, function (e) { console.log("BINDING_ADD_ERR " + e); });
  app.fail().then(function (v) { console.log("BINDING_FAIL_UNEXPECTED " + v); }, function (e) { console.log("BINDING_FAIL " + e); });
  var tc = performance.now();
  app.cookieCount().then(function (v) {
    console.log("BINDING_UIWAIT " + v + " ms=" + (performance.now() - tc).toFixed(1));
  }, function (e) { console.log("BINDING_UIWAIT_ERR " + e); });
  var frame = document.createElement("iframe");
  frame.src = "/";
  document.body.appendChild(frame);
}
</script></body></html>`
