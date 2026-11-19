package gomium

import (
	"github.com/rs/zerolog"
	cef "github.com/timzifer/gomium/internal/cef"
	"image"
	"image/draw"
	"net/url"
	"time"
)

// Every exported function of the binding, forwarded. These are wrappers
// rather than `var F = cef.F` bindings so that parameter names and doc
// comments survive into go doc, and so that they cannot be reassigned.

// CEFVersion returns the full CEF version string this binary was compiled
// against, e.g. "151.3.14+g5d67476+chromium-151.0.7922.72". It reflects the
// headers used at build time and does not touch the loaded library, so it is
// safe to call at any point - including before Run and against a mismatched
// libcef.
func CEFVersion() string { return cef.CEFVersion() }

func CefCurrentlyOn(id CefThreadId) bool { return cef.CefCurrentlyOn(id) }

// ExecuteSubprocess runs the CEF helper-process loop and terminates the
// process with its exit code - it does not return on success. Call it from
// main() when IsSubprocess() is true, before any other initialization.
// Returns an error only when setup fails or the process is not actually a
// CEF helper process.
func ExecuteSubprocess(opts ...Option) error { return cef.ExecuteSubprocess(opts...) }

func Get() (CefApp, bool) { return cef.Get() }

// GlobalCookieManager returns the cookie manager of the global request
// context. Cookie data lives at the app's WithCachePath location, or in
// memory when no cache path is set.
func GlobalCookieManager() CookieManager { return cef.GlobalCookieManager() }

// GlobalRequestContext returns the context that browsers without an explicit
// context use. Useful to compare via IsGlobal or to reach its CookieManager
// through the same interface.
func GlobalRequestContext() (RequestContext, error) { return cef.GlobalRequestContext() }

// IsSubprocess reports whether the current process was launched by CEF as a
// helper process (renderer, gpu-process, utility, crashpad-handler, ...).
// CEF starts these as new instances of the main executable with a --type=
// switch. Hosts must detect this before any application logic runs -
// single-instance guards or setup work in the main flow would otherwise
// kill or delay the helper and leave the browser hanging.
func IsSubprocess() bool { return cef.IsSubprocess() }

func New(opts ...Option) (CefApp, error) { return cef.New(opts...) }

func NewFocusEvent(gained bool) Event { return cef.NewFocusEvent(gained) }

func NewImageRenderHandler(img draw.Image, imageChanged func(image.Image)) RenderHandler {
	return cef.NewImageRenderHandler(img, imageChanged)
}

func NewKeyDownEvent(nativeKeyCode, windowsKeyCode int, isSystemKey bool, modifiers ...KeyEventModifier) Event {
	return cef.NewKeyDownEvent(nativeKeyCode, windowsKeyCode, isSystemKey, modifiers...)
}

func NewKeyTypedEvent(r rune, modifiers ...KeyEventModifier) Event {
	return cef.NewKeyTypedEvent(r, modifiers...)
}

func NewKeyUpEvent(nativeKeyCode, windowsKeyCode int, isSystemKey bool, modifiers ...KeyEventModifier) Event {
	return cef.NewKeyUpEvent(nativeKeyCode, windowsKeyCode, isSystemKey, modifiers...)
}

func NewMouseClickEvent(x, y int, button MouseButton, direction MouseButtonDirection) Event {
	return cef.NewMouseClickEvent(x, y, button, direction)
}

func NewMouseMovedEvent(x, y int, boundaryLeft ...bool) Event {
	return cef.NewMouseMovedEvent(x, y, boundaryLeft...)
}

// NewRequestContext creates an isolated browsing profile. Without options the
// context lives purely in memory (incognito). Pass the result to browsers via
// WithRequestContext.
func NewRequestContext(opts ...RequestContextOption) (RequestContext, error) {
	return cef.NewRequestContext(opts...)
}

func NewResizedEvent() Event { return cef.NewResizedEvent() }

func NewScrollEvent(x, y int, deltaX, deltaY int) Event {
	return cef.NewScrollEvent(x, y, deltaX, deltaY)
}

func SetDebugLogger(zLogger zerolog.Logger) { cef.SetDebugLogger(zLogger) }

func SetMemoryLogger(zLogger zerolog.Logger) { cef.SetMemoryLogger(zLogger) }

func V8ListenerFuncs(created, released func(b CefBrowser, f CefFrame)) V8Listener {
	return cef.V8ListenerFuncs(created, released)
}

// WithAcceptLanguageList sets the comma-delimited Accept-Language header and
// navigator.language value for browsers using this context.
func WithAcceptLanguageList(languages string) RequestContextOption {
	return cef.WithAcceptLanguageList(languages)
}

// WithBackgroundColor sets the browser background as ARGB, e.g. 0xFFFFFFFF
// for opaque white or 0 for a transparent surface.
func WithBackgroundColor(argb uint32) BrowserOption { return cef.WithBackgroundColor(argb) }

func WithBrowserCreationTimeout(timeout time.Duration) ClientOption {
	return cef.WithBrowserCreationTimeout(timeout)
}

// WithBrowserSubprocessPath points CEF at a separate executable for its helper
// processes (renderer, gpu, utility, ...). path must be absolute.
//
// Leave it unset unless the helpers really are a different binary. Empty means
// the main executable on Windows and Linux, and on macOS it means CEF derives
// the helper from the app bundle - "Contents/Frameworks/<app> Helper.app/
// Contents/MacOS/<app> Helper", which is the layout tools/cefbundle produces.
func WithBrowserSubprocessPath(path string) Option { return cef.WithBrowserSubprocessPath(path) }

func WithCachePath(cachePath string) Option { return cef.WithCachePath(cachePath) }

// WithCertificateErrorHandler decides per request whether an invalid TLS
// certificate is accepted - the surgical alternative to the global
// WithIgnoreCertificateErrors app option.
func WithCertificateErrorHandler(handler CertificateErrorHandler) ClientOption {
	return cef.WithCertificateErrorHandler(cef.CertificateErrorHandler(handler))
}

// WithCertificateErrorsAcceptedFor accepts invalid certificates for exactly
// the given hosts (case-insensitive, an optional port on the host entry is
// matched too) and denies everything else.
func WithCertificateErrorsAcceptedFor(hosts ...string) ClientOption {
	return cef.WithCertificateErrorsAcceptedFor(hosts...)
}

// WithContextCachePath places the context's profile data in dir. The
// directory must be equal to or a child of the app's root cache path (set
// via WithCachePath or WithRootCachePath) - CEF rejects paths outside of it.
// Without this option the context is in-memory: nothing persists, which
// makes NewRequestContext() an incognito profile.
func WithContextCachePath(dir string) RequestContextOption { return cef.WithContextCachePath(dir) }

func WithContextMenuHandler(handler ContextMenuHandler) ClientOption {
	return cef.WithContextMenuHandler(handler)
}

// WithContextMenuSuppressed swallows every right-click menu - in windowless
// rendering the default menu would appear as a detached native window.
func WithContextMenuSuppressed() ClientOption { return cef.WithContextMenuSuppressed() }

// WithCustomScheme registers a scheme the embedder serves itself, e.g.
//
//	gomium.WithCustomScheme("help", gomium.SchemeStandard|gomium.SchemeSecure)
//
// Requests to it reach the ResourceInterceptor of the browser's
// RequestHandler, exactly like an intercepted http request; returning nil
// there fails the request instead of falling back to a network load, because
// nothing else can serve the scheme.
//
// The option MUST be passed identically to New and to ExecuteSubprocess:
// Chromium registers schemes per process, and a renderer that does not know
// the scheme refuses to parse its URLs.
func WithCustomScheme(name string, options SchemeOptions) Option {
	return cef.WithCustomScheme(name, options)
}

// WithV8Listener registers a listener for the V8 contexts of the render
// process. Pass it to New and to ExecuteSubprocess, like every option - the
// listener runs in the render process, synchronously on its main thread.
func WithV8Listener(listener V8Listener) Option { return cef.WithV8Listener(listener) }

// WithBinding exposes fn to page JavaScript as path, e.g. "app.getItems".
// fn is func(...) error, func(...) T or func(...) (T, error). It is only
// reachable through a BindingGuard that admits the calling browser (see
// WithBindingGuard). A bad path or signature is reported by New.
func WithBinding(path string, fn any) Option {
	return cef.WithBinding(path, fn)
}

// WithBindingGuard sets which binding calls of the browser's frames may run.
// Without a guard every call is rejected.
func WithBindingGuard(guard BindingGuard) ClientOption {
	return cef.WithBindingGuard(guard)
}

// WithDefaultBindingGuard sets the guard for every browser without a guard of
// its own (WithBindingGuard), popup browsers included.
func WithDefaultBindingGuard(guard BindingGuard) Option {
	return cef.WithDefaultBindingGuard(guard)
}

// SameOrigin reports whether a and b share scheme, host and effective port.
// A nil address matches nothing.
func SameOrigin(a, b *url.URL) bool { return cef.SameOrigin(a, b) }

func WithDefaultInitialUrl(initialUrl string) ClientOption {
	return cef.WithDefaultInitialUrl(initialUrl)
}

func WithDisplayHandler(handler DisplayHandler) ClientOption { return cef.WithDisplayHandler(handler) }

func WithDisplayHandlerFuncs(onTitleChange func(Browser, string), onAddressChange func(Browser, *url.URL), onConsoleMessage func(Browser, string, int, string), onFaviconUrlChange func(Browser, []*url.URL)) ClientOption {
	return cef.WithDisplayHandlerFuncs(onTitleChange, onAddressChange, onConsoleMessage, onFaviconUrlChange)
}

// WithDownloadDirectory accepts every download and stores it in dir under the
// suggested file name. Existing files are overwritten by Chromium's usual
// uniquify rules. onUpdate may be nil; otherwise it receives every progress
// update including the final Complete/Canceled/Interrupted state.
func WithDownloadDirectory(dir string, onUpdate func(instance Browser, item DownloadItem)) ClientOption {
	return cef.WithDownloadDirectory(dir, onUpdate)
}

func WithDownloadHandler(handler DownloadHandler) ClientOption {
	return cef.WithDownloadHandler(handler)
}

func WithDragHandler(handler DragHandler) ClientOption { return cef.WithDragHandler(handler) }

func WithFileAccessFromFileUrls(fileAccessFromFileUrls bool) BrowserOption {
	return cef.WithFileAccessFromFileUrls(fileAccessFromFileUrls)
}

// WithFileDropHandler reports the native paths of a file drag as soon as it
// enters the browser window and lets the drag continue. The drop itself stays
// an ordinary HTML5 drop event in the page - this option only supplies what
// the page cannot see, the paths behind the dropped files.
func WithFileDropHandler(fn func(instance Browser, files []string)) ClientOption {
	return cef.WithFileDropHandler(fn)
}

func WithFindHandler(handler FindHandler) ClientOption { return cef.WithFindHandler(handler) }

// WithFocusHandler observes browser focus (Tab out, focus gain). Optional.
func WithFocusHandler(handler FocusHandler) ClientOption { return cef.WithFocusHandler(handler) }

// WithKeyboardHandler sees every key before and after the page. Optional.
func WithKeyboardHandler(handler KeyboardHandler) ClientOption {
	return cef.WithKeyboardHandler(handler)
}

// WithAuthHandler answers HTTP and proxy auth challenges. Every challenge is
// routed here; without a handler (or with AuthDefault) the request is cancelled.
func WithAuthHandler(handler AuthHandler) ClientOption { return cef.WithAuthHandler(handler) }

// WithFrameHandler observes frame lifecycle (iframes, main frame swaps).
func WithFrameHandler(handler FrameHandler) ClientOption { return cef.WithFrameHandler(handler) }

// WithFileDialogHandler takes over file pickers (<input type=file>, save
// dialogs). Optional; without it CEF shows its native dialog.
func WithFileDialogHandler(handler FileDialogHandler) ClientOption {
	return cef.WithFileDialogHandler(handler)
}

// WithFindHandlerFunc receives the results of Browser.Find searches.
func WithFindHandlerFunc(fn func(instance Browser, result FindResult)) ClientOption {
	return cef.WithFindHandlerFunc(fn)
}

func WithIgnoreCertificateErrors(ignore bool) Option { return cef.WithIgnoreCertificateErrors(ignore) }

func WithInitialUrl(url string) BrowserOption { return cef.WithInitialUrl(url) }

func WithInitializationTimeout(duration time.Duration) Option {
	return cef.WithInitializationTimeout(duration)
}

func WithJsDialogHandler(handler JsDialogHandler) ClientOption {
	return cef.WithJsDialogHandler(handler)
}

// WithJsDialogsSuppressed drops every alert/confirm/prompt silently (confirm
// and prompt resolve as canceled) and leaves pages without asking on
// onbeforeunload. The right default for embeds that never want browser
// dialogs to stall the page.
func WithJsDialogsSuppressed() ClientOption { return cef.WithJsDialogsSuppressed() }

func WithLifeSpanHandler(handler LifeSpanHandler) ClientOption {
	return cef.WithLifeSpanHandler(handler)
}

// WithLifeSpanHandlerFuncs takes the three life-span events; every one of
// them may be nil.
func WithLifeSpanHandlerFuncs(onAfterCreated, onBeforeClose func(), doClose func() bool) ClientOption {
	return cef.WithLifeSpanHandlerFuncs(onAfterCreated, onBeforeClose, doClose)
}

func WithLoadHandler(handler LoadHandler) ClientOption { return cef.WithLoadHandler(handler) }

// WithLoadHandlerFuncs registers load callbacks without implementing the full
// LoadHandler interface. Individual callbacks may be nil.
func WithLoadHandlerFuncs(onLoadingStateChange func(instance Browser, isLoading, canGoBack, canGoForward bool), onLoadStart func(instance Browser, isMainFrame bool), onLoadEnd func(instance Browser, isMainFrame bool, httpStatusCode int), onLoadError func(instance Browser, isMainFrame bool, errorCode int, errorText, failedUrl string)) ClientOption {
	return cef.WithLoadHandlerFuncs(onLoadingStateChange, onLoadStart, onLoadEnd, onLoadError)
}

// WithLocalesDirPath points CEF at the directory holding the locale .pak
// files. path must be absolute.
//
// Empty means the "locales" directory next to the executable. Ignored on
// macOS for the same reason as WithResourcesDirPath.
func WithLocalesDirPath(path string) Option { return cef.WithLocalesDirPath(path) }

func WithLogSeverity(severity LogSeverity) Option { return cef.WithLogSeverity(severity) }

func WithLogger(logger zerolog.Logger) Option { return cef.WithLogger(logger) }

func WithMessagePumpWorker(pumpWorker WorkerCallback) Option {
	return cef.WithMessagePumpWorker(cef.WorkerCallback(pumpWorker))
}

func WithMessagePumpWorkerInterval(interval time.Duration) Option {
	return cef.WithMessagePumpWorkerInterval(interval)
}

// WithPersistSessionCookies also persists cookies without an expiry date.
// Only meaningful together with WithContextCachePath.
func WithPersistSessionCookies(persist bool) RequestContextOption {
	return cef.WithPersistSessionCookies(persist)
}

func WithPopupHandler(handler PopupHandler) ClientOption {
	return cef.WithPopupHandler(cef.PopupHandler(handler))
}

// WithPopupView keeps popups offscreen: a popup that PopupHandler lets through
// becomes a windowless browser with a client of its own, built from the view
// the factory returns. Without it CEF opens a native popup window.
func WithPopupView(factory PopupViewFactory) ClientOption {
	return cef.WithPopupView(cef.PopupViewFactory(factory))
}

// WithPopupsInOpener suppresses every popup window and instead navigates the
// opener browser to the popup's target URL. The pragmatic default for
// single-view embeds where target="_blank" links should not vanish.
func WithPopupsInOpener() ClientOption { return cef.WithPopupsInOpener() }

func WithRemoteDebuggingPort(port int) Option { return cef.WithRemoteDebuggingPort(port) }

func WithRenderHandler(handler RenderHandler) ClientOption { return cef.WithRenderHandler(handler) }

// WithRequestContext creates the browser inside the given browsing profile
// instead of the global context. See NewRequestContext.
func WithRequestContext(context RequestContext) BrowserOption { return cef.WithRequestContext(context) }

func WithRequestHandler(handler RequestHandler) ClientOption { return cef.WithRequestHandler(handler) }

// WithResourcesDirPath points CEF at the directory holding cef.pak,
// chrome_100_percent.pak, icudtl.dat and friends. path must be absolute.
//
// Empty means CEF looks next to the executable, which is what a distribution
// laid out the usual way provides. Set it when the resources live somewhere
// else - a shared installation directory, or a development tree where the
// binary is built into ./build and the runtime sits in ./Release.
//
// Ignored on macOS: there the resources are inside the framework bundle, and
// WithFrameworkDirPath is what points at them.
func WithResourcesDirPath(path string) Option { return cef.WithResourcesDirPath(path) }

// WithRootCachePath sets the parent directory that every cache path must
// live in - the global one and all request-context paths (see
// WithContextCachePath). Defaults to the WithCachePath value.
func WithRootCachePath(rootCachePath string) Option { return cef.WithRootCachePath(rootCachePath) }

func WithUniversalAccessFromFileUrls(fileAccessFromFileUrls bool) BrowserOption {
	return cef.WithUniversalAccessFromFileUrls(fileAccessFromFileUrls)
}

func WithUserAgent(agent string) Option { return cef.WithUserAgent(agent) }

func WithWebSecurity(webSecurity bool) BrowserOption { return cef.WithWebSecurity(webSecurity) }

func WithWebgl(webgl bool) BrowserOption { return cef.WithWebgl(webgl) }

// WithWindow creates the browser in an operating-system window of its own
// instead of rendering offscreen. The window carries no browser interface -
// no address bar, no tabs -, only the page.
//
// It is mutually exclusive with WithRenderHandler on the client: a client
// that has a render handler always renders offscreen, and the render handler
// wins. Implemented on Windows and macOS; elsewhere NewBrowser reports an
// error.
func WithWindow(title string, width, height int) BrowserOption {
	return cef.WithWindow(title, width, height)
}

// WithWindowBounds is WithWindow with an explicit position. The rectangle is
// in screen coordinates; its origin is where the window's upper left corner
// lands.
func WithWindowBounds(title string, bounds image.Rectangle) BrowserOption {
	return cef.WithWindowBounds(title, bounds)
}

func WithWindowlessFrameRate(rate int) BrowserOption { return cef.WithWindowlessFrameRate(rate) }

// WithDevTools hands the DevTools protocol of every browser of the client to
// receiver. receiver runs on its own goroutine each time the agent attaches -
// once after creation, then after every detach. Re-enable the domains you use
// there, since CEF drops subscriptions on detach. No remote debugging port is
// needed. See DevTools for details.
func WithDevTools(receiver func(instance Browser, dt DevTools)) ClientOption {
	return cef.WithDevTools(receiver)
}
