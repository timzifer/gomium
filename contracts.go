package gomium

import (
	"image"
	"net/url"

	cef "github.com/timzifer/gomium/internal/cef"
)

// The contracts you implement.
//
// Everything else in this package is a type alias over internal/cef, which
// keeps caller code compiling unchanged but costs documentation: go doc does
// not follow an alias, so `go doc gomium.LoadHandler` would print the alias
// line and stop. For the types a caller has to *write* - the handler
// interfaces and the callback signatures - that is the wrong trade, so they
// are spelled out here instead.
//
// The rule, and the condition that makes it work: what you implement is
// declared here, what you receive is an alias. Go satisfies interfaces
// structurally, so a value implementing the interface below also implements
// its internal/cef counterpart - no adapter, no wrapper object, no
// allocation. That holds only as long as everything these signatures mention
// (Browser, Request, DownloadItem, ContextMenuParams, the callback handles,
// ...) stays an alias, i.e. stays the identical type. Redeclaring both ends
// of a reference would make the method sets differ and break the
// substitution, so do not move a type out of types.go without checking what
// names it in a signature here.
//
// The five func types work the same way, except that func types are only
// convertible, not assignable - the conversion happens once inside the
// wrapper in api.go and costs nothing at runtime. They qualify because each
// appears only as a parameter of an exported top-level function.
// ResultCallback, CefTask and ResourceContentFn do not: they appear in
// methods of aliased interfaces, where the caller would have to convert.
//
// The assertions at the bottom of this file keep the two sides identical.

// LoadHandler receives browser load progress events. Callbacks originate on
// the browser process UI thread and are dispatched on their own goroutines.
type LoadHandler interface {
	// OnLoadingStateChange fires once when loading starts and once when it
	// ends (completion, cancellation or failure). It brackets all
	// OnLoadStart/OnLoadEnd/OnLoadError calls and carries the current
	// navigation state.
	OnLoadingStateChange(instance Browser, isLoading, canGoBack, canGoForward bool)
	// OnLoadStart fires after a navigation committed, before content loads.
	// Not called for same-page navigations (fragments, history state).
	OnLoadStart(instance Browser, isMainFrame bool)
	// OnLoadEnd fires when a frame finished loading.
	OnLoadEnd(instance Browser, isMainFrame bool, httpStatusCode int)
	// OnLoadError fires when a navigation fails or is canceled. errorCode is
	// a net error code from net/base/net_error_list.h.
	OnLoadError(instance Browser, isMainFrame bool, errorCode int, errorText string, failedUrl string)
}

type DisplayHandler interface {
	OnTitleChange(instance Browser, title string)
	OnAddressChange(instance Browser, url *url.URL)
	OnFaviconUrlChange(instance Browser, urls []*url.URL)
}

type ConsoleMessageHandler interface {
	DisplayHandler
	OnConsoleMessage(instance Browser, source string, line int, message string)
}

type RequestHandler interface {
	OnProtocolExecution(b Browser, url *url.URL) (stopLoad bool)
	OnBeforeBrowse(b Browser, url *url.URL) (stopLoad bool)

	OnBeforeResourceLoad(b Browser, request *Request, cb ResourceLoadAsyncResult) ResourceLoadResult
}

// ResourceInterceptor is optionally implemented by RequestHandler
// implementations. InterceptResource is called on the CEF IO thread for
// every resource request and must return quickly: either nil to let CEF
// load the resource normally, or a ResourceContentFn that produces the
// response. Unlike rewriting the request URL, serving content this way
// is not a redirect - the original URL is preserved, so CSP and
// same-origin checks are unaffected.
type ResourceInterceptor interface {
	InterceptResource(b Browser, requestUrl *url.URL, method string) ResourceContentFn
}

// ContextMenuHandler takes over context menu display. RunContextMenu
// runs synchronously on the browser process UI thread - capture the
// callback, return true, show an own (e.g. Fyne) menu and finish via
// the callback. Return false for CEF's default handling.
type ContextMenuHandler interface {
	RunContextMenu(instance Browser, params ContextMenuParams, model []ContextMenuItem, callback ContextMenuCallback) bool
}

// JsDialogHandler receives JavaScript dialog requests (alert, confirm,
// prompt, onbeforeunload). Both callbacks run synchronously on the
// browser process UI thread - decide quickly and return. To show an own
// (e.g. Fyne) dialog: capture callback, spawn a goroutine, return
// JsDialogHandled and execute callback once the user decided.
type JsDialogHandler interface {
	// OnJsDialog decides how to treat an alert/confirm/prompt.
	OnJsDialog(instance Browser, originUrl string, dialogType JsDialogType, message, defaultPrompt string, callback JsDialogCallback) JsDialogAction
	// OnBeforeUnloadDialog decides the "leave site?" dialog.
	// JsDialogSuppressed auto-confirms leaving the page.
	OnBeforeUnloadDialog(instance Browser, message string, isReload bool, callback JsDialogCallback) JsDialogAction
}

// DownloadHandler receives download events. CanDownload runs synchronously
// on the browser process UI thread and must return quickly; the other
// callbacks are dispatched on their own goroutines.
type DownloadHandler interface {
	// CanDownload decides whether a user-initiated download may start.
	CanDownload(instance Browser, url string, requestMethod string) bool
	// OnBeforeDownload determines the target path for a download. Return
	// the full file path including the file name, or an empty string to
	// download to the suggested name in the default temp directory.
	OnBeforeDownload(instance Browser, item DownloadItem) (downloadPath string)
	// OnDownloadUpdated fires on every status or progress change, before
	// and after OnBeforeDownload.
	OnDownloadUpdated(instance Browser, item DownloadItem, control DownloadControl)
}

// DragHandler is told about drag operations that enter the browser
// window from outside the browser. Called synchronously on the browser
// process UI thread, because its return value decides whether the drag
// continues - a handler that blocks freezes the drag.
//
// The callback fires when the pointer ENTERS the window, not when the
// files are released. It is therefore a source of paths, not a signal
// that something was dropped: the drop itself arrives in the page as an
// ordinary HTML5 drop event. A page that wants both has to hold on to
// the paths until its own drop handler runs.
type DragHandler interface {
	// OnDragEnter reports the native paths of a file drag. Returning
	// true cancels the drag, which means the page never sees a drop
	// event either.
	OnDragEnter(instance Browser, files []string) bool
}

// FindHandler receives find-in-page results. Dispatched on its own
// goroutine.
type FindHandler interface {
	OnFindResult(instance Browser, result FindResult)
}

type LifeSpanHandler interface {
	OnAfterCreated()
	DoClose() bool
	OnBeforeClose()
}

type RenderHandler interface {
	GetViewRect() image.Rectangle
	DoPaint(areas []DirtyArea)
	CursorChanged(cursorType CursorType)
}

// ResizableRenderHandler is implemented by render handlers whose surface
// can change size at runtime. After calling Resize, trigger
// Browser.Redraw() so CEF picks up the new view rect.
type ResizableRenderHandler interface {
	RenderHandler
	Resize(width, height int)
}

// PopupRenderHandler is an optional extension of RenderHandler for the popup
// widget CEF uses for <select> dropdowns. A RenderHandler that does not
// implement it never receives popups (they are dropped). Implementing it is
// the escape hatch: the handler then draws the popup itself, e.g. as a Fyne
// list, and forwards clicks in view coordinates - CEF routes them into the
// popup. NewImageRenderHandler implements it by compositing the popup into
// its image.
type PopupRenderHandler interface {
	// PopupShow is called when CEF shows (true) or hides (false) the popup.
	PopupShow(show bool)
	// PopupSize reports the popup rectangle in view coordinates.
	PopupSize(rect image.Rectangle)
	// DoPaintPopup receives popup pixels; area rectangles are popup-local.
	DoPaintPopup(areas []DirtyArea)
}

// ScreenInfoRenderHandler is an optional RenderHandler extension for HiDPI-
// aware hosts. Implement it on your own RenderHandler; without it CEF assumes
// scale 1 and the view rect as the screen, which renders blurry on scaled
// displays. Returning ok=false from either method keeps CEF's default for
// that query.
type ScreenInfoRenderHandler interface {
	GetScreenInfo() (info ScreenInfo, ok bool)
	// GetScreenPoint maps a view-local point to screen coordinates (device
	// pixels on Windows/Linux).
	GetScreenPoint(viewX, viewY int) (screenX, screenY int, ok bool)
}

// FocusHandler observes keyboard focus of the browser. Callbacks run on the
// browser process UI thread - keep them short.
//
// Escape hatch: hand focus back to your own widgets from OnTakeFocus. With
// next=false the user tabbed backwards out of the page, so the host should
// move focus to its previous widget.
type FocusHandler interface {
	// OnTakeFocus fires when the browser gives up focus (Tab past the last or
	// before the first element). next is true for forward Tab.
	OnTakeFocus(instance Browser, next bool)
	// OnSetFocus fires before the browser takes focus. Return true to
	// suppress the focus change, false to accept it.
	OnSetFocus(instance Browser, source FocusSource) bool
	// OnGotFocus fires after the browser received focus.
	OnGotFocus(instance Browser)
}

// KeyboardHandler sees every key before (OnPreKeyEvent) and after
// (OnKeyEvent) the page does. Callbacks run on the browser process UI thread.
//
// Use it for host shortcuts (return handled=true to swallow the key) and to
// decide which shortcuts the page may see: OnPreKeyEvent can report
// isShortcut=true to let CEF treat the key as a browser shortcut.
type KeyboardHandler interface {
	OnPreKeyEvent(instance Browser, event KeyboardEvent) (handled, isShortcut bool)
	OnKeyEvent(instance Browser, event KeyboardEvent) (handled bool)
}

// FileDialogHandler receives <input type=file>, save and folder requests.
// Runs on the browser process UI thread - return fast.
//
// Escape hatch: return handled=true to take the dialog over (e.g. show a
// Fyne file picker) and call callback once the user decided. Return
// handled=false to let CEF show its native dialog.
type FileDialogHandler interface {
	OnFileDialog(instance Browser, mode FileDialogMode, title, defaultFilePath string, acceptFilters []string, callback FileDialogCallback) (handled bool)
}

// AuthHandler answers HTTP and proxy auth challenges with a login form of
// your own. Runs synchronously on the browser process UI thread.
//
// Return AuthHandled to take the challenge over and call the callback later -
// or right away. Return AuthDefault to cancel the challenge (CEF's own
// dialog is disabled).
type AuthHandler interface {
	OnAuthCredentials(instance Browser, request AuthRequest, callback AuthCallback) AuthAction
}

// FrameHandler observes frame lifecycle (iframes, main frame swaps).
// Callbacks run on the browser process UI thread.
//
// The CefFrame arguments are valid only for the duration of the call. Retain
// nothing; to use a frame later, keep its identifier and look it up again.
type FrameHandler interface {
	OnFrameCreated(instance Browser, frame CefFrame)
	OnFrameDestroyed(instance Browser, frame CefFrame)
	OnFrameAttached(instance Browser, frame CefFrame, reattached bool)
	OnFrameDetached(instance Browser, frame CefFrame)
	OnMainFrameChanged(instance Browser, oldFrame, newFrame CefFrame)
}

// FileDialogCallback finishes a dialog taken over by a FileDialogHandler.
type FileDialogCallback = cef.FileDialogCallback

// V8Listener is told about every V8 context of the render process - register
// it with WithV8Listener. Both methods run synchronously on the renderer main
// thread, before any page script of a new context; browser and frame are
// borrowed for the call. Do not block in them.
type V8Listener interface {
	V8ContextCreated(CefBrowser, CefFrame)
	V8ContextReleased(CefBrowser, CefFrame)
}

// PopupHandler decides what happens when a page requests a popup. Return true
// to suppress the popup window - typically after routing request.TargetUrl
// somewhere else (same browser, new tab, OS browser). Return false to let CEF
// create the popup. Runs synchronously on the browser process UI thread, so
// it must not block; hand long work to a goroutine and return.
type PopupHandler func(instance Browser, request PopupRequest) bool

// CertificateErrorHandler decides per request whether a load with an
// invalid TLS certificate proceeds. certError is a net error code (e.g.
// -202 ERR_CERT_AUTHORITY_INVALID). Runs synchronously on the browser
// process UI thread - decide quickly or return CertificateDecideAsync
// and finish via the callback. Never called when the app runs with
// WithIgnoreCertificateErrors(true) - CEF accepts everything then.
type CertificateErrorHandler func(instance Browser, certError int, requestUrl string, callback CertificateErrorCallback) CertificateAction

type WorkerCallback func(delay int, worker Worker)

// BindingGuard decides whether a binding call may run. It is set per browser
// with WithBindingGuard, or for all browsers without one with
// WithDefaultBindingGuard. false rejects the call.
type BindingGuard func(call BindingCall) bool

// PopupViewFactory builds the PopupView for a popup (see WithPopupView).
// Returning false keeps CEF's native popup window.
type PopupViewFactory func(opener Browser, request PopupRequest) (PopupView, bool)

// Compile-time proof that each declaration above is exactly its internal/cef
// counterpart. Each pair asserts both inclusions, which together mean
// equality: a method added to or removed from the binding fails to build
// here, naming the method, instead of silently changing what callers must
// implement.
//
// These are compile-time only - no init work, nothing in the binary.
var (
	_ cef.LoadHandler             = (LoadHandler)(nil)
	_ LoadHandler                 = (cef.LoadHandler)(nil)
	_ cef.DisplayHandler          = (DisplayHandler)(nil)
	_ DisplayHandler              = (cef.DisplayHandler)(nil)
	_ cef.ConsoleMessageHandler   = (ConsoleMessageHandler)(nil)
	_ ConsoleMessageHandler       = (cef.ConsoleMessageHandler)(nil)
	_ cef.RequestHandler          = (RequestHandler)(nil)
	_ RequestHandler              = (cef.RequestHandler)(nil)
	_ cef.ResourceInterceptor     = (ResourceInterceptor)(nil)
	_ ResourceInterceptor         = (cef.ResourceInterceptor)(nil)
	_ cef.ContextMenuHandler      = (ContextMenuHandler)(nil)
	_ ContextMenuHandler          = (cef.ContextMenuHandler)(nil)
	_ cef.JsDialogHandler         = (JsDialogHandler)(nil)
	_ JsDialogHandler             = (cef.JsDialogHandler)(nil)
	_ cef.DownloadHandler         = (DownloadHandler)(nil)
	_ DownloadHandler             = (cef.DownloadHandler)(nil)
	_ cef.DragHandler             = (DragHandler)(nil)
	_ DragHandler                 = (cef.DragHandler)(nil)
	_ cef.FindHandler             = (FindHandler)(nil)
	_ FindHandler                 = (cef.FindHandler)(nil)
	_ cef.LifeSpanHandler         = (LifeSpanHandler)(nil)
	_ LifeSpanHandler             = (cef.LifeSpanHandler)(nil)
	_ cef.RenderHandler           = (RenderHandler)(nil)
	_ RenderHandler               = (cef.RenderHandler)(nil)
	_ cef.ResizableRenderHandler  = (ResizableRenderHandler)(nil)
	_ ResizableRenderHandler      = (cef.ResizableRenderHandler)(nil)
	_ cef.PopupRenderHandler      = (PopupRenderHandler)(nil)
	_ PopupRenderHandler          = (cef.PopupRenderHandler)(nil)
	_ cef.V8Listener              = (V8Listener)(nil)
	_ V8Listener                  = (cef.V8Listener)(nil)
	_ cef.ScreenInfoRenderHandler = (ScreenInfoRenderHandler)(nil)
	_ ScreenInfoRenderHandler     = (cef.ScreenInfoRenderHandler)(nil)
	_ cef.FocusHandler            = (FocusHandler)(nil)
	_ FocusHandler                = (cef.FocusHandler)(nil)
	_ cef.KeyboardHandler         = (KeyboardHandler)(nil)
	_ KeyboardHandler             = (cef.KeyboardHandler)(nil)
	_ cef.FileDialogHandler       = (FileDialogHandler)(nil)
	_ FileDialogHandler           = (cef.FileDialogHandler)(nil)
	_ cef.AuthHandler             = (AuthHandler)(nil)
	_ AuthHandler                 = (cef.AuthHandler)(nil)
	_ cef.FrameHandler            = (FrameHandler)(nil)
	_ FrameHandler                = (cef.FrameHandler)(nil)

	// Func types are convertible rather than assignable; a conversion in each
	// direction proves the signatures are identical.
	_ = cef.PopupHandler(PopupHandler(nil))
	_ = PopupHandler(cef.PopupHandler(nil))
	_ = cef.CertificateErrorHandler(CertificateErrorHandler(nil))
	_ = CertificateErrorHandler(cef.CertificateErrorHandler(nil))
	_ = cef.WorkerCallback(WorkerCallback(nil))
	_ = WorkerCallback(cef.WorkerCallback(nil))
	_ = cef.BindingGuard(BindingGuard(nil))
	_ = BindingGuard(cef.BindingGuard(nil))
	_ = cef.PopupViewFactory(PopupViewFactory(nil))
	_ = PopupViewFactory(cef.PopupViewFactory(nil))
)
