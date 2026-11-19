# gomium – API reference

Go binding for CEF (Chromium Embedded Framework). Two modes of operation:
offscreen rendering for embedding in Fyne/custom UIs, or a window of its own
(`WithWindow`, Windows and macOS). Current state: CEF 156.0.27, pinned in
[`cefdl/cef.version`](../cefdl/cef.version).

This file is the quick reference for the public API. Details live in the
GoDoc on the symbols themselves.

---

## Minimal skeleton

```go
func main() {
    runtime.LockOSThread()

    // libcef is bound at runtime - BEFORE anything else, including before
    // ExecuteSubprocess (helpers are new instances of this executable).
    // Windows: "Release/libcef.dll", Linux: "Release/libcef.so", macOS:
    // "Release/Chromium Embedded Framework.framework" (the bundle or the
    // binary inside it).
    if err := gomium.LoadLibcef(filepath.Join("Release", "libcef.dll")); err != nil {
        log.Fatal(err)
    }

    // macOS only, also before both branches: CEF talks to the
    // NSApplication via CefAppProtocol and does not run under any other.
    // if err := gomium.InstallMacApplication(); err != nil { log.Fatal(err) }

    // Detect and branch off CEF helper processes
    if gomium.IsSubprocess() {
        _ = gomium.ExecuteSubprocess(opts()...)
        return
    }

    app, err := gomium.New(opts()...)             // cef_initialize
    if err != nil { log.Fatal(err) }
    if err := app.Run(ctx); err != nil { ... }    // start the message loop

    client := app.NewClient(clientOpts()...)
    browser, err := client.NewBrowser(
        gomium.WithInitialUrl("https://example.org"),
        gomium.WithWindowlessFrameRate(30),
    )
    ...
    _ = app.Close()   // must run on the cef_initialize thread
}
```

**Important:** Every app needs `WithCachePath` with its own directory. Without
it, all CEF apps share the default user data directory — if one is already
running (e.g. the terminal app), `cef_initialize` of the second one fails with
"Opening in existing browser session".

---

## App level

| Symbol | Purpose |
|---|---|
| `New(opts ...Option) (CefApp, error)` | initializes CEF (once per process) |
| `Get() (CefApp, bool)` | get the running instance |
| `CefApp.Run(ctx)` | start the message loop |
| `CefApp.NewClient(opts ...ClientOption) Client` | create a browser client |
| `CefApp.AddV8Listener(l V8Listener)` | renderer: V8 context created/released |
| `CefApp.Close()` | close browsers + `cef_shutdown`; pumps on its own; with an external pump it returns an error off the init thread |
| `IsSubprocess() bool` / `ExecuteSubprocess(opts...)` | helper process branch in `main()` |
| `LoadLibcef(path string) error` | bind libcef at runtime (before anything else) |
| `CEFVersion() string` | bound CEF version |
| `InstallMacApplication() error` | macOS: provide an `NSApplication` with `CefAppProtocol` (before `New`/`ExecuteSubprocess`) |

### App options (`Option`)

| Option | Purpose |
|---|---|
| `WithCachePath(dir)` | Profile/cache directory. **Mandatory in real apps** (singleton! Cookies + localStorage persist only with it) |
| `WithRootCachePath(dir)` | parent directory of all cache paths incl. RequestContexts; default = CachePath |
| `WithLogger(*slog.Logger)` / `WithLogSeverity(sev)` | gomium's logging (`nil` = off; `LevelTrace` below Debug) / CEF's own log |
| `WithRemoteDebuggingPort(port)` | DevTools: point Chrome at `localhost:port` |
| `WithUserAgent(ua)` | global user agent |
| `WithIgnoreCertificateErrors(bool)` | ignore certificate errors globally (footgun — prefer the client option `WithCertificateErrorsAcceptedFor`) |
| `WithMessagePumpWorker(cb)` / `WithMessagePumpWorkerInterval(d)` | external message pump (Fyne main loop integration) |
| `WithMultiThreadedMessageLoop(bool)` | CEF's own loop thread (Windows) instead of an external pump |
| `WithInitializationTimeout(d)` | timeout for `New` |
| `WithModuleHandle(h)` | Windows: module handle (`GetModuleHandle(nil)`) |
| `WithCustomScheme(name, options)` | custom scheme (`help://`, `app://`) — see below |
| `WithV8Listener(l)` | renderer: V8 context created/released — see [V8 lifecycle](#v8-lifecycle) |

---

## Client level

`Client` = one `cef_client_t` with a set of handlers; `NewBrowser` creates
browsers on it (one after another).

### Client options (`ClientOption`)

| Option | Purpose |
|---|---|
| `WithRenderHandler(h)` | **enables offscreen rendering**; `NewImageRenderHandler` see below |
| `WithDisplayHandler(h)` / `WithDisplayHandlerFuncs(...)` | title, URL, favicon, console |
| `WithLoadHandler(h)` / `WithLoadHandlerFuncs(...)` | loading state, errors, back/forward status |
| `WithLifeSpanHandler(h)` / `WithLifeSpanHandlerFuncs(...)` | afterCreated / doClose / beforeClose |
| `WithPopupHandler(fn)` | intercept `window.open`/`target=_blank` |
| `WithPopupsInOpener()` | suppress popups, load the URL in the opener |
| `WithPopupView(factory)` | open allowed popups offscreen, with their own client (own render handler and guard) |
| `WithDownloadHandler(h)` | control downloads (path, progress, cancel/pause/resume) |
| `WithDownloadDirectory(dir, onUpdate)` | download everything to `dir`, optional progress callback |
| `WithJsDialogHandler(h)` | handle alert/confirm/prompt/beforeunload |
| `WithJsDialogsSuppressed()` | silence all JS dialogs |
| `WithContextMenuHandler(h)` | take over the right-click menu |
| `WithContextMenuSuppressed()` | swallow the right-click menu |
| `WithRequestHandler(h)` | intercept navigation/requests, ResourceInterceptor |
| `WithCertificateErrorHandler(h)` | decide TLS certificate errors per request (see below) |
| `WithCertificateErrorsAcceptedFor(hosts...)` | accept invalid certificates only for these hosts, reject the rest |
| `WithFindHandler(h)` / `WithFindHandlerFunc(fn)` | receive results of `Browser.Find` |
| `WithDragHandler(h)` / `WithFileDropHandler(fn)` | native paths of a file drag (see below) |
| `WithDefaultInitialUrl(url)` | fallback URL for browsers without `WithInitialUrl` |
| `WithBrowserCreationTimeout(d)` | timeout for `NewBrowser` |
| `WithDevTools(fn)` | DevTools protocol per browser, `fn` on every attach (see below) |

---

## Browser level

### Browser options (`BrowserOption`)

| Option | Purpose |
|---|---|
| `WithInitialUrl(url)` | start URL |
| `WithRequestContext(rc)` | browser in its own profile (see RequestContexts) |
| `WithWindowlessFrameRate(fps)` | OSR frame rate (1–60) |
| `WithBackgroundColor(argb)` | background, default opaque white |
| `WithWebgl(bool)` | WebGL |
| `WithWebSecurity(bool)` / `WithUniversalAccessFromFileUrls(bool)` / `WithFileAccessFromFileUrls(bool)` | since CEF 151 only command-line switches; the options have no effect |
| `WithWindow(title, w, h)` / `WithWindowBounds(title, rect)` | own window instead of OSR (Windows and macOS, see below) |
| `WithParentView(view)` | macOS: browser into an existing `NSView` instead of its own window |

### Browser methods

```go
LoadUrl(url string)                LoadString(data, url string)
Reload() / Redraw() / Close()      Identifier() int
GoBack() / GoForward()             CanGoBack() / CanGoForward() bool
SetZoomLevel(level float64)        ZoomLevel() (float64, error)  // 0 = default; level = log(factor)/log(1.2)
Find(text, forward, matchCase, findNext)    // results via WithFindHandlerFunc
StopFinding(clearSelection bool)
SetAudioMuted(muted bool)          IsAudioMuted() (bool, error)
PrintToPdf(path string, opts PdfPrintOptions) error  // blocks until the file is written (timeout 30 s)
FocusGained() / FocusLost()
MouseMoved(x, y, mouseLeft, mods...)
MouseClicked(x, y, button, direction, clickCount, mods...)
Scrolled(x, y, deltaX, deltaY, mods...)     // +deltaY = up, notch = 120
KeyEvent(type, nativeKey, windowsKey, char, isSystem, mods...)
ExecuteJavascript(source, url string, line int)
Cef() CefBrowser                   // low-level access (borrowed, see below)
```

`KeyEventModifier`: `Shift`, `Control`, `Alt`, `LeftMouseButton`, …
`MouseButton`: `Left`, `Middle`, `Right`. `MouseButtonDirection`: `Down`, `Up`.

---

## Rendering (OSR → Fyne)

```go
rh := gomium.NewImageRenderHandler(image.NewRGBA(image.Rect(0, 0, w, h)),
    func(img image.Image) { /* push into the Fyne canvas */ })
client := app.NewClient(gomium.WithRenderHandler(rh))
```

`NewImageRenderHandler` hands every callback a fresh copy of the whole view — simple
and safe to keep, but 8 MB per paint at 1080p. For 30–60 fps, large or HiDPI views use
`NewFrameRenderHandler`: it reuses one surface and passes the dirty rectangles, nothing
is allocated per paint.

```go
rh := gomium.NewFrameRenderHandler(w, h, func(frame *image.RGBA, dirty []image.Rectangle) {
    // frame and dirty are valid only inside this call: copy the dirty rects
    // into your texture / buffer, do not keep frame
})
```

- `RenderHandler`: `GetViewRect`, `DoPaint([]DirtyArea)`, `CursorChanged`.
- `ResizableRenderHandler`: additionally `Resize(w, h)` — then `Browser.Redraw()`.
- OSR forces the Alloy runtime style (Chrome style cannot do OSR).

### Popups (`<select>` dropdowns) — `PopupRenderHandler`

CEF delivers the dropdown as a separate surface (view coordinates). A
`RenderHandler` without this interface does **not** get popups (they are
discarded). Implement optionally:

```go
type PopupRenderHandler interface {
    PopupShow(show bool)              // show/hide the popup
    PopupSize(rect image.Rectangle)   // position+size in view coordinates
    DoPaintPopup(areas []DirtyArea)   // rect popup-local, buffer = RGBA tile
}
```

- `NewImageRenderHandler` and `NewFrameRenderHandler` implement it and composite the
  popup into the image.
  Existing users see dropdowns without any change.
- **Escape hatch:** your own handler with `GetViewRect`/`DoPaint`/`CursorChanged`
  plus the three popup methods. Then the host app draws the popup itself
  (e.g. a Fyne `widget.List` over the canvas, position from `PopupSize`).
- A wrapper that only embeds `RenderHandler` (not the image handler)
  gets no popup methods and stays with the discard variant until it
  implements them itself.
- The mouse stays with CEF: forward clicks on your own popup in view coordinates
  to `Browser`; Chromium routes them into the popup (incl. selection).

---

### Auth challenges (Basic, Digest, NTLM, Proxy) — `AuthHandler`

Every HTTP auth challenge goes to the client handler. Without `WithAuthHandler`
(or with `AuthDefault`) the request is cancelled; the page shows the 401 response.
There is no CEF default dialog any more, because the `disable-chrome-login-prompt` switch
applies process-wide (see `cef-internals.md`). With `WithAuthHandler`:

```go
type AuthHandler interface {
    OnAuthCredentials(instance Browser, request AuthRequest, callback AuthCallback) AuthAction
}
```

- `AuthHandled`: the host takes over (e.g. a Fyne login form). Call `callback.Continue(user, pass)`
  or `callback.Cancel()` exactly once, also later and from any goroutine.
  If neither is ever called, the request hangs.
- `AuthDefault`: the request is cancelled (same as no handler).
- `AuthRequest`: `OriginURL`, `Host`, `Port`, `Realm`, `Scheme`, `IsProxy`.

**Status: verified** (CEF 156.0.27, Windows). The handler is called for every challenge.
With `Continue` the protected page loads; with `Cancel` it shows the 401 response.
The Fyne UI with its own login dialog is confirmed in the test harness `cmd/`.

---

## Own window (Windows and macOS)

```go
client := app.NewClient(clientOpts...)   // WITHOUT WithRenderHandler
br, err := client.NewBrowser(
    gomium.WithWindow("My App", 1200, 860),
    gomium.WithInitialUrl("https://127.0.0.1:7337/ui/"),
)
```

CEF creates the window itself; mouse, keyboard, IME, resizing and scrolling are
handled by Chromium. No `RenderHandler`, no input forwarding, no `Redraw`.

- **`WithRenderHandler` beats `WithWindow`.** If the client has a
  render handler, rendering is offscreen and `WithWindow` is ignored.
- **Window closed = browser dead.** `LifeSpanHandler.OnBeforeClose` is the
  signal; reopening means `NewBrowser`, not reviving.
- Title and position: `WithWindowBounds(title, image.Rect(x, y, x+w, y+h))`.
- Neither Windows nor macOS: `NewBrowser` returns "windowed browsers are only
  implemented on windows and macos".

### Windows

The window is created via `CreateWindowEx` with
`WS_OVERLAPPEDWINDOW|WS_CLIPCHILDREN|WS_CLIPSIBLINGS|WS_VISIBLE`.

- **Both loop variants work.** `WithMultiThreadedMessageLoop(true)` lets
  CEF pump its own thread. With an external pump
  (`WithMessagePumpWorker`), every worker additionally drains the
  Win32 message queue of the calling thread — CEF's external pump
  itself never calls `PeekMessage`. The worker must therefore run on the
  `cef_initialize` thread; input latency is at most
  `WithMessagePumpWorkerInterval` (default 1000/30 ms). While the window is
  being dragged/resized, Windows' modal loop runs and CEF pauses until it ends.
- **Blocking calls on the UI thread pump on their own.** With an external pump,
  the `cef_initialize` thread is CEF's UI thread. `NewBrowser`, `Client.Close`,
  cookie operations, `PrintToPdf` and DevTools `Call` wait there for a
  UI-thread callback and therefore drive `cef_do_message_loop_work`
  themselves while waiting — otherwise they would run into their timeout. This
  does not work from inside a CEF callback (no reentrant pumping).
- **Runtime style is pinned to Alloy.** Chrome style would bring Chromium's
  own browser UI; the window is meant to show only the page.

### macOS

```go
br, err := client.NewBrowser(
    gomium.WithWindow("My App", 1200, 860),
    // optional: into an existing NSView instead of a window of its own
    gomium.WithParentView(view),
    gomium.WithInitialUrl("https://127.0.0.1:7337/ui/"),
)
```

- **Call `InstallMacApplication()` first** (see Minimal skeleton). Without
  an `NSApplication` that implements `CefAppProtocol`, CEF does not start.
- **Without `WithParentView`, CEF creates the `NSWindow` itself**; the title comes from
  `WithWindow`. With a parent view, the bounds are the frame *inside* the
  view, and the title belongs to the host's window — it is ignored.
- **`WithMultiThreadedMessageLoop` does not exist on macOS.** CEF requires
  its message loop on the main thread; what remains is the external pump
  (`WithMessagePumpWorker`), processed on the `cef_initialize` thread.
- Runtime style is pinned to Alloy as on Windows; with a parent view
  CEF forces Alloy anyway.

---

## Drag & Drop (file paths)

```go
client := app.NewClient(gomium.WithFileDropHandler(func(bi gomium.Browser, files []string) {
    data, _ := json.Marshal(files)
    bi.ExecuteJavascript("window.__droppedFiles = "+string(data), "", 0)
}))
```

`on_drag_enter` fires **when the pointer enters the window** — not on
release. The handler is therefore the source of the paths; the trigger is the
ordinary HTML5 `drop` event in the page. Usual pattern: Go puts the
paths into a JS variable, and the page's `drop` handler reads them from there.
Without this detour, JS only knows the file names, never the paths.

- `WithFileDropHandler(fn)` reports only file drags (`is_file`) with at least
  one path and always lets the drag continue.
- `WithDragHandler(h)` is the full version: `OnDragEnter(bi, files) bool`;
  `true` cancels the drag — then the page sees no `drop` either.
- **Synchronous on the UI thread**, because the return value decides the drag.
  Do not block.
- The page must answer `dragover` with `preventDefault`, otherwise there is
  no `drop` and Chromium navigates to the file.

---

## Handler contracts

> These interfaces are written out in `contracts.go`, not aliased —
> `go doc gomium.LoadHandler` and friends therefore show the complete
> method set including comments. For the types the library *provides*
> (`Browser`, `CefFrame`, `Cookie`, …), this file remains the reference.

All handler callbacks come from the **browser process UI thread**.
Events (Display/Load/Download) are dispatched onto separate goroutines —
not critical. **Decision callbacks run synchronously** and must not
block; pattern: capture the callback object, start a goroutine, return immediately.

### LoadHandler

```go
OnLoadingStateChange(b, isLoading, canGoBack, canGoForward bool)  // wraps every load, carries the navigation state
OnLoadStart(b, isMainFrame bool)
OnLoadEnd(b, isMainFrame bool, httpStatus int)
OnLoadError(b, isMainFrame bool, errorCode int, errorText, failedUrl string)  // net error codes
```

### PopupHandler

```go
type PopupHandler func(b Browser, req PopupRequest) bool  // true = suppress popup
// PopupRequest: TargetUrl, TargetFrameName, TargetDisposition (WodNewForegroundTab...), UserGesture
```

Caution: `window.open()` **without a user gesture** is swallowed by Chromium's
popup blocker before the handler sees anything.

Without `WithPopupView`, CEF opens a native window for an allowed popup.

### WithPopupView

```go
type PopupViewFactory func(opener Browser, req PopupRequest) (PopupView, bool)
type PopupView struct {
    Options []ClientOption   // must contain WithRenderHandler
    Attach  func(Browser)    // receives the popup browser after creation
}
```

The popup runs windowless (Alloy) with its **own client**, built from
`Options`. As a result it shares no render handler with the opener,
and the opener's binding guard does not apply to the popup: without its own
guard (`WithBindingGuard`) it falls back to the default guard. The client
is closed as soon as its popup browser is closed. `Attach` and the
factory run on the UI thread and must not block. If the
factory returns `false` or `Options` contains no render handler, CEF falls
back to the native window.

### DownloadHandler

```go
CanDownload(b, url, method string) bool                  // synchronous, answer quickly
OnBeforeDownload(b, item DownloadItem) (path string)     // "" = suggested name in the temp directory
OnDownloadUpdated(b, item DownloadItem, ctl DownloadControl)  // ctl only valid during the call
```

`DownloadItem` is a pure Go snapshot (Id, Url, SuggestedFileName,
FullPath, MimeType, Received-/TotalBytes, CurrentSpeed, PercentComplete,
InProgress/Complete/Canceled/Interrupted, InterruptReason).
`DownloadControl`: `Cancel/Pause/Resume`.

### JsDialogHandler

```go
OnJsDialog(b, originUrl string, t JsDialogType, msg, defaultPrompt string, cb JsDialogCallback) JsDialogAction
OnBeforeUnloadDialog(b, msg string, isReload bool, cb JsDialogCallback) JsDialogAction
```

`JsDialogType`: `Alert`, `Confirm`, `Prompt`.
`JsDialogAction`:

- `JsDialogHandled` — your own dialog; call `cb.Continue(success, userInput)` **exactly once** (async allowed)
- `JsDialogDefault` — CEF default behavior
- `JsDialogSuppressed` — silently discard the dialog; for beforeunload = leave the page without asking

### ContextMenuHandler

```go
RunContextMenu(b, params ContextMenuParams, model []ContextMenuItem, cb ContextMenuCallback) bool
```

- `params`: X/Y, LinkUrl, SourceUrl, PageUrl, SelectionText, IsEditable,
  TypeFlags (`CmTypeLink|CmTypeSelection|...`), EditFlags (`CmEditCanCopy|...`)
- `model`: Chromium's default menu as a snapshot (CommandId, Label, Type,
  Enabled, Checked, SubMenu) — the basis for your own Fyne menu
- Finish exactly once: `cb.Continue(commandId)` executes a Chromium command
  (`MenuIdCopy`, `MenuIdPaste`, `MenuIdBack`, …) or `cb.Cancel()`
- `return false` = CEF default (in OSR: a free-standing native window — usually
  unwanted; use `WithContextMenuSuppressed` for that)

### FindHandler

```go
br.Find("search text", true, false, false)   // new search
br.Find("search text", true, false, true)    // findNext: same search, next match
gomium.WithFindHandlerFunc(func(b gomium.Browser, r gomium.FindResult) { ... })
// FindResult: Identifier, Count, ActiveMatchOrdinal, FinalUpdate
// Several updates per search; FinalUpdate marks the last one.
br.StopFinding(true)                      // true = clear the selection
```

### CertificateErrorHandler

```go
type CertificateErrorHandler func(b Browser, certError int, requestUrl string,
    cb CertificateErrorCallback) CertificateAction
// certError = net error code, e.g. -202 ERR_CERT_AUTHORITY_INVALID
```

`CertificateAction`:

- `CertificateDeny` — cancel the request (CEF default)
- `CertificateAccept` — load despite the invalid certificate
- `CertificateDecideAsync` — later call `cb.Continue(allow)` **exactly once** (any goroutine allowed)

Runs synchronously on the UI thread — decide quickly or go async.
Is **never** called when the app runs with `WithIgnoreCertificateErrors(true)`.
For the standard case "only these hosts": `WithCertificateErrorsAcceptedFor("host:port", ...)`.

### RequestHandler + ResourceInterceptor

```go
type RequestHandler interface {
    OnProtocolExecution(b, url) (stopLoad bool)   // mailto:, custom schemes
    OnBeforeBrowse(b, url) (stopLoad bool)        // allow/stop navigation
    OnBeforeResourceLoad(b, req *Request, cb ResourceLoadAsyncResult) ResourceLoadResult
    // ResourceLoadContinue | ResourceLoadContinueAsync (later cb.Continue/Cancel) | ResourceLoadCancel
}

// optionally implement in addition:
type ResourceInterceptor interface {
    InterceptResource(b, url, method) ResourceContentFn  // IO thread, return quickly
}
// ResourceContentFn runs on its own goroutine and may block:
// return &ResourceContent{Data: ..., MimeType: "application/pdf", Status: 0 /* =200 */}
// MimeType is the bare type ("text/html"); a charset parameter is
// split off and set separately, everything else is dropped.
// nil = ERR_ABORTED. No redirect: the original URL stays, CSP/same-origin untouched.
```

### Custom schemes

```go
opts := []gomium.Option{
    gomium.WithCustomScheme("help", gomium.SchemeStandard|gomium.SchemeSecure),
}
app, _ := gomium.New(opts...)          // browser process
gomium.ExecuteSubprocess(opts...)      // helper processes — THE SAME list
```

Requests to the scheme land in the `ResourceInterceptor` of the browser's
`RequestHandler`, just like an intercepted http request. Here, however,
`nil` means **error** and not "load normally": behind a custom scheme
there is no network to fall back on.

The option must go to `New` **and** to `ExecuteSubprocess`: Chromium
registers schemes per process, and a renderer that does not know the scheme
refuses to even parse its URIs — the page ends in
`ERR_UNKNOWN_URL_SCHEME`, even though the browser process could serve it.

| Flag (`SchemeOptions`) | Effect |
|---|---|
| `SchemeStandard` | origin + normal URL parsing (`scheme://host/path`); basis for everything except `SchemeLocal` |
| `SchemeLocal` | like `file://`: no links from other origins |
| `SchemeDisplayIsolated` | displayable only from content of the same scheme |
| `SchemeSecure` | like https: secure context, no mixed-content warning |
| `SchemeCorsEnabled` | CORS requests allowed |
| `SchemeCspBypassing` | exempt from CSP |
| `SchemeFetchEnabled` | `fetch()` / XHR allowed |

The first request must already find the handler in place: build the browser
without `WithInitialUrl` (it then sits on `about:blank`), set the handler, then
`LoadUrl`.

---

## DevTools protocol (`WithDevTools`)

No remote debugging port needed. Calls and events run in-process via
CEF's DevTools observer.

```go
client := gomium.NewClient(
    gomium.WithDevTools(func(instance gomium.Browser, dt gomium.DevTools) {
        // runs on every attach: after creation and after every detach
        dt.Call(ctx, "Runtime.enable", nil)
        dt.On("Runtime.consoleAPICalled", func(params json.RawMessage) {
            log.Println(string(params))
        })
    }),
)
```

- `Call(ctx, method, params)` waits for the result. Protocol errors are
  `*DevToolsError`. A detach during an open call yields
  `ErrDevToolsDetached`, closing the browser yields `ErrDevToolsClosed`.
- Events are delivered in arrival order. The handler runs on its own
  goroutine, not on the UI thread.
- Lifecycle: an attach holds until CEF detaches the agent. Then open
  `Call`s expire with `ErrDevToolsDetached`, and all `On` handlers are removed.
  The receiver then runs again and re-registers domains and handlers. An
  `On` outside the receiver therefore only holds until the next detach.
- In the live test, **no** operation triggered a detach, tested with
  `cmd/harness -devtools`:
  - Cross-site navigation (`example.com` to `example.org`): the receiver
    did not run again, `On` handlers kept firing.
  - `Page.crash`: the call ran into the timeout, no detach.
  - Renderer crash via `chrome://crash`: Blink reported the crash, the
    agent stayed attached.
  The agent is the browser-level agent. A detach apparently only comes when
  CEF terminates the agent itself. As a result, the re-attach path (receiver runs again)
  has not yet been triggered in the live test. The unit tests cover
  it, not the real CEF side.
- The receiver runs on its own goroutine. The agent only attaches once CEF
  sends it a message. For this, the session internally sends a bootstrap
  command on attach. A receiver that calls `Page.enable` immediately therefore sees
  `about:blank` if the page is still loading.
- The receiver is also called for popups.
- After the browser is closed, new calls fail with `ErrDevToolsClosed`.

### Security: remote debugging port

`cmd/harness` sets `remote_debugging_port` (default 6696). Through it, any local process
has full control over the browser, including cookies and
page data. `WithDevTools` does not need the port. For your own applications,
omit the port or set it to a fixed value with access protection.

### Live test: `cmd/harness -devtools`

```
go run ./cmd/harness -devtools
```

The probe receiver (`cmd/harness/devtools_probe.go`) logs:

1. `devtools: attached` with `attach=1`
2. `Page.enable`, `Runtime.evaluate` (URL and title) and `Page.loadEventFired`
3. After the first load: `Page.navigate` to `https://example.org/`
4. Expected on a re-attach: a second `devtools: attached` with `attach=2`
   and `evaluate` with the new URL. In the live run this second attach did not
   come with the cross-site navigation. `attach=1` remains; the probe thus shows
   the actual state.

Alternatively `GOMIUM_DEVTOOLS_TEST=1`. The other `GOMIUM_*_TEST` switches
are independent of it.

---

## Cookies

```go
cm := gomium.GlobalCookieManager()           // global profile
cm  = ctx.CookieManager()                    // per RequestContext

cm.SetCookie("https://host/", gomium.Cookie{ Name: "session", Value: "...",
    Path: "/", Secure: true, Expires: &t })  // Expires nil = session cookie
cm.Cookies()                                 // all
cm.CookiesForUrl(url, includeHttpOnly)       // as a request would get them
cm.DeleteCookies(url, name)                  // "" args widen the scope: everything / host-wide
cm.Flush()                                   // store to disk
```

All calls block until the store confirms (timeout 10 s) — do **not** call them
from inside CEF callbacks. Persistence across restarts only with a cache path.
`Cookie`: Name, Value, Domain (empty = host cookie), Path, Secure, HttpOnly,
Expires, SameSite (`CookieSameSite*`), Priority (`CookiePriority*`);
Creation/LastAccess only filled when reading.

---

## RequestContexts (browser profiles)

A separate storage universe per context: cookies, localStorage, cache, HTTP auth.

```go
// Incognito (in-memory, nothing persisted):
ctx, _ := gomium.NewRequestContext()

// Persistent - the directory MUST be under WithRootCachePath:
ctx, _ := gomium.NewRequestContext(
    gomium.WithContextCachePath(`C:\data\app\tenant-a`),
    gomium.WithPersistSessionCookies(true),
    gomium.WithAcceptLanguageList("de-DE,de"),
)

br, _ := client.NewBrowser(gomium.WithRequestContext(ctx), ...)
```

`RequestContext`: `CookieManager()`, `CachePath()`, `IsGlobal()`, `Release()`.
`GlobalRequestContext()` returns the default profile through the same interface.

---

## JavaScript

### Running JS

```go
br.ExecuteJavascript(src, url, line)                          // fire-and-forget
br.Cef().MainFrame().ExecuteJavaScriptWithResult(src, nil, 0,
    func(result *gomium.Exported, err *gomium.ExceptionError) { ... })
// result.Native() -> Go value (map/slice/string/float/bool/nil)
```

### V8 lifecycle

`gomium.WithV8Listener(gomium.V8ListenerFuncs(created, released))` — pass it to
both `New` and `ExecuteSubprocess`, since V8 contexts exist only in the render
process (`CefApp.AddV8Listener` on the browser-process app never fires). Called per
frame on context creation (the place for JS injection, e.g. a localStorage shim). Both run
synchronously on the renderer main thread, before any page script of the new context —
do not block in them; browser and frame are only valid during the call.

---

## Low level (`Cef*` types)

`Browser.Cef()` returns `CefBrowser` (borrowed — do not hold it past the
browser's close): `MainFrame()`, `Frames()`,
`IsPopup()`, `StopLoad()`, …
`CefFrame`: `LoadUrl`, `ExecuteJavaScript(WithResult)`, `V8()`, `IsMain()`, …

Rules:

- Release `Cef*` objects obtained from CEF with `Release()` where the API
  documents it; the high-level API (`Browser`, handler snapshots) manages itself.
- `CefCurrentlyOn(TidUI / TidIO / TidRenderer)` for thread assertions.
- `SetDebugLogger` / `SetMemoryLogger` for bridge/refcount tracing.

---

## Test harness `cmd/harness`

`go build -o harness.exe ./cmd/harness`, scenarios via env (full list: [`cmd/harness/README.md`](../cmd/harness/README.md)):

| Variable | Scenario |
|---|---|
| `GOMIUM_RUN_SECONDS` | run time (default 20) |
| `GOMIUM_TEST_URL` | start URL |
| `GOMIUM_DEBUG_PORT` | DevTools port (default 6696) |
| `GOMIUM_COOKIE_TEST=1` | CookieManager: set/read/delete |
| `GOMIUM_CONTEXT_TEST=1` | RequestContext isolation + browser in the context |
| `GOMIUM_POPUP_TEST=1` | trusted click on `target=_blank` → opener routing |
| `GOMIUM_DOWNLOAD_TEST=1` | attachment download into the temp directory |
| `GOMIUM_DIALOG_TEST=1` | answer alert/confirm/prompt from Go + context menu |
| `GOMIUM_NAV_TEST=1` | navigation sequence (location.href, link click, window.open) |
| `GOMIUM_SOFTNAV_TEST=1` | SPA soft navigation (regression test for the profiler crash) |
| `GOMIUM_INTERCEPT_TEST=1` | ResourceInterceptor serves PDFs locally |
| `GOMIUM_RESIZE_STRESS=1` | OSR resize endurance test |
| `GOMIUM_WINDOW_TEST=1` | real window instead of OSR, drop page on `127.0.0.1:18776`; dragging files in shows names (JS) and paths (Go) side by side |
| `GOMIUM_MT_LOOP=1` | multithreaded message loop instead of an external pump |

The harness uses its own profile `%TEMP%\gomium-test-profile`.

---

## Known pitfalls

1. **Process singleton:** two CEF apps without (distinct) `WithCachePath`
   → the second one does not start ("Opening in existing browser session").
2. **Popup blocker:** `window.open` without a user gesture never reaches `PopupHandler`.
3. **Persistence:** without `WithCachePath`, cookies **and** localStorage are in-memory.
4. **RequestContext paths** must be under `root_cache_path`.
5. **Decision callbacks** (CanDownload, PopupHandler, OnJsDialog,
   RunContextMenu, InterceptResource, CertificateErrorHandler) run synchronously
   on the UI/IO thread — never block, never call CookieManager or the like from them.
   The same applies to the blocking browser methods `PrintToPdf`, `ZoomLevel`,
   `IsAudioMuted`: do not call them from inside CEF callbacks (risk of deadlock).
6. **Callback objects** (`JsDialogCallback`, `ContextMenuCallback`,
   `DownloadControl`) must be used exactly once, or only within the documented time window.
7. **Shim:** add new `cef_*` exports to `internal/cefshim/cef_shim_symbols.txt`
   and run `go generate ./internal/cefshim`, otherwise you get an
   undefined reference.
8. **Never remove the three command-line switches in `app_bridge.c`.** They fix
   *three independent* crashes and only happen to sit next to each other:
   `--disable-stack-profiler` (+ features `ThreadGroupProfiler`,
   `SamplingProfilerReporting`) against the unwind crash across Go frames,
   `--disable-gpu-watchdog` against the 15 s kill of the GPU process, and
   `ImmersiveReadAnything` off against the soft-navigation null deref.
   Details: [cef-internals.md](cef-internals.md).
9. **No interior Go pointers to C.** Passing a `&goStruct.field` to a
   CEF function makes cgo check the enclosing object; if it finds
   Go pointers there (a `string` is enough), you get
   `cgo argument has Go pointer to unpinned Go pointer`. Whether the check
   fires depends on the `go` directive of the **main module** — the call
   survives gomium's own test app (`go 1.23`) and dies in the host with a
   newer toolchain. The pattern in `browser_window_windows.go` and
   `browser_window_darwin.go`: build it in C memory with `cefString`, copy the
   struct value into it, free only the wrapper.
10. **Call platform-specific symbols behind build tags.**
    `GetModuleHandle`, `WithModuleHandle` and `WithMultiThreadedMessageLoop`
    exist only on Windows, `InstallMacApplication` and `WithParentView`
    only on macOS — cross-platform hosts need their own
    `_windows.go`/`_darwin.go` files for them (see `examples/` for a model).
