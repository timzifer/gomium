package cef

/*
#include <string.h>
#include "browser_client_bridge.h"
#include "requesthandler_bridge.h"
*/
import "C"
import (
	"sync"
	"unsafe"
)

type (
	// AuthRequest describes an HTTP Basic/Digest/NTLM or proxy auth challenge.
	AuthRequest struct {
		OriginURL string
		Host      string
		Port      int
		Realm     string
		Scheme    string
		IsProxy   bool
	}

	// AuthAction tells CEF how the handler treated a challenge.
	AuthAction int

	// AuthCallback finishes a challenge the handler took over. Exactly one of
	// Continue or Cancel must be called. Safe from any goroutine. Cancel fails
	// the request (the page sees an auth error).
	AuthCallback interface {
		Continue(username, password string)
		Cancel()
	}

	// AuthHandler answers HTTP and proxy auth challenges. Runs synchronously on
	// the browser process UI thread.
	//
	// Escape hatch: return AuthHandled to take the challenge over (e.g. show a
	// Fyne login form) and call the callback later - or call it right away.
	// Return AuthDefault to cancel the challenge (CEF's own dialog is disabled).
	AuthHandler interface {
		OnAuthCredentials(instance Browser, request AuthRequest, callback AuthCallback) AuthAction
	}

	authCallback struct {
		ptr  *C.cef_auth_callback_t
		once *sync.Once
	}
)

const (
	// AuthDefault cancels the challenge; CEF's default dialog is disabled.
	AuthDefault AuthAction = iota
	// AuthHandled: the handler takes the challenge and finishes it through
	// the callback. Not calling the callback leaves the request pending.
	AuthHandled
)

func (c authCallback) Continue(username, password string) {
	c.once.Do(func() {
		defer releaseBase(unsafe.Pointer(c.ptr))

		user := cefString(username)
		defer cefStringFree(user)
		pass := cefString(password)
		defer cefStringFree(pass)

		C.cBridgeAuthCallbackContinue(c.ptr, user, pass)
	})
}

func (c authCallback) Cancel() {
	c.once.Do(func() {
		defer releaseBase(unsafe.Pointer(c.ptr))
		C.cBridgeAuthCallbackCancel(c.ptr)
	})
}

// discard drops the reference without touching the callback - used when the
// handler did not take the challenge over.
func (c authCallback) discard() {
	c.once.Do(func() {
		releaseBase(unsafe.Pointer(c.ptr))
	})
}

//export goOnAuthCredentials
func goOnAuthCredentials(self *C.cef_request_handler_t,
	cefBrowser *C.cef_browser_t,
	originUrl *C.cef_string_t,
	isProxy C.int,
	host *C.cef_string_t,
	port C.int,
	realm *C.cef_string_t,
	scheme *C.cef_string_t,
	callback *C.cef_auth_callback_t) C.int {
	defer releaseBase(unsafe.Pointer(cefBrowser))

	wrapped := authCallback{ptr: callback, once: &sync.Once{}}

	c := clientWhere(func(c *client) bool { return c.requestHandlerPtr == self })
	if c == nil {
		wrapped.discard()
		return 0
	}

	c.Lock()
	handler := c.authHandler
	c.Unlock()
	if handler == nil {
		wrapped.discard()
		return 0
	}

	browserInstance := c.browserByIdentifier(int(C.cBridgeGetBrowserIdentifier(cefBrowser)))
	req := AuthRequest{
		OriginURL: cefToGoString(originUrl),
		Host:      cefToGoString(host),
		Port:      int(port),
		Realm:     cefToGoString(realm),
		Scheme:    cefToGoString(scheme),
		IsProxy:   isProxy != 0,
	}

	if handler.OnAuthCredentials(browserInstance, req, wrapped) == AuthHandled {
		return 1
	}
	wrapped.discard()
	return 0
}
