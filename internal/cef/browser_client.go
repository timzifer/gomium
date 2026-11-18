package cef

/*
#include <string.h>
#include "browser_client_bridge.h"
#include "renderhandler_bridge.h"
#include "requesthandler_bridge.h"
#include "lifespanhandler_bridge.h"
#include "displayhandler_bridge.h"
#include "loadhandler_bridge.h"
#include "devtools_bridge.h"
#include "downloadhandler_bridge.h"
#include "requestcontext_bridge.h"
#include "jsdialoghandler_bridge.h"
#include "focushandler_bridge.h"
#include "keyboardhandler_bridge.h"
#include "filedialoghandler_bridge.h"
#include "framehandler_bridge.h"
#include "contextmenuhandler_bridge.h"
#include "findhandler_bridge.h"
#include "draghandler_bridge.h"

*/
import "C"
import (
	"errors"
	"github.com/rs/zerolog"
	"io"
	"sync"
	"time"
	"unsafe"
)

type (
	client struct {
		sync.Mutex

		ptr    *C.cef_client_t
		logger zerolog.Logger

		renderHandlerPtr *C.cef_render_handler_t
		renderHandler    RenderHandler

		requestHandlerPtr *C.cef_request_handler_t
		requestHandler    RequestHandler

		certificateErrorHandler CertificateErrorHandler
		authHandler             AuthHandler

		// bindingGuard decides which binding calls of this client's frames run.
		bindingGuard BindingGuard

		// resource callbacks moved to their own struct in CEF 151 - it is
		// returned from cef_request_handler_t.get_resource_request_handler
		resourceRequestHandlerPtr *C.cef_resource_request_handler_t

		lifeSpanHandlerPtr *C.cef_life_span_handler_t
		lifeSpanHandler    LifeSpanHandler
		popupHandler       PopupHandler

		// DevTools: one observer per client, registered on every browser
		devToolsReceiver    func(instance Browser, dt DevTools)
		devToolsObserverPtr *C.cef_dev_tools_message_observer_t

		displayHandlerPtr *C.cef_display_handler_t
		displayHandler    DisplayHandler

		loadHandlerPtr *C.cef_load_handler_t
		loadHandler    LoadHandler

		jsDialogHandlerPtr *C.cef_jsdialog_handler_t
		jsDialogHandler    JsDialogHandler

		contextMenuHandlerPtr *C.cef_context_menu_handler_t
		contextMenuHandler    ContextMenuHandler

		findHandlerPtr *C.cef_find_handler_t
		findHandler    FindHandler

		dragHandlerPtr *C.cef_drag_handler_t
		dragHandler    DragHandler

		focusHandlerPtr *C.cef_focus_handler_t
		focusHandler    FocusHandler

		keyboardHandlerPtr *C.cef_keyboard_handler_t
		keyboardHandler    KeyboardHandler

		fileDialogHandlerPtr *C.cef_dialog_handler_t
		fileDialogHandler    FileDialogHandler

		frameHandlerPtr *C.cef_frame_handler_t
		frameHandler    FrameHandler

		downloadHandlerPtr *C.cef_download_handler_t
		downloadHandler    DownloadHandler

		// pendingBrowser is set by NewBrowser while it waits for the
		// browser created by cef_browser_host_create_browser to arrive in
		// goOnAfterCreated. Guarded by the client mutex.
		pendingBrowser chan *C.cef_browser_t
		// pendingNew is the wrapper NewBrowser fills for pendingBrowser.
		// goOnAfterCreated registers it right away, so that handlers - and
		// the extension call guard - find the browser with its settings
		// before NewBrowser has woken up. Guarded by the client mutex.
		pendingNew *browser

		// popupView builds the offscreen view of popups (WithPopupView).
		popupView PopupViewFactory
		// popupAttach receives the browser of a popup client; popupOwned marks
		// clients the popup path created. Guarded by the client mutex.
		popupAttach func(Browser)
		popupOwned  bool
		// pendingPopups holds the clients openPopupView created for popups
		// of this (opener) client that CEF has not built yet. An entry leaves
		// when the popup browser is created, when CEF aborts the popup
		// (the client is closed then), or when this client closes.
		// popupOpener/popupSlot point back from such a popup client to its
		// entry until the popup browser exists. Guarded by the client mutex.
		pendingPopups map[popupSlot]*client
		popupOpener   *client
		popupSlot     popupSlot

		createTimeout     time.Duration
		closeTimeout      time.Duration
		defaultInitialUrl string
		browsers          map[int]*browser
	}

	Client interface {
		io.Closer

		NewBrowser(opts ...BrowserOption) (Browser, error)
	}
)

//export goOnProcessMessageReceived
func goOnProcessMessageReceived(ptr *C.cef_client_t, cefBrowser *C.cef_browser_t, cFrame *C.cef_frame_t, sourceProcess C.cef_process_id_t, message *C.cef_process_message_t) C.int {
	defer releaseBase(unsafe.Pointer(cefBrowser))
	defer releaseBase(unsafe.Pointer(cFrame))
	defer releaseBase(unsafe.Pointer(message))

	c := getClient(ptr)
	if c == nil {
		return 0
	}

	msg, err := getProcessMessageInstance(processMessage{message})
	if err != nil {
		c.logger.Warn().Err(err).Msg("could not decode process message")
		return 0
	}

	switch payload := msg.(type) {
	case *javaScriptEvaluationResponse:
		resolveJavaScriptEvaluation(payload.Id, payload.Result, payload.Exception)
		return 1
	case *bindingCallRequest:
		handleBindingCallRequest(c, payload, cefFrame{cFrame})
		return 1
	default:
		c.logger.Debug().Any("source-process", sourceProcess).Msgf("unhandled process message type %T", payload)
		return 0
	}
}

func cefStateFromBool(state bool) C.cef_state_t {
	if state == true {
		return C.STATE_ENABLED
	} else {
		return C.STATE_DISABLED
	}
}

func (c *client) NewBrowser(opts ...BrowserOption) (Browser, error) {
	c.Lock()

	if c.ptr == nil {
		c.Unlock()
		return nil, errors.New("browser-client is already closed")
	}
	if c.pendingBrowser != nil {
		c.Unlock()
		return nil, errors.New("another browser creation is already in flight")
	}

	newBrowser := &browser{
		settings: settings{
			initialUrl:      c.defaultInitialUrl,
			backgroundColor: 0xFFFFFFFF,
		},
	}

	for _, opt := range opts {
		opt(newBrowser)
	}

	// cef_window_info_t carries a mandatory size field since CEF 126
	newBrowser.windowInfo.size = C.sizeof_cef_window_info_t

	if nil != c.renderHandler {
		c.logger.Debug().Msg("activating windowless-rendering...")
		newBrowser.windowInfo.windowless_rendering_enabled = 1
		// windowless rendering requires the Alloy runtime style - the Chrome
		// style (the 151 default) does not support OSR
		newBrowser.windowInfo.runtime_style = C.CEF_RUNTIME_STYLE_ALLOY
	} else if newBrowser.settings.windowed {
		c.logger.Debug().Msg("creating a windowed browser...")
		if err := newBrowser.applyWindowInfo(); err != nil {
			c.Unlock()
			return nil, err
		}
	}

	// universal_access_from_file_urls, file_access_from_file_urls and
	// web_security no longer exist in cef_browser_settings_t - the equivalent
	// command-line switches are the only remaining knobs
	browserSettings := C.cef_browser_settings_t{
		size:             C.sizeof_cef_browser_settings_t,
		webgl:            cefStateFromBool(newBrowser.settings.webgl),
		background_color: C.cef_color_t(newBrowser.settings.backgroundColor),
	}

	if nil != newBrowser.settings.windowlessFrameRate {
		browserSettings.windowless_frame_rate = C.int(*newBrowser.settings.windowlessFrameRate)
	}

	pending := make(chan *C.cef_browser_t, 1)
	c.pendingBrowser = pending
	c.pendingNew = newBrowser
	clientPtr := c.ptr
	c.Unlock()

	when := time.Now()
	c.logger.Debug().Msg("creating cef-browser")

	initialUrl := cefString(newBrowser.settings.initialUrl)
	defer cefStringFree(initialUrl)

	// cef_browser_host_create_browser consumes a reference on the client and,
	// when given, on the request context.
	addRefBase(unsafe.Pointer(clientPtr))
	var contextPtr *C.cef_request_context_t
	if newBrowser.requestContext != nil {
		contextPtr = newBrowser.requestContext.ptr
		addRefBase(unsafe.Pointer(contextPtr))
	}
	created := int(C.cef_browser_host_create_browser(&newBrowser.windowInfo, clientPtr, initialUrl, &browserSettings, nil, contextPtr))
	// CEF has copied the window info by now, so the window title can go back.
	newBrowser.releaseWindowInfo()

	if created == 0 {
		c.Lock()
		c.pendingBrowser = nil
		c.pendingNew = nil
		c.Unlock()
		return nil, errors.New("browser creation failed")
	}
	c.logger.Debug().Msg("browser creation successful, waiting for on_create-event")

	// on_after_created runs on the UI thread; with an external pump and the
	// caller on that thread, the wait has to pump (see uiWaitPump)
	uw := beginUIWait()
	defer uw.stop()
	timeout := time.NewTimer(c.createTimeout)
	defer timeout.Stop()

	var browserPtr *C.cef_browser_t
wait:
	for {
		select {
		case browserPtr = <-pending:
			break wait
		case <-uw.tick():
			uw.run()
		case <-timeout.C:
			// detach the pending channel so goOnAfterCreated registers a
			// late-arriving browser on its own, then check once more whether it
			// arrived while we were detaching
			c.Lock()
			if c.pendingBrowser == pending {
				c.pendingBrowser = nil
				c.pendingNew = nil
			}
			c.Unlock()

			select {
			case browserPtr = <-pending:
				break wait
			default:
				err := errors.New("timed out while waiting for cef-browser")
				c.logger.Error().Dur("duration", time.Since(when)).Dur("timeout", c.createTimeout).Err(err).Send()
				return nil, err
			}
		}
	}

	c.Lock()
	if c.pendingBrowser == pending {
		c.pendingBrowser = nil
		c.pendingNew = nil
	}
	// goOnAfterCreated normally filled and registered the wrapper already
	// (see pendingNew); handlers may be using it by now, so it is only
	// written when that did not happen.
	if newBrowser.ptr == nil {
		newBrowser.ptr = browserPtr
		newBrowser.identifier = int(C.cBridgeGetBrowserIdentifier(browserPtr))
	}
	c.browsers[newBrowser.identifier] = newBrowser
	c.Unlock()

	c.logger.Debug().Dur("duration", time.Since(when)).Msg("browser received")
	return newBrowser, nil
}

func getClient(ptr *C.cef_client_t) *client {
	app := getInstance()
	if app == nil {
		return nil
	}

	app.clientMutex.Lock()
	defer app.clientMutex.Unlock()

	return app.clients[unsafe.Pointer(ptr)]
}

// browserByIdentifier resolves a browser registered on this client. Returns
// an untyped nil interface on a miss so callers can safely compare with nil.
func (c *client) browserByIdentifier(identifier int) Browser {
	c.Lock()
	defer c.Unlock()

	if b, exists := c.browsers[identifier]; exists && b != nil {
		return b
	}
	return nil
}

// Close implements io.Closer. With an external message pump and the caller on
// the thread that ran cef_initialize, Close pumps the message loop while it
// waits for the browsers to shut down (see uiWaitPump).
func (c *client) Close() error {
	return c.closeInternal(uiWaitPump())
}

func (c *client) closeInternal(pump func()) error {
	c.Lock()
	if c.ptr == nil {
		c.Unlock()
		return nil
	}
	browsers := make([]*browser, 0, len(c.browsers))
	for _, b := range c.browsers {
		browsers = append(browsers, b)
	}
	c.Unlock()

	c.logger.Trace().Msg("closing client")

	for _, b := range browsers {
		if closeErr := b.Close(); closeErr != nil {
			c.logger.Warn().Int("identifier", b.identifier).Err(closeErr).Msg("error closing browser")
		}
	}

	// wait until on_before_close removed all browsers; the message loop must
	// keep moving while we wait, so CEF can actually deliver those callbacks
	deadline := time.Now().Add(c.closeTimeout)
	for {
		c.Lock()
		remaining := len(c.browsers)
		c.Unlock()
		if remaining == 0 {
			break
		}
		if time.Now().After(deadline) {
			c.logger.Warn().Int("remaining", remaining).Msg("timed out waiting for browsers to close")
			break
		}
		if pump != nil {
			pump()
		}
		time.Sleep(10 * time.Millisecond)
	}

	c.Lock()
	if c.ptr == nil {
		c.Unlock()
		return nil
	}

	clientKey := unsafe.Pointer(c.ptr)

	c.browsers = make(map[int]*browser)
	// popups this client opened that CEF has not built yet die with it; a
	// popup client that is still pending leaves its opener's list
	pendingPopups := c.pendingPopups
	c.pendingPopups = nil
	opener, slot := c.popupOpener, c.popupSlot
	c.popupOpener = nil

	C.cClientFree(c.ptr)
	c.ptr = nil

	if c.renderHandlerPtr != nil {
		C.cRenderHandlerFree(c.renderHandlerPtr)
		c.renderHandlerPtr = nil
	}
	if c.requestHandlerPtr != nil {
		C.cRequestHandlerFree(c.requestHandlerPtr)
		c.requestHandlerPtr = nil
	}
	if c.resourceRequestHandlerPtr != nil {
		C.cResourceRequestHandlerFree(c.resourceRequestHandlerPtr)
		c.resourceRequestHandlerPtr = nil
	}
	if c.lifeSpanHandlerPtr != nil {
		C.cLifeSpanHandlerFree(c.lifeSpanHandlerPtr)
		c.lifeSpanHandlerPtr = nil
	}
	if c.displayHandlerPtr != nil {
		C.cDisplayHandlerFree(c.displayHandlerPtr)
		c.displayHandlerPtr = nil
	}
	if c.loadHandlerPtr != nil {
		C.cLoadHandlerFree(c.loadHandlerPtr)
		c.loadHandlerPtr = nil
	}
	if c.devToolsObserverPtr != nil {
		C.cDevToolsObserverFree(c.devToolsObserverPtr)
		c.devToolsObserverPtr = nil
	}
	if c.downloadHandlerPtr != nil {
		C.cDownloadHandlerFree(c.downloadHandlerPtr)
		c.downloadHandlerPtr = nil
	}
	if c.jsDialogHandlerPtr != nil {
		C.cJsDialogHandlerFree(c.jsDialogHandlerPtr)
		c.jsDialogHandlerPtr = nil
	}
	if c.contextMenuHandlerPtr != nil {
		C.cContextMenuHandlerFree(c.contextMenuHandlerPtr)
		c.contextMenuHandlerPtr = nil
	}
	if c.findHandlerPtr != nil {
		C.cFindHandlerFree(c.findHandlerPtr)
		c.findHandlerPtr = nil
	}
	if c.dragHandlerPtr != nil {
		C.cDragHandlerFree(c.dragHandlerPtr)
		c.dragHandlerPtr = nil
	}
	if c.focusHandlerPtr != nil {
		C.cFocusHandlerFree(c.focusHandlerPtr)
		c.focusHandlerPtr = nil
	}
	if c.keyboardHandlerPtr != nil {
		C.cKeyboardHandlerFree(c.keyboardHandlerPtr)
		c.keyboardHandlerPtr = nil
	}
	if c.fileDialogHandlerPtr != nil {
		C.cFileDialogHandlerFree(c.fileDialogHandlerPtr)
		c.fileDialogHandlerPtr = nil
	}
	if c.frameHandlerPtr != nil {
		C.cFrameHandlerFree(c.frameHandlerPtr)
		c.frameHandlerPtr = nil
	}
	// release the client mutex before touching the registry - the lock order
	// everywhere else is clientMutex first, client mutex second
	c.Unlock()

	if app := getInstance(); app != nil {
		app.clientMutex.Lock()
		delete(app.clients, clientKey)
		app.clientMutex.Unlock()
	}

	if opener != nil {
		opener.forgetPendingPopup(slot, c)
	}
	for _, popup := range pendingPopups {
		popup.Lock()
		popup.popupOpener = nil
		popup.Unlock()
		if closeErr := popup.closeInternal(pump); closeErr != nil {
			c.logger.Warn().Err(closeErr).Msg("error closing pending popup client")
		}
	}

	c.logger.Trace().Msg("client closed")
	return nil
}

func (c *cefApp) NewClient(options ...ClientOption) Client {
	c.clientMutex.Lock()
	defer c.clientMutex.Unlock()

	newClient := &client{
		ptr:    C.cClientCreate(),
		logger: c.options.logger.With().Str("component", "client").Logger(),

		renderHandlerPtr:          C.cRenderHandlerCreate(),
		requestHandlerPtr:         C.cRequestHandlerCreate(),
		resourceRequestHandlerPtr: C.cResourceRequestHandlerCreate(),
		lifeSpanHandlerPtr:        C.cLifeSpanHandlerCreate(),
		displayHandlerPtr:         C.cDisplayHandlerCreate(),
		loadHandlerPtr:            C.cLoadHandlerCreate(),
		downloadHandlerPtr:        C.cDownloadHandlerCreate(),
		jsDialogHandlerPtr:        C.cJsDialogHandlerCreate(),
		contextMenuHandlerPtr:     C.cContextMenuHandlerCreate(),
		findHandlerPtr:            C.cFindHandlerCreate(),
		dragHandlerPtr:            C.cDragHandlerCreate(),
		focusHandlerPtr:           C.cFocusHandlerCreate(),
		keyboardHandlerPtr:        C.cKeyboardHandlerCreate(),
		fileDialogHandlerPtr:      C.cFileDialogHandlerCreate(),
		frameHandlerPtr:           C.cFrameHandlerCreate(),

		createTimeout:     500 * time.Millisecond,
		closeTimeout:      5 * time.Second,
		defaultInitialUrl: "about:blank",

		browsers: make(map[int]*browser),
	}

	for _, option := range options {
		option(newClient)
	}
	if newClient.devToolsReceiver != nil {
		newClient.devToolsObserverPtr = C.cDevToolsObserverCreate()
	}

	c.clients[unsafe.Pointer(newClient.ptr)] = newClient
	return newClient
}
