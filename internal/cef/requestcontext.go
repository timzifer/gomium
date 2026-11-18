package cef

/*
#include <string.h>
#include "requestcontext_bridge.h"
*/
import "C"
import (
	"errors"
	"unsafe"
)

type (
	// RequestContext is an isolated browsing profile: its own cookies,
	// localStorage, cache and credentials. Browsers created with the same
	// context share storage; browsers with different contexts are fully
	// separated from each other and from the global context.
	RequestContext interface {
		// CookieManager returns the cookie store scoped to this context.
		CookieManager() CookieManager
		// CachePath returns the context's disk location; empty for an
		// in-memory (incognito) context.
		CachePath() string
		// IsGlobal reports whether this is the global context that browsers
		// without an explicit context use.
		IsGlobal() bool
		// Release drops the Go-held reference. Browsers created with the
		// context keep it alive on the CEF side; the Go wrapper must not be
		// used afterwards.
		Release()
	}

	requestContext struct {
		ptr *C.cef_request_context_t
	}

	RequestContextOption func(*requestContextSettings)

	requestContextSettings struct {
		cachePath             string
		persistSessionCookies bool
		acceptLanguageList    string
	}
)

// WithContextCachePath places the context's profile data in dir. The
// directory must be equal to or a child of the app's root cache path (set
// via WithCachePath or WithRootCachePath) - CEF rejects paths outside of it.
// Without this option the context is in-memory: nothing persists, which
// makes NewRequestContext() an incognito profile.
func WithContextCachePath(dir string) RequestContextOption {
	return func(s *requestContextSettings) {
		s.cachePath = dir
	}
}

// WithPersistSessionCookies also persists cookies without an expiry date.
// Only meaningful together with WithContextCachePath.
func WithPersistSessionCookies(persist bool) RequestContextOption {
	return func(s *requestContextSettings) {
		s.persistSessionCookies = persist
	}
}

// WithAcceptLanguageList sets the comma-delimited Accept-Language header and
// navigator.language value for browsers using this context.
func WithAcceptLanguageList(languages string) RequestContextOption {
	return func(s *requestContextSettings) {
		s.acceptLanguageList = languages
	}
}

// NewRequestContext creates an isolated browsing profile. Without options the
// context lives purely in memory (incognito). Pass the result to browsers via
// WithRequestContext.
func NewRequestContext(opts ...RequestContextOption) (RequestContext, error) {
	if getInstance() == nil {
		return nil, errors.New("cef-app is not initialized")
	}

	var s requestContextSettings
	for _, opt := range opts {
		opt(&s)
	}

	cSettings := (*C.cef_request_context_settings_t)(C.calloc(1, C.sizeof_cef_request_context_settings_t))
	defer func() {
		C.cef_string_clear(&cSettings.cache_path)
		C.cef_string_clear(&cSettings.accept_language_list)
		C.free(unsafe.Pointer(cSettings))
	}()

	cSettings.size = C.sizeof_cef_request_context_settings_t
	if s.cachePath != "" {
		cefStringInto(&cSettings.cache_path, s.cachePath)
	}
	if s.persistSessionCookies {
		cSettings.persist_session_cookies = 1
	}
	if s.acceptLanguageList != "" {
		cefStringInto(&cSettings.accept_language_list, s.acceptLanguageList)
	}

	ptr := C.cef_request_context_create_context(cSettings, nil)
	if ptr == nil {
		return nil, errors.New("request context creation failed")
	}

	// Scheme handler factories are registered per request context, so a
	// browser in its own context would otherwise not reach the custom
	// schemes the app registered (see WithCustomScheme).
	if app := getInstance(); app != nil {
		app.registerSchemeHandlerFactories(ptr)
	}

	return &requestContext{ptr: ptr}, nil
}

// GlobalRequestContext returns the context that browsers without an explicit
// context use. Useful to compare via IsGlobal or to reach its CookieManager
// through the same interface.
func GlobalRequestContext() (RequestContext, error) {
	if getInstance() == nil {
		return nil, errors.New("cef-app is not initialized")
	}
	ptr := C.cef_request_context_get_global_context()
	if ptr == nil {
		return nil, errors.New("no global request context available")
	}
	return &requestContext{ptr: ptr}, nil
}

func (r *requestContext) CookieManager() CookieManager {
	return cookieManager{source: func() (*C.cef_cookie_manager_t, error) {
		if r.ptr == nil {
			return nil, errors.New("request context already released")
		}
		manager := C.cRequestContextGetCookieManager(r.ptr)
		if manager == nil {
			return nil, errors.New("no cookie manager for this request context")
		}
		return manager, nil
	}}
}

func (r *requestContext) CachePath() string {
	if r.ptr == nil {
		return ""
	}
	return cefUserfreeToGoString(C.cRequestContextGetCachePath(r.ptr))
}

func (r *requestContext) IsGlobal() bool {
	return r.ptr != nil && C.cRequestContextIsGlobal(r.ptr) != 0
}

func (r *requestContext) Release() {
	if r.ptr != nil {
		releaseBase(unsafe.Pointer(r.ptr))
		r.ptr = nil
	}
}
