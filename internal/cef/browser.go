package cef

/*
#include <string.h>
#include "browser_client_bridge.h"
#include "browser_bridge.h"
*/
import "C"
import (
	"errors"
	"fmt"
	"image"
	"io"
	"sync"
	"time"
	"unicode/utf16"
	"unsafe"
)

type (
	browser struct {
		sync.Mutex

		ptr *C.cef_browser_t

		windowInfo C.cef_window_info_t
		settings   settings
		identifier int

		// requestContext is only read during NewBrowser - the browser keeps
		// the context alive on the CEF side afterwards
		requestContext *requestContext

		// DevTools session and the observer registration that feeds it;
		// both set in OnAfterCreated, cleared in OnBeforeClose
		devTools             *devToolsSession
		devToolsRegistration *C.cef_registration_t
	}

	Browser interface {
		io.Closer

		Identifier() int

		Redraw() error
		Reload() error
		// ReloadIgnoreCache reloads the page and refetches every resource.
		// The cached copies are ignored, not deleted - use
		// ExecuteDevToolsMethod("Network.clearBrowserCache", nil) for that.
		ReloadIgnoreCache() error

		// ExecuteDevToolsMethod runs a method of the DevTools protocol
		// (https://chromedevtools.github.io/devtools-protocol/) against this
		// browser. Neither an open DevTools window nor a remote debugging
		// port is needed; the call goes through CEF itself.
		//
		// params may be nil for methods without parameters; supported value
		// types are string, bool and int. The returned number is the message
		// id CEF assigned - the result of the method arrives asynchronously
		// and is not reported back here.
		//
		// The call blocks until the CEF UI thread has accepted the message.
		ExecuteDevToolsMethod(method string, params map[string]any) (int, error)

		LoadString(data string, url string)
		LoadUrl(url string)

		GoBack()
		GoForward()
		CanGoBack() bool
		CanGoForward() bool

		// SetZoomLevel: 0 = default; jede Stufe entspricht Chromiums
		// Zoom-Faktor-Schritten (level = log(faktor)/log(1.2)).
		SetZoomLevel(level float64)
		// ZoomLevel reads the current zoom on the UI thread and blocks until
		// the value is available.
		ZoomLevel() (float64, error)

		// Find searches the page; results arrive via WithFindHandlerFunc.
		// findNext continues the running search (same text) at the next hit.
		Find(text string, forward, matchCase, findNext bool)
		StopFinding(clearSelection bool)

		SetAudioMuted(muted bool)
		// IsAudioMuted reads the state on the UI thread and blocks until the
		// value is available.
		IsAudioMuted() (bool, error)

		// PrintToPdf renders the current page into a PDF file and blocks
		// until the file is written.
		PrintToPdf(path string, options PdfPrintOptions) error

		FocusGained()
		FocusLost()

		// MouseMoved forwards a mouse move. Pass the currently pressed
		// mouse-button / modifier flags (e.g. KeyEventModifierLeftMouseButton
		// while dragging) so text selection and dragging work.
		MouseMoved(x, y int, mouseLeft bool, modifiers ...KeyEventModifier)
		// MouseClicked forwards a click. clickCount is 1 for single clicks,
		// 2/3 for double/triple clicks.
		MouseClicked(x, y int, button MouseButton, direction MouseButtonDirection, clickCount int, modifiers ...KeyEventModifier)
		// Scrolled forwards a mouse-wheel event. CEF expects deltas in device
		// units where a positive deltaY scrolls up (one wheel notch = 120).
		Scrolled(x, y int, deltaX, deltaY int, modifiers ...KeyEventModifier)
		KeyEvent(eventType KeyEventType, nativeKeyCode, windowsKeyCode int, character rune, isSystemKey bool, modifiers ...KeyEventModifier)
		ExecuteJavascript(source string, url string, line int)

		Cef() CefBrowser
	}
	BrowserNotInitialized struct {
	}

	BrowserOption func(*browser)

	settings struct {
		// The maximum rate in frames per second (fps) that CefRenderHandler::OnPaint
		// will be called for a windowless browser. The actual fps may be lower if
		// the browser cannot generate frames at the requested rate. The minimum
		// value is 1 and the maximum value is 60 (default 30). This value can also be
		// changed dynamically via CefBrowserHost::SetWindowlessFrameRate.
		windowlessFrameRate *int

		// Controls whether file URLs will have access to all URLs. Also configurable
		// using the "allow-universal-access-from-files" command-line switch.
		universalAccessFromFileUrls bool

		// Controls whether file URLs will have access to other file URLs. Also
		// configurable using the "allow-access-from-files" command-line switch.
		fileAccessFromFileUrls bool

		// Controls whether web security restrictions (same-origin policy) will be
		// enforced. Disabling this setting is not recommend as it will allow risky
		// security behavior such as cross-site scripting (XSS). Also configurable
		// using the "disable-web-security" command-line switch.
		webSecurity bool

		// Controls whether WebGL can be used. Note that WebGL requires hardware
		// support and may not work on all systems even when enabled. Also
		// configurable using the "disable-webgl" command-line switch.
		webgl bool

		// Background color of the browser in ARGB. For windowless rendering a
		// fully transparent value (0) makes CEF composite with alpha, which
		// most consumers do not expect - the default is opaque white.
		backgroundColor uint32

		initialUrl string

		// windowed asks CEF to create an operating-system window for this
		// browser instead of painting into a RenderHandler. Implemented on
		// Windows and macOS.
		windowed     bool
		windowTitle  string
		windowBounds image.Rectangle

		// parentView is an NSView the browser is created in instead of a
		// window of its own; see WithParentView (macOS only). It points
		// into AppKit-owned memory, never into the Go heap.
		parentView unsafe.Pointer
	}
)

// Cef returns a borrowed view on the underlying cef_browser_t. It is only
// valid while the browser is alive; do not retain it across the browser's
// close. Returns nil once the browser has been closed.
func (b *browser) Cef() CefBrowser {
	b.Lock()
	defer b.Unlock()

	if nil == b.ptr {
		return nil
	}
	return &cefBrowser{cStruct: b.ptr}
}

func (b *browser) ExecuteJavascript(source string, url string, line int) {
	b.Lock()
	defer b.Unlock()

	if nil == b.ptr {
		return
	}

	sourceString := cefString(source)
	urlString := cefString(url)
	defer cefStringFree(sourceString)
	defer cefStringFree(urlString)

	C.cBridgeBrowserExecuteJavascript(b.ptr, sourceString, urlString, C.int(line))
}

func (b *browser) Identifier() int {
	return b.identifier
}

// invalidate releases the reference held on the underlying cef_browser_t.
// Called from on_before_close when CEF destroys the browser.
func (b *browser) invalidate() {
	b.Lock()
	defer b.Unlock()

	if nil != b.ptr {
		releaseBase(unsafe.Pointer(b.ptr))
		b.ptr = nil
	}
}

func (bni *BrowserNotInitialized) Error() string {
	return "BrowserNotInitialized"
}

func WithWindowlessFrameRate(rate int) BrowserOption {
	localRate := rate
	return func(browser *browser) {
		browser.settings.windowlessFrameRate = &localRate
	}
}

func WithUniversalAccessFromFileUrls(fileAccessFromFileUrls bool) BrowserOption {
	return func(browser *browser) {
		browser.settings.universalAccessFromFileUrls = fileAccessFromFileUrls
	}
}

func WithFileAccessFromFileUrls(fileAccessFromFileUrls bool) BrowserOption {
	return func(browser *browser) {
		browser.settings.fileAccessFromFileUrls = fileAccessFromFileUrls
	}
}

func WithWebSecurity(webSecurity bool) BrowserOption {
	return func(browser *browser) {
		browser.settings.webSecurity = webSecurity
	}
}

func WithWebgl(webgl bool) BrowserOption {
	return func(browser *browser) {
		browser.settings.webgl = webgl
	}
}

func WithInitialUrl(url string) BrowserOption {
	return func(browser *browser) {
		browser.settings.initialUrl = url
	}
}

// WithWindow creates the browser in an operating-system window of its own
// instead of rendering offscreen. The window carries no browser interface -
// no address bar, no tabs -, only the page.
//
// It is mutually exclusive with WithRenderHandler on the client: a client
// that has a render handler always renders offscreen, and the render handler
// wins. Implemented on Windows and macOS; elsewhere NewBrowser reports an
// error.
func WithWindow(title string, width, height int) BrowserOption {
	return WithWindowBounds(title, image.Rect(0, 0, width, height))
}

// WithWindowBounds is WithWindow with an explicit position. The rectangle is
// in screen coordinates; its origin is where the window's upper left corner
// lands.
func WithWindowBounds(title string, bounds image.Rectangle) BrowserOption {
	return func(browser *browser) {
		browser.settings.windowed = true
		browser.settings.windowTitle = title
		browser.settings.windowBounds = bounds.Canon()
	}
}

// WithRequestContext creates the browser inside the given browsing profile
// instead of the global context. See NewRequestContext.
func WithRequestContext(context RequestContext) BrowserOption {
	return func(browser *browser) {
		if impl, ok := context.(*requestContext); ok {
			browser.requestContext = impl
		}
	}
}

// WithBackgroundColor sets the browser background as ARGB, e.g. 0xFFFFFFFF
// for opaque white or 0 for a transparent surface.
func WithBackgroundColor(argb uint32) BrowserOption {
	return func(browser *browser) {
		browser.settings.backgroundColor = argb
	}
}

func (b *browser) GoBack() {
	b.Lock()
	defer b.Unlock()
	if b.ptr != nil {
		C.cBridgeBrowserGoBack(b.ptr)
	}
}

func (b *browser) GoForward() {
	b.Lock()
	defer b.Unlock()
	if b.ptr != nil {
		C.cBridgeBrowserGoForward(b.ptr)
	}
}

func (b *browser) CanGoBack() bool {
	b.Lock()
	defer b.Unlock()
	return b.ptr != nil && C.cBridgeBrowserCanGoBack(b.ptr) != 0
}

func (b *browser) CanGoForward() bool {
	b.Lock()
	defer b.Unlock()
	return b.ptr != nil && C.cBridgeBrowserCanGoForward(b.ptr) != 0
}

func (b *browser) SetZoomLevel(level float64) {
	b.Lock()
	defer b.Unlock()
	if b.ptr != nil {
		C.cBridgeBrowserSetZoomLevel(b.ptr, C.double(level))
	}
}

// retainPtr hands out the underlying browser pointer with an extra reference
// for use outside the browser mutex (UI-thread tasks). Release after use.
func (b *browser) retainPtr() *C.cef_browser_t {
	b.Lock()
	defer b.Unlock()
	if b.ptr == nil {
		return nil
	}
	addRefBase(unsafe.Pointer(b.ptr))
	return b.ptr
}

func (b *browser) ZoomLevel() (float64, error) {
	ptr := b.retainPtr()
	if ptr == nil {
		return 0, &BrowserNotInitialized{}
	}
	defer releaseBase(unsafe.Pointer(ptr))

	var level float64
	if !runOnUiThread(func() { level = float64(C.cBridgeBrowserGetZoomLevel(ptr)) }) {
		return 0, errors.New("could not read zoom level on the ui thread")
	}
	return level, nil
}

func (b *browser) Find(text string, forward, matchCase, findNext bool) {
	b.Lock()
	defer b.Unlock()
	if b.ptr == nil {
		return
	}
	cText := cefString(text)
	defer cefStringFree(cText)
	C.cBridgeBrowserFind(b.ptr, cText, cBool(forward), cBool(matchCase), cBool(findNext))
}

func (b *browser) StopFinding(clearSelection bool) {
	b.Lock()
	defer b.Unlock()
	if b.ptr != nil {
		C.cBridgeBrowserStopFinding(b.ptr, cBool(clearSelection))
	}
}

func (b *browser) SetAudioMuted(muted bool) {
	b.Lock()
	defer b.Unlock()
	if b.ptr != nil {
		C.cBridgeBrowserSetAudioMuted(b.ptr, cBool(muted))
	}
}

func (b *browser) IsAudioMuted() (bool, error) {
	ptr := b.retainPtr()
	if ptr == nil {
		return false, &BrowserNotInitialized{}
	}
	defer releaseBase(unsafe.Pointer(ptr))

	var muted bool
	if !runOnUiThread(func() { muted = C.cBridgeBrowserIsAudioMuted(ptr) != 0 }) {
		return false, errors.New("could not read audio state on the ui thread")
	}
	return muted, nil
}

func cBool(v bool) C.int {
	if v {
		return 1
	}
	return 0
}

// PdfPrintOptions keeps Chromium's defaults for everything not listed.
type PdfPrintOptions struct {
	Landscape       bool
	PrintBackground bool
}

var (
	pdfPendingLock sync.Mutex
	pdfPending     = make(map[unsafe.Pointer]chan bool)
)

//export goPdfPrintFinished
func goPdfPrintFinished(self unsafe.Pointer, ok C.int) {
	pdfPendingLock.Lock()
	ch, found := pdfPending[self]
	delete(pdfPending, self)
	pdfPendingLock.Unlock()

	if found {
		ch <- ok != 0
	}
}

// forgetPdfPending unblocks a waiter whose callback struct is destroyed
// without ever having been called. Called from goMemoryRelease; a no-op for
// unrelated pointers.
func forgetPdfPending(it unsafe.Pointer) {
	pdfPendingLock.Lock()
	ch, found := pdfPending[it]
	delete(pdfPending, it)
	pdfPendingLock.Unlock()

	if found {
		close(ch)
	}
}

func (b *browser) PrintToPdf(path string, options PdfPrintOptions) error {
	ptr := b.retainPtr()
	if ptr == nil {
		return &BrowserNotInitialized{}
	}
	defer releaseBase(unsafe.Pointer(ptr))

	callback := C.cPdfPrintCallbackCreate()
	ch := make(chan bool, 1)
	pdfPendingLock.Lock()
	pdfPending[unsafe.Pointer(callback)] = ch
	pdfPendingLock.Unlock()

	cPath := cefString(path)
	defer cefStringFree(cPath)

	// print_to_pdf consumes the callback reference
	C.cBridgeBrowserPrintToPdf(ptr, cPath, cBool(options.Landscape), cBool(options.PrintBackground), callback)

	// on_pdf_print_finished runs on the UI thread (see uiWaitPump)
	uw := beginUIWait()
	defer uw.stop()
	timeout := time.NewTimer(30 * time.Second)
	defer timeout.Stop()

	for {
		select {
		case ok, delivered := <-ch:
			if !delivered {
				return errors.New("pdf print aborted")
			}
			if !ok {
				return errors.New("pdf print failed")
			}
			return nil
		case <-uw.tick():
			uw.run()
		case <-timeout.C:
			pdfPendingLock.Lock()
			delete(pdfPending, unsafe.Pointer(callback))
			pdfPendingLock.Unlock()
			return errors.New("timed out waiting for the pdf print")
		}
	}
}

func (b *browser) Redraw() error {
	b.Lock()
	defer b.Unlock()

	if nil == b.ptr {
		return &BrowserNotInitialized{}
	}
	C.cBridgeBrowserWasResized(b.ptr)
	return nil

}

func (b *browser) Reload() error {
	b.Lock()
	defer b.Unlock()

	if nil == b.ptr {
		return &BrowserNotInitialized{}
	}
	C.cBridgeBrowserReload(b.ptr)
	return nil

}

func (b *browser) ReloadIgnoreCache() error {
	b.Lock()
	defer b.Unlock()

	if nil == b.ptr {
		return &BrowserNotInitialized{}
	}
	C.cBridgeBrowserReloadIgnoreCache(b.ptr)
	return nil

}

func (b *browser) ExecuteDevToolsMethod(method string, params map[string]any) (int, error) {
	ptr := b.retainPtr()
	if ptr == nil {
		return 0, &BrowserNotInitialized{}
	}
	defer releaseBase(unsafe.Pointer(ptr))

	cMethod := cefString(method)
	defer cefStringFree(cMethod)

	cParams, err := devToolsParams(params)
	if err != nil {
		return 0, err
	}

	var messageID C.int
	if !runOnUiThread(func() {
		messageID = C.cBridgeBrowserExecuteDevToolsMethod(ptr, cMethod, cParams)
	}) {
		// The bridge never ran, so nothing took the dictionary over.
		if cParams != nil {
			C.cBridgeDictionaryRelease(cParams)
		}
		return 0, fmt.Errorf("could not run the devtools method %q on the ui thread", method)
	}

	// execute_dev_tools_method answers with 0 for a message it refused to
	// submit - a malformed method name, or a call that did not reach the UI
	// thread after all.
	if messageID == 0 {
		return 0, fmt.Errorf("devtools method %q was not submitted", method)
	}
	return int(messageID), nil
}

// devToolsParams builds the parameter dictionary for a DevTools method.
//
// The result is handed to execute_dev_tools_method, which takes the reference
// over - the caller only has to release it when that call never happens.
//
// refcheck:fresh
func devToolsParams(params map[string]any) (*C.cef_dictionary_value_t, error) {
	if len(params) == 0 {
		return nil, nil
	}

	dictionary := C.cBridgeDictionaryCreate()
	if dictionary == nil {
		return nil, errors.New("could not create the devtools parameter dictionary")
	}

	for key, value := range params {
		cKey := cefString(key)
		switch typed := value.(type) {
		case string:
			cValue := cefString(typed)
			C.cBridgeDictionarySetString(dictionary, cKey, cValue)
			cefStringFree(cValue)
		case bool:
			C.cBridgeDictionarySetBool(dictionary, cKey, cBool(typed))
		case int:
			C.cBridgeDictionarySetInt(dictionary, cKey, C.int(typed))
		default:
			cefStringFree(cKey)
			C.cBridgeDictionaryRelease(dictionary)
			return nil, fmt.Errorf("unsupported devtools parameter type %T for %q", value, key)
		}
		cefStringFree(cKey)
	}
	return dictionary, nil
}

// LoadString loads the given HTML via a data: URL - CEF removed the frame's
// load_string. The url parameter can no longer be honoured as the document
// origin; it is ignored.
func (b *browser) LoadString(data string, url string) {
	b.Lock()
	defer b.Unlock()

	if nil == b.ptr {
		return
	}

	urlString := cefString(dataURLForHTML(data))
	defer cefStringFree(urlString)

	C.cBridgeBrowserLoadUrl(b.ptr, urlString)
}

func (b *browser) LoadUrl(url string) {
	b.Lock()
	defer b.Unlock()

	if nil == b.ptr {
		return
	}

	urlString := cefString(url)
	defer cefStringFree(urlString)

	C.cBridgeBrowserLoadUrl(b.ptr, urlString)
}
func (b *browser) FocusGained() {
	b.Lock()
	defer b.Unlock()

	if nil == b.ptr {
		return
	}
	C.cBridgeBrowserSendFocusEvent(b.ptr, C.int(1))
}
func (b *browser) FocusLost() {
	b.Lock()
	defer b.Unlock()

	if nil == b.ptr {
		return
	}
	C.cBridgeBrowserSendFocusEvent(b.ptr, C.int(0))
}

func (b *browser) MouseMoved(x, y int, mouseLeft bool, modifiers ...KeyEventModifier) {
	b.Lock()
	defer b.Unlock()

	if nil == b.ptr {
		return
	}

	mouseLeaveInt := C.int(0)
	if mouseLeft {
		mouseLeaveInt = C.int(1)
	}
	C.cBridgeBrowserSendMouseMoveEvent(b.ptr, C.int(x), C.int(y), mouseLeaveInt, C.uint32_t(KeyEventModifiers(modifiers).Sum()))
}

func (b *browser) Scrolled(x, y int, deltaX, deltaY int, modifiers ...KeyEventModifier) {
	b.Lock()
	defer b.Unlock()

	if nil == b.ptr {
		return
	}

	C.cBridgeBrowserSendMouseWheelEvent(b.ptr, C.int(x), C.int(y), C.int(deltaX), C.int(deltaY), C.uint32_t(KeyEventModifiers(modifiers).Sum()))
}

func (b *browser) KeyEvent(
	eventType KeyEventType,
	nativeKeyCode, windowsKeyCode int,
	character rune,
	isSystemKey bool,
	modifiers ...KeyEventModifier,
) {

	b.Lock()
	defer b.Unlock()

	if nil == b.ptr {
		return
	}

	isk := C.int(0)
	if isSystemKey {
		isk = C.int(1)
	}
	mods := C.uint32_t(KeyEventModifiers(modifiers).Sum())

	if eventType == KeyEventTypeChar && character > 0xFFFF {
		// characters outside the BMP must be sent as a UTF-16 surrogate pair
		high, low := utf16.EncodeRune(character)
		C.cBridgeBrowserSendKeyEvent(b.ptr, C.cef_key_event_type_t(eventType), C.int(nativeKeyCode), C.int(windowsKeyCode), C.char16_t(uint16(high)), isk, mods)
		C.cBridgeBrowserSendKeyEvent(b.ptr, C.cef_key_event_type_t(eventType), C.int(nativeKeyCode), C.int(windowsKeyCode), C.char16_t(uint16(low)), isk, mods)
		return
	}

	C.cBridgeBrowserSendKeyEvent(
		b.ptr,
		C.cef_key_event_type_t(eventType),
		C.int(nativeKeyCode),
		C.int(windowsKeyCode),
		C.char16_t(uint16(character)),
		isk,
		mods,
	)
}

func (b *browser) MouseClicked(x, y int, button MouseButton, direction MouseButtonDirection, clickCount int, modifiers ...KeyEventModifier) {

	b.Lock()
	defer b.Unlock()

	if nil == b.ptr {
		return
	}
	if clickCount < 1 {
		clickCount = 1
	}
	C.cBridgeBrowserSendMouseClickEvent(b.ptr, C.int(x), C.int(y), C.int(button), C.int(direction), C.int(clickCount), C.uint32_t(KeyEventModifiers(modifiers).Sum()))
}

// Close asks CEF to close the browser. Once CEF destroyed the browser,
// on_before_close releases the underlying reference and this wrapper becomes
// inert.
//
// The close is forced: windowless browsers have no OS window, so the
// unforced close path (do_close returning false -> OS close notification)
// would stall forever.
//
// The mutex must NOT be held across the close_browser call: with a forced
// close CEF may run the whole teardown synchronously on the calling (UI)
// thread, and the resulting on_before_close callback needs the same mutex.
func (b *browser) Close() error {
	// A reference of our own, not a plain read: on_before_close (a
	// window.close() from the page, on CEF's thread) can drop the browser's
	// last reference while this call is still on its way into get_host.
	ptr := b.retainPtr()
	if nil == ptr {
		return nil
	}
	defer releaseBase(unsafe.Pointer(ptr))
	C.cBridgeBrowserCloseBrowser(ptr, C.int(1))
	return nil
}
