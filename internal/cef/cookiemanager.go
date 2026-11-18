package cef

/*
#include <string.h>
#include "cookiemanager_bridge.h"
*/
import "C"
import (
	"errors"
	"math"
	"sync"
	"time"
	"unsafe"
)

type (
	CookieSameSite int
	CookiePriority int

	// Cookie mirrors cef_cookie_t. Creation and LastAccess are informational
	// and populated by the store when reading; they are ignored on SetCookie.
	// A nil Expires makes a session cookie.
	Cookie struct {
		Name  string
		Value string
		// Domain: empty creates a host-only cookie for the SetCookie url.
		// Domain cookies are stored with a leading "." and are visible to
		// sub-domains.
		Domain string
		// Path: non-empty limits the cookie to URLs at or below the path.
		Path       string
		Secure     bool
		HttpOnly   bool
		Creation   time.Time
		LastAccess time.Time
		Expires    *time.Time
		SameSite   CookieSameSite
		Priority   CookiePriority
	}

	// CookieManager reads and writes the browser cookie store. All calls
	// block until the store confirmed the operation, so they must not be
	// invoked from CEF callback goroutines that the UI thread waits on.
	// Persistence across restarts requires WithCachePath on the app.
	CookieManager interface {
		// SetCookie stores a cookie for the given URL. The URL decides host
		// and scheme checks (Secure cookies need an https URL).
		SetCookie(url string, cookie Cookie) error
		// Cookies returns every cookie in the store.
		Cookies() ([]Cookie, error)
		// CookiesForUrl returns the cookies that would be sent for url.
		CookiesForUrl(url string, includeHttpOnly bool) ([]Cookie, error)
		// DeleteCookies deletes matching cookies and reports how many went
		// away. Empty url deletes everything; url without name deletes all
		// host cookies of that url; url plus name deletes that one cookie.
		DeleteCookies(url string, cookieName string) (int, error)
		// Flush writes the backing store to disk.
		Flush() error
	}

	// cookieManager works against whatever store source yields - the global
	// manager or a request context's manager. source returns an owned
	// reference; every operation releases it when done.
	cookieManager struct {
		source func() (*C.cef_cookie_manager_t, error)
	}

	cookieVisitState struct {
		cookies []Cookie
		done    chan []Cookie
	}
)

const (
	CookieSameSiteUnspecified   = CookieSameSite(C.CEF_COOKIE_SAME_SITE_UNSPECIFIED)
	CookieSameSiteNoRestriction = CookieSameSite(C.CEF_COOKIE_SAME_SITE_NO_RESTRICTION)
	CookieSameSiteLaxMode       = CookieSameSite(C.CEF_COOKIE_SAME_SITE_LAX_MODE)
	CookieSameSiteStrictMode    = CookieSameSite(C.CEF_COOKIE_SAME_SITE_STRICT_MODE)

	CookiePriorityLow    = CookiePriority(C.CEF_COOKIE_PRIORITY_LOW)
	CookiePriorityMedium = CookiePriority(C.CEF_COOKIE_PRIORITY_MEDIUM)
	CookiePriorityHigh   = CookiePriority(C.CEF_COOKIE_PRIORITY_HIGH)
)

// cookieOperationTimeout bounds every blocking cookie call. The callbacks
// arrive on the browser process UI thread - if the message loop stopped, the
// timeout keeps callers from hanging forever.
const cookieOperationTimeout = 10 * time.Second

var (
	cookiePendingLock  sync.Mutex
	cookieVisitPending = make(map[unsafe.Pointer]*cookieVisitState)
	cookieAckPending   = make(map[unsafe.Pointer]chan int)

	errCookieTimeout      = errors.New("timed out waiting for the cookie store")
	errCookieInaccessible = errors.New("cookies cannot be accessed")
)

// GlobalCookieManager returns the cookie manager of the global request
// context. Cookie data lives at the app's WithCachePath location, or in
// memory when no cache path is set.
func GlobalCookieManager() CookieManager {
	return cookieManager{source: globalManagerPtr}
}

// forgetCookiePending resolves bookkeeping for a visitor or callback struct
// that is being destroyed. Called from goMemoryRelease on final release; a
// no-op for unrelated pointers. A visitor resolves with whatever was
// collected - CEF releases it after the last visit call, which is the only
// end-of-visit signal (visit never fires for an empty store). An ack channel
// still present here means on_complete never fired; closing it unblocks the
// waiter into its error path.
func forgetCookiePending(it unsafe.Pointer) {
	cookiePendingLock.Lock()
	visit, visitFound := cookieVisitPending[it]
	delete(cookieVisitPending, it)
	ack, ackFound := cookieAckPending[it]
	delete(cookieAckPending, it)
	cookiePendingLock.Unlock()

	if visitFound {
		visit.done <- visit.cookies
	}
	if ackFound {
		close(ack)
	}
}

//export goCookieVisit
func goCookieVisit(self *C.cef_cookie_visitor_t, cookie *C.struct__cef_cookie_t, count C.int, total C.int) C.int {
	cookiePendingLock.Lock()
	state := cookieVisitPending[unsafe.Pointer(self)]
	if state != nil {
		state.cookies = append(state.cookies, cookieFromCef(cookie))
	}
	cookiePendingLock.Unlock()

	if state == nil {
		// the waiter timed out - stop visiting
		return 0
	}
	return 1
}

//export goCookieAckComplete
func goCookieAckComplete(self unsafe.Pointer, value C.int) {
	cookiePendingLock.Lock()
	ack, found := cookieAckPending[self]
	delete(cookieAckPending, self)
	cookiePendingLock.Unlock()

	if found {
		ack <- int(value)
	}
}

func baseTimeToGoTime(baseTime C.cef_basetime_t) time.Time {
	var cefTime C.cef_time_t
	if C.cef_time_from_basetime(baseTime, &cefTime) == 0 {
		return time.Time{}
	}
	var seconds C.double
	if C.cef_time_to_doublet(&cefTime, &seconds) == 0 || seconds == 0 {
		return time.Time{}
	}
	whole, frac := math.Modf(float64(seconds))
	return time.Unix(int64(whole), int64(frac*1e9))
}

func goTimeToBaseTime(t time.Time) C.cef_basetime_t {
	var cefTime C.cef_time_t
	// cef_time_from_doublet expects seconds since epoch
	C.cef_time_from_doublet(C.double(float64(t.UnixMilli())/1e3), &cefTime)
	var baseTime C.cef_basetime_t
	C.cef_time_to_basetime(&cefTime, &baseTime)
	return baseTime
}

func cookieFromCef(cookie *C.struct__cef_cookie_t) Cookie {
	result := Cookie{
		Name:       cefToGoString(&cookie.name),
		Value:      cefToGoString(&cookie.value),
		Domain:     cefToGoString(&cookie.domain),
		Path:       cefToGoString(&cookie.path),
		Secure:     cookie.secure != 0,
		HttpOnly:   cookie.httponly != 0,
		Creation:   baseTimeToGoTime(cookie.creation),
		LastAccess: baseTimeToGoTime(cookie.last_access),
		SameSite:   CookieSameSite(cookie.same_site),
		Priority:   CookiePriority(cookie.priority),
	}
	if cookie.has_expires != 0 {
		expires := baseTimeToGoTime(cookie.expires)
		result.Expires = &expires
	}
	return result
}

// cookieToCef builds a C-allocated cef_cookie_t. Free with cookieFreeCef.
func cookieToCef(cookie Cookie) *C.struct__cef_cookie_t {
	final := (*C.struct__cef_cookie_t)(C.calloc(1, C.sizeof_struct__cef_cookie_t))
	final.size = C.sizeof_struct__cef_cookie_t
	cefStringInto(&final.name, cookie.Name)
	cefStringInto(&final.value, cookie.Value)
	cefStringInto(&final.domain, cookie.Domain)
	cefStringInto(&final.path, cookie.Path)
	if cookie.Secure {
		final.secure = 1
	}
	if cookie.HttpOnly {
		final.httponly = 1
	}
	if cookie.Expires != nil {
		final.has_expires = 1
		final.expires = goTimeToBaseTime(*cookie.Expires)
	}
	final.same_site = C.cef_cookie_same_site_t(cookie.SameSite)
	final.priority = C.cef_cookie_priority_t(cookie.Priority)
	return final
}

func cookieFreeCef(cookie *C.struct__cef_cookie_t) {
	C.cef_string_clear(&cookie.name)
	C.cef_string_clear(&cookie.value)
	C.cef_string_clear(&cookie.domain)
	C.cef_string_clear(&cookie.path)
	C.free(unsafe.Pointer(cookie))
}

func globalManagerPtr() (*C.cef_cookie_manager_t, error) {
	if getInstance() == nil {
		return nil, errors.New("cef-app is not initialized")
	}
	manager := C.cCookieManagerGetGlobal()
	if manager == nil {
		return nil, errors.New("no global cookie manager available")
	}
	return manager, nil
}

// visitCookies runs one of the visit_*_cookies calls and waits for the
// collected result. run receives the registered visitor and returns the CEF
// success flag.
func (m cookieManager) visitCookies(run func(*C.cef_cookie_manager_t, *C.cef_cookie_visitor_t) C.int) ([]Cookie, error) {
	manager, err := m.source()
	if err != nil {
		return nil, err
	}
	defer releaseBase(unsafe.Pointer(manager))

	visitor := C.cCookieVisitorCreate()
	state := &cookieVisitState{done: make(chan []Cookie, 1)}
	cookiePendingLock.Lock()
	cookieVisitPending[unsafe.Pointer(visitor)] = state
	cookiePendingLock.Unlock()

	// the visitor reference is consumed by the visit call; the final release
	// resolves state.done through forgetCookiePending
	failed := run(manager, visitor) == 0

	// the visitor runs on the UI thread (see uiWaitPump)
	uw := beginUIWait()
	defer uw.stop()
	timeout := time.NewTimer(cookieOperationTimeout)
	defer timeout.Stop()

	for {
		select {
		case cookies := <-state.done:
			if failed {
				return nil, errCookieInaccessible
			}
			return cookies, nil
		case <-uw.tick():
			uw.run()
		case <-timeout.C:
			cookiePendingLock.Lock()
			delete(cookieVisitPending, unsafe.Pointer(visitor))
			cookiePendingLock.Unlock()
			return nil, errCookieTimeout
		}
	}
}

// awaitAck waits on the channel registered for a callback struct. failed
// marks the CEF call itself as unsuccessful; the channel closing without a
// value means the callback never fired. The channel is captured at
// registration - goCookieAckComplete removes the map entry before sending,
// so a map lookup here would race.
func awaitAck(callback unsafe.Pointer, ack chan int, failed bool) (int, error) {
	// cookie callbacks run on the UI thread (see uiWaitPump)
	uw := beginUIWait()
	defer uw.stop()
	timeout := time.NewTimer(cookieOperationTimeout)
	defer timeout.Stop()

	for {
		select {
		case value, delivered := <-ack:
			if failed || !delivered {
				return 0, errCookieInaccessible
			}
			return value, nil
		case <-uw.tick():
			uw.run()
		case <-timeout.C:
			cookiePendingLock.Lock()
			delete(cookieAckPending, callback)
			cookiePendingLock.Unlock()
			return 0, errCookieTimeout
		}
	}
}

func registerAck(callback unsafe.Pointer) chan int {
	ack := make(chan int, 1)
	cookiePendingLock.Lock()
	cookieAckPending[callback] = ack
	cookiePendingLock.Unlock()
	return ack
}

func (m cookieManager) Cookies() ([]Cookie, error) {
	return m.visitCookies(func(manager *C.cef_cookie_manager_t, visitor *C.cef_cookie_visitor_t) C.int {
		return C.cCookieManagerVisitAllCookies(manager, visitor)
	})
}

func (m cookieManager) CookiesForUrl(url string, includeHttpOnly bool) ([]Cookie, error) {
	cUrl := cefString(url)
	defer cefStringFree(cUrl)

	include := C.int(0)
	if includeHttpOnly {
		include = 1
	}
	return m.visitCookies(func(manager *C.cef_cookie_manager_t, visitor *C.cef_cookie_visitor_t) C.int {
		return C.cCookieManagerVisitUrlCookies(manager, cUrl, include, visitor)
	})
}

func (m cookieManager) SetCookie(url string, cookie Cookie) error {
	manager, err := m.source()
	if err != nil {
		return err
	}
	defer releaseBase(unsafe.Pointer(manager))

	cUrl := cefString(url)
	defer cefStringFree(cUrl)
	cCookie := cookieToCef(cookie)
	defer cookieFreeCef(cCookie)

	callback := C.cSetCookieCallbackCreate()
	ack := registerAck(unsafe.Pointer(callback))

	failed := C.cCookieManagerSetCookie(manager, cUrl, cCookie, callback) == 0
	success, err := awaitAck(unsafe.Pointer(callback), ack, failed)
	if err != nil {
		return err
	}
	if success == 0 {
		return errors.New("cookie was rejected by the store")
	}
	return nil
}

func (m cookieManager) DeleteCookies(url string, cookieName string) (int, error) {
	manager, err := m.source()
	if err != nil {
		return 0, err
	}
	defer releaseBase(unsafe.Pointer(manager))

	// NULL url deletes all cookies, so empty strings stay nil here
	var cUrl, cName *C.cef_string_t
	if url != "" {
		cUrl = cefString(url)
		defer cefStringFree(cUrl)
	}
	if cookieName != "" {
		cName = cefString(cookieName)
		defer cefStringFree(cName)
	}

	callback := C.cDeleteCookiesCallbackCreate()
	ack := registerAck(unsafe.Pointer(callback))

	failed := C.cCookieManagerDeleteCookies(manager, cUrl, cName, callback) == 0
	return awaitAck(unsafe.Pointer(callback), ack, failed)
}

func (m cookieManager) Flush() error {
	manager, err := m.source()
	if err != nil {
		return err
	}
	defer releaseBase(unsafe.Pointer(manager))

	callback := C.cCompletionCallbackCreate()
	ack := registerAck(unsafe.Pointer(callback))

	failed := C.cCookieManagerFlushStore(manager, callback) == 0
	_, err = awaitAck(unsafe.Pointer(callback), ack, failed)
	return err
}
