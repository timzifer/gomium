package cef

/*
#include "lifespanhandler_bridge.h"
#include "browser_client_bridge.h"
*/
import "C"
import "unsafe"

type LifeSpanHandler interface {
	OnAfterCreated()
	DoClose() bool
	OnBeforeClose()
}

// WindowOpenDisposition describes where a popup wants to open.
type WindowOpenDisposition int

const (
	WodUnknown             = WindowOpenDisposition(C.CEF_WOD_UNKNOWN)
	WodCurrentTab          = WindowOpenDisposition(C.CEF_WOD_CURRENT_TAB)
	WodSingletonTab        = WindowOpenDisposition(C.CEF_WOD_SINGLETON_TAB)
	WodNewForegroundTab    = WindowOpenDisposition(C.CEF_WOD_NEW_FOREGROUND_TAB)
	WodNewBackgroundTab    = WindowOpenDisposition(C.CEF_WOD_NEW_BACKGROUND_TAB)
	WodNewPopup            = WindowOpenDisposition(C.CEF_WOD_NEW_POPUP)
	WodNewWindow           = WindowOpenDisposition(C.CEF_WOD_NEW_WINDOW)
	WodSaveToDisk          = WindowOpenDisposition(C.CEF_WOD_SAVE_TO_DISK)
	WodOffTheRecord        = WindowOpenDisposition(C.CEF_WOD_OFF_THE_RECORD)
	WodIgnoreAction        = WindowOpenDisposition(C.CEF_WOD_IGNORE_ACTION)
	WodSwitchToTab         = WindowOpenDisposition(C.CEF_WOD_SWITCH_TO_TAB)
	WodNewPictureInPicture = WindowOpenDisposition(C.CEF_WOD_NEW_PICTURE_IN_PICTURE)
)

// PopupRequest carries the information about a popup the page wants to open
// (window.open, target="_blank", shift-click, ...).
type PopupRequest struct {
	TargetUrl         string
	TargetFrameName   string
	TargetDisposition WindowOpenDisposition
	UserGesture       bool
}

// PopupHandler decides what happens when a page requests a popup. Return true
// to suppress the popup window - typically after routing request.TargetUrl
// somewhere else (same browser, new tab, OS browser). Return false to let CEF
// create the popup. Runs synchronously on the browser process UI thread, so
// it must not block; hand long work to a goroutine and return.
type PopupHandler func(instance Browser, request PopupRequest) bool

// PopupView is the offscreen view of a popup that PopupHandler let through.
// The popup gets a client of its own, so its paints, binding guard and
// handlers stay apart from the opener's.
type PopupView struct {
	// Options configure the popup's client. They must include a render
	// handler; without one the popup falls back to a native window.
	Options []ClientOption
	// Attach receives the popup browser once CEF has created it. It runs on
	// the UI thread and must not block.
	Attach func(Browser)
}

// PopupViewFactory builds the view for a popup. Returning false keeps CEF's
// native popup window. Runs on the UI thread and must not block.
type PopupViewFactory func(opener Browser, request PopupRequest) (PopupView, bool)

func lifeSpanHandler(handler *C.cef_life_span_handler_t) (*client, LifeSpanHandler) {
	app := getInstance()
	if app == nil {
		return nil, nil
	}

	app.clientMutex.Lock()
	defer app.clientMutex.Unlock()

	for _, c := range app.clients {
		c.Lock()
		match := c.lifeSpanHandlerPtr == handler
		h := c.lifeSpanHandler
		c.Unlock()
		if match {
			return c, h
		}
	}

	return nil, nil
}

//export goGetLifeSpanHandler
func goGetLifeSpanHandler(ptr *C.cef_client_t) (result *C.cef_life_span_handler_t) {

	c := getClient(ptr)
	if c == nil {
		return nil
	}

	c.Lock()
	defer c.Unlock()

	if c.lifeSpanHandlerPtr == nil {
		return nil
	}
	addRefBase(unsafe.Pointer(c.lifeSpanHandlerPtr))
	return c.lifeSpanHandlerPtr
}

//export goOnAfterCreated
func goOnAfterCreated(self *C.cef_life_span_handler_t, cb *C.cef_browser_t) {
	c, handler := lifeSpanHandler(self)
	if c == nil {
		// nobody to hand the browser to - drop the reference we received
		releaseBase(unsafe.Pointer(cb))
		return
	}

	// The reference carried by the callback parameter is retained: it either
	// travels to NewBrowser through the pending channel or is stored on a
	// standalone browser wrapper (popups, timed-out creations).
	identifier := int(C.cBridgeGetBrowserIdentifier(cb))

	c.Lock()
	pending := c.pendingBrowser
	c.pendingBrowser = nil
	if pending != nil && c.pendingNew != nil {
		// Register the wrapper NewBrowser is waiting with now: the renderer
		// may send its first extension call before NewBrowser wakes up, and
		// the guard must find this browser's settings then. NewBrowser sets
		// the same values again; it is blocked on pending until the send
		// below, so nothing races on the fields.
		c.pendingNew.ptr = cb
		c.pendingNew.identifier = identifier
		c.browsers[identifier] = c.pendingNew
	}
	c.pendingNew = nil
	c.Unlock()

	if pending != nil {
		// capacity 1 and only one producer - this never blocks
		pending <- cb
	} else {
		b := &browser{ptr: cb, identifier: identifier}
		c.Lock()
		c.browsers[identifier] = b
		attach := c.popupAttach
		c.Unlock()
		if attach != nil {
			attach(b)
		}
	}

	// the wrapper is registered now; the DevTools observer needs it to resolve
	// the browser in its callbacks
	c.Lock()
	observer := c.devToolsObserverPtr
	registered := c.browsers[identifier]
	c.Unlock()
	if observer != nil && registered != nil {
		attachDevTools(registered, cb, observer)
	}

	if handler != nil {
		handler.OnAfterCreated()
	}
}

//export goOnBeforePopup
func goOnBeforePopup(self *C.cef_life_span_handler_t,
	cefBrowser *C.cef_browser_t,
	frame *C.cef_frame_t,
	targetUrl *C.cef_string_t,
	targetFrameName *C.cef_string_t,
	targetDisposition C.cef_window_open_disposition_t,
	userGesture C.int,
	windowInfo *C.cef_window_info_t,
	popupClient **C.cef_client_t) C.int {
	defer releaseBase(unsafe.Pointer(cefBrowser))
	defer releaseBase(unsafe.Pointer(frame))

	c, _ := lifeSpanHandler(self)
	if c == nil {
		return 0
	}

	c.Lock()
	handler := c.popupHandler
	view := c.popupView
	c.Unlock()
	if handler == nil && view == nil {
		return 0
	}

	browserInstance := c.browserByIdentifier(int(C.cBridgeGetBrowserIdentifier(cefBrowser)))
	request := PopupRequest{
		TargetUrl:         cefToGoString(targetUrl),
		TargetFrameName:   cefToGoString(targetFrameName),
		TargetDisposition: WindowOpenDisposition(targetDisposition),
		UserGesture:       userGesture != 0,
	}

	if handler != nil && handler(browserInstance, request) {
		return 1
	}
	if view == nil {
		return 0
	}
	popup, ok := view(browserInstance, request)
	if !ok || !openPopupView(popup, windowInfo, popupClient) {
		return 0
	}
	return 0
}

// openPopupView creates the client of an offscreen popup and tells CEF to
// build the popup windowless with that client. Returns false when the view
// cannot render - CEF then opens its native window.
func openPopupView(view PopupView, windowInfo *C.cef_window_info_t, popupClient **C.cef_client_t) bool {
	app := getInstance()
	if app == nil {
		return false
	}

	created, ok := app.NewClient(view.Options...).(*client)
	if !ok {
		return false
	}

	created.Lock()
	renders := created.renderHandler != nil
	created.popupAttach = view.Attach
	created.popupOwned = true
	created.Unlock()
	if !renders {
		_ = created.Close()
		return false
	}

	// CEF releases the reference it receives here; the client wrapper keeps its own
	addRefBase(unsafe.Pointer(created.ptr))
	*popupClient = created.ptr

	windowInfo.size = C.sizeof_cef_window_info_t
	windowInfo.windowless_rendering_enabled = 1
	// windowless rendering requires the Alloy runtime style, as for NewBrowser
	windowInfo.runtime_style = C.CEF_RUNTIME_STYLE_ALLOY
	return true
}

//export goDoClose
func goDoClose(self *C.cef_life_span_handler_t, cb *C.cef_browser_t) C.int {
	defer releaseBase(unsafe.Pointer(cb))

	result := false
	if _, handler := lifeSpanHandler(self); handler != nil {
		result = handler.DoClose()
	}

	if result {
		return C.int(1)
	}
	return C.int(0)
}

//export goOnBeforeClose
func goOnBeforeClose(self *C.cef_life_span_handler_t, cb *C.cef_browser_t) {
	defer releaseBase(unsafe.Pointer(cb))

	c, handler := lifeSpanHandler(self)
	if c != nil {
		identifier := int(C.cBridgeGetBrowserIdentifier(cb))

		c.Lock()
		b, exists := c.browsers[identifier]
		delete(c.browsers, identifier)
		// a popup client lives exactly as long as its popup browser
		closePopup := c.popupOwned && len(c.browsers) == 0
		c.Unlock()

		if exists && b != nil {
			b.detachDevTools()
			b.invalidate()
		}
		if closePopup {
			c.logger.Debug().Msg("closing popup client")
			go func() { _ = c.Close() }()
		}
	}

	if handler != nil {
		handler.OnBeforeClose()
	}
}
