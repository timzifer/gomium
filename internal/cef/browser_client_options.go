package cef

import (
	"net/url"
	"path/filepath"
	"strings"
	"time"
)

type (
	ClientOption func(*client)

	lifeSpanHandlerFunc struct {
		onAfterCreated func()
		doClose        func() bool
		onBeforeClose  func()
	}
	displayHandlerFunc struct {
		onTitleChange      func(Browser, string)
		onAddressChange    func(Browser, *url.URL)
		onConsoleMessage   func(Browser, string, int, string)
		onFaviconUrlChange func(Browser, []*url.URL)
	}
)

// The wrappers below all tolerate a nil function. Whoever wants one of four
// events has to pass three nils, and a constructor that takes them and then
// crashes on the first callback is a trap - the crash arrives on CEF's UI
// thread, far away from the call that set it.

func (d displayHandlerFunc) OnTitleChange(bi Browser, title string) {
	if d.onTitleChange != nil {
		d.onTitleChange(bi, title)
	}
}

func (d displayHandlerFunc) OnAddressChange(bi Browser, theUrl *url.URL) {
	if d.onAddressChange != nil {
		d.onAddressChange(bi, theUrl)
	}
}

func (d displayHandlerFunc) OnFaviconUrlChange(bi Browser, urls []*url.URL) {
	if d.onFaviconUrlChange != nil {
		d.onFaviconUrlChange(bi, urls)
	}
}

func (d displayHandlerFunc) OnConsoleMessage(bi Browser, source string, line int, message string) {
	if d.onConsoleMessage != nil {
		d.onConsoleMessage(bi, source, line, message)
	}
}

func (l lifeSpanHandlerFunc) OnAfterCreated() {
	if l.onAfterCreated != nil {
		l.onAfterCreated()
	}
}

// DoClose without a function returns false - the CEF default, which lets the
// browser be destroyed. Refusing the close would be the more surprising
// answer to "no opinion".
func (l lifeSpanHandlerFunc) DoClose() bool {
	if l.doClose == nil {
		return false
	}
	return l.doClose()
}

func (l lifeSpanHandlerFunc) OnBeforeClose() {
	if l.onBeforeClose != nil {
		l.onBeforeClose()
	}
}

// WithLifeSpanHandlerFuncs takes the three life-span events; every one of
// them may be nil.
func WithLifeSpanHandlerFuncs(onAfterCreated, onBeforeClose func(), doClose func() bool) ClientOption {
	return func(o *client) {
		o.lifeSpanHandler = &lifeSpanHandlerFunc{
			onAfterCreated: onAfterCreated,
			doClose:        doClose,
			onBeforeClose:  onBeforeClose,
		}
	}
}

func WithDisplayHandlerFuncs(onTitleChange func(Browser, string), onAddressChange func(Browser, *url.URL), onConsoleMessage func(Browser, string, int, string), onFaviconUrlChange func(Browser, []*url.URL)) ClientOption {
	return func(o *client) {
		o.displayHandler = &displayHandlerFunc{
			onTitleChange:      onTitleChange,
			onAddressChange:    onAddressChange,
			onConsoleMessage:   onConsoleMessage,
			onFaviconUrlChange: onFaviconUrlChange,
		}
	}
}

func WithBrowserCreationTimeout(timeout time.Duration) ClientOption {
	return func(o *client) {
		o.createTimeout = timeout
	}
}

func WithDefaultInitialUrl(initialUrl string) ClientOption {
	return func(o *client) {
		o.defaultInitialUrl = initialUrl
	}
}

func WithRequestHandler(handler RequestHandler) ClientOption {
	return func(c *client) {
		c.requestHandler = handler
	}
}

// WithCertificateErrorHandler decides per request whether an invalid TLS
// certificate is accepted - the surgical alternative to the global
// WithIgnoreCertificateErrors app option.
func WithCertificateErrorHandler(handler CertificateErrorHandler) ClientOption {
	return func(c *client) {
		c.certificateErrorHandler = handler
	}
}

// WithCertificateErrorsAcceptedFor accepts invalid certificates for exactly
// the given hosts (case-insensitive, an optional port on the host entry is
// matched too) and denies everything else.
func WithCertificateErrorsAcceptedFor(hosts ...string) ClientOption {
	allowed := make(map[string]bool, len(hosts))
	for _, h := range hosts {
		allowed[strings.ToLower(h)] = true
	}
	return WithCertificateErrorHandler(func(_ Browser, _ int, requestUrl string, _ CertificateErrorCallback) CertificateAction {
		parsed, err := url.Parse(requestUrl)
		if err != nil {
			return CertificateDeny
		}
		if allowed[strings.ToLower(parsed.Host)] || allowed[strings.ToLower(parsed.Hostname())] {
			return CertificateAccept
		}
		return CertificateDeny
	})
}

func WithDisplayHandler(handler DisplayHandler) ClientOption {
	return func(c *client) {
		c.displayHandler = handler
	}
}

func WithRenderHandler(handler RenderHandler) ClientOption {
	return func(c *client) {
		c.renderHandler = handler
	}
}

func WithLifeSpanHandler(handler LifeSpanHandler) ClientOption {
	return func(c *client) {
		c.lifeSpanHandler = handler
	}
}

func WithPopupHandler(handler PopupHandler) ClientOption {
	return func(c *client) {
		c.popupHandler = handler
	}
}

// WithPopupView keeps popups offscreen: every popup that PopupHandler lets
// through becomes a windowless browser with a client of its own, built from
// the view the factory returns. Without this option CEF opens a native popup
// window. Popups do not share the opener's render handler or binding guard.
func WithPopupView(factory PopupViewFactory) ClientOption {
	return func(c *client) {
		c.popupView = factory
	}
}

// WithPopupsInOpener suppresses every popup window and instead navigates the
// opener browser to the popup's target URL. The pragmatic default for
// single-view embeds where target="_blank" links should not vanish.
func WithPopupsInOpener() ClientOption {
	return WithPopupHandler(func(instance Browser, request PopupRequest) bool {
		if instance != nil && request.TargetUrl != "" {
			instance.LoadUrl(request.TargetUrl)
		}
		return true
	})
}

func WithLoadHandler(handler LoadHandler) ClientOption {
	return func(c *client) {
		c.loadHandler = handler
	}
}

type loadHandlerFunc struct {
	onLoadingStateChange func(Browser, bool, bool, bool)
	onLoadStart          func(Browser, bool)
	onLoadEnd            func(Browser, bool, int)
	onLoadError          func(Browser, bool, int, string, string)
}

func (l loadHandlerFunc) OnLoadingStateChange(bi Browser, isLoading, canGoBack, canGoForward bool) {
	if l.onLoadingStateChange != nil {
		l.onLoadingStateChange(bi, isLoading, canGoBack, canGoForward)
	}
}

func (l loadHandlerFunc) OnLoadStart(bi Browser, isMainFrame bool) {
	if l.onLoadStart != nil {
		l.onLoadStart(bi, isMainFrame)
	}
}

func (l loadHandlerFunc) OnLoadEnd(bi Browser, isMainFrame bool, httpStatusCode int) {
	if l.onLoadEnd != nil {
		l.onLoadEnd(bi, isMainFrame, httpStatusCode)
	}
}

func (l loadHandlerFunc) OnLoadError(bi Browser, isMainFrame bool, errorCode int, errorText string, failedUrl string) {
	if l.onLoadError != nil {
		l.onLoadError(bi, isMainFrame, errorCode, errorText, failedUrl)
	}
}

// WithLoadHandlerFuncs registers load callbacks without implementing the full
// LoadHandler interface. Individual callbacks may be nil.
func WithLoadHandlerFuncs(
	onLoadingStateChange func(instance Browser, isLoading, canGoBack, canGoForward bool),
	onLoadStart func(instance Browser, isMainFrame bool),
	onLoadEnd func(instance Browser, isMainFrame bool, httpStatusCode int),
	onLoadError func(instance Browser, isMainFrame bool, errorCode int, errorText, failedUrl string),
) ClientOption {
	return func(c *client) {
		c.loadHandler = &loadHandlerFunc{
			onLoadingStateChange: onLoadingStateChange,
			onLoadStart:          onLoadStart,
			onLoadEnd:            onLoadEnd,
			onLoadError:          onLoadError,
		}
	}
}

func WithJsDialogHandler(handler JsDialogHandler) ClientOption {
	return func(c *client) {
		c.jsDialogHandler = handler
	}
}

type suppressJsDialogs struct{}

func (suppressJsDialogs) OnJsDialog(Browser, string, JsDialogType, string, string, JsDialogCallback) JsDialogAction {
	return JsDialogSuppressed
}

func (suppressJsDialogs) OnBeforeUnloadDialog(Browser, string, bool, JsDialogCallback) JsDialogAction {
	return JsDialogSuppressed
}

// WithJsDialogsSuppressed drops every alert/confirm/prompt silently (confirm
// and prompt resolve as canceled) and leaves pages without asking on
// onbeforeunload. The right default for embeds that never want browser
// dialogs to stall the page.
func WithJsDialogsSuppressed() ClientOption {
	return WithJsDialogHandler(suppressJsDialogs{})
}

func WithContextMenuHandler(handler ContextMenuHandler) ClientOption {
	return func(c *client) {
		c.contextMenuHandler = handler
	}
}

type suppressContextMenu struct{}

func (suppressContextMenu) RunContextMenu(_ Browser, _ ContextMenuParams, _ []ContextMenuItem, callback ContextMenuCallback) bool {
	callback.Cancel()
	return true
}

// WithContextMenuSuppressed swallows every right-click menu - in windowless
// rendering the default menu would appear as a detached native window.
func WithContextMenuSuppressed() ClientOption {
	return WithContextMenuHandler(suppressContextMenu{})
}

func WithFindHandler(handler FindHandler) ClientOption {
	return func(c *client) {
		c.findHandler = handler
	}
}

// WithFindHandlerFunc receives the results of Browser.Find searches.
func WithFindHandlerFunc(fn func(instance Browser, result FindResult)) ClientOption {
	return WithFindHandler(findHandlerFunc(fn))
}

// WithFocusHandler observes browser focus (Tab out, focus gain). Optional.
func WithFocusHandler(handler FocusHandler) ClientOption {
	return func(c *client) {
		c.focusHandler = handler
	}
}

// WithFileDialogHandler takes over file pickers (<input type=file>, save
// dialogs). Optional; without it CEF shows its native dialog.
func WithFileDialogHandler(handler FileDialogHandler) ClientOption {
	return func(c *client) {
		c.fileDialogHandler = handler
	}
}

// WithAuthHandler answers HTTP and proxy auth challenges. Every challenge is
// routed here; without a handler (or with AuthDefault) the request is cancelled.
func WithAuthHandler(handler AuthHandler) ClientOption {
	return func(c *client) {
		c.authHandler = handler
	}
}

// WithFrameHandler observes frame lifecycle (iframes, main frame swaps).
// Optional.
func WithFrameHandler(handler FrameHandler) ClientOption {
	return func(c *client) {
		c.frameHandler = handler
	}
}

// WithKeyboardHandler sees every key before and after the page. Optional;
// without it the page receives all keys unchanged.
func WithKeyboardHandler(handler KeyboardHandler) ClientOption {
	return func(c *client) {
		c.keyboardHandler = handler
	}
}

func WithDragHandler(handler DragHandler) ClientOption {
	return func(c *client) {
		c.dragHandler = handler
	}
}

// WithFileDropHandler reports the native paths of a file drag as soon as it
// enters the browser window and lets the drag continue. The drop itself stays
// an ordinary HTML5 drop event in the page - this option only supplies what
// the page cannot see, the paths behind the dropped files.
func WithFileDropHandler(fn func(instance Browser, files []string)) ClientOption {
	return WithDragHandler(dragHandlerFunc(func(bi Browser, files []string) bool {
		if len(files) > 0 {
			fn(bi, files)
		}
		return false
	}))
}

func WithDownloadHandler(handler DownloadHandler) ClientOption {
	return func(c *client) {
		c.downloadHandler = handler
	}
}

type directoryDownloadHandler struct {
	dir      string
	onUpdate func(Browser, DownloadItem)
}

func (d directoryDownloadHandler) CanDownload(Browser, string, string) bool { return true }

func (d directoryDownloadHandler) OnBeforeDownload(_ Browser, item DownloadItem) string {
	return filepath.Join(d.dir, item.SuggestedFileName)
}

func (d directoryDownloadHandler) OnDownloadUpdated(bi Browser, item DownloadItem, _ DownloadControl) {
	if d.onUpdate != nil {
		d.onUpdate(bi, item)
	}
}

// WithDownloadDirectory accepts every download and stores it in dir under the
// suggested file name. Existing files are overwritten by Chromium's usual
// uniquify rules. onUpdate may be nil; otherwise it receives every progress
// update including the final Complete/Canceled/Interrupted state.
func WithDownloadDirectory(dir string, onUpdate func(instance Browser, item DownloadItem)) ClientOption {
	return func(c *client) {
		c.downloadHandler = directoryDownloadHandler{dir: dir, onUpdate: onUpdate}
	}
}
