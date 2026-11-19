package cef

/*
#include <stdlib.h>
#include "app_bridge.h"
#include "browser_client_bridge.h"
#include "resourcehandler_bridge.h"
#include "scheme_bridge.h"
*/
import "C"
import (
	"fmt"
	"regexp"
	"unsafe"
)

type (
	// SchemeOptions is the bitmask describing how Chromium treats a custom
	// scheme. Combine the Scheme* constants; SchemeStandard is what makes a
	// scheme carry an origin ("help://content/index.html") instead of being
	// an opaque URL, and everything except SchemeLocal builds on it.
	SchemeOptions int

	// CustomScheme is a scheme the embedder serves itself. Register it with
	// WithCustomScheme before the app starts and answer its requests from a
	// ResourceInterceptor.
	CustomScheme struct {
		// Name is the scheme without "://", lower case, e.g. "help".
		Name string
		// Options describes how Chromium treats the scheme.
		Options SchemeOptions
	}
)

const (
	// SchemeStandard gives the scheme an origin and the usual URL parsing
	// ("scheme://host/path"). Required for everything below except
	// SchemeLocal, and required for a page that wants storage or CORS.
	SchemeStandard SchemeOptions = 1 << 0
	// SchemeLocal treats the scheme like file://: other origins may not link
	// to it, and it may not be loaded from a remote page.
	SchemeLocal SchemeOptions = 1 << 1
	// SchemeDisplayIsolated allows the scheme to be displayed only from
	// content of the same scheme.
	SchemeDisplayIsolated SchemeOptions = 1 << 2
	// SchemeSecure treats the scheme like https: no mixed-content warnings,
	// and features that require a secure context work.
	SchemeSecure SchemeOptions = 1 << 3
	// SchemeCorsEnabled allows CORS requests to the scheme.
	SchemeCorsEnabled SchemeOptions = 1 << 4
	// SchemeCspBypassing exempts the scheme from Content-Security-Policy.
	SchemeCspBypassing SchemeOptions = 1 << 5
	// SchemeFetchEnabled allows fetch() and XMLHttpRequest against the
	// scheme.
	SchemeFetchEnabled SchemeOptions = 1 << 6
)

// schemeNamePattern is what RFC 3986 allows in a scheme, restricted to lower
// case: CEF compares scheme names case-sensitively against the lower-cased
// URL, so an upper-case registration silently never matches.
var schemeNamePattern = regexp.MustCompile(`^[a-z][a-z0-9+.-]*$`)

// builtinSchemes must not be re-registered - CEF documents add_custom_scheme
// as "should not be called" for them, and doing it anyway breaks the built-in
// handling.
var builtinSchemes = map[string]bool{
	"http": true, "https": true, "file": true, "ftp": true,
	"about": true, "data": true, "blob": true, "javascript": true,
	"chrome": true, "chrome-extension": true,
}

// WithCustomScheme registers a scheme the embedder serves itself, e.g.
//
//	WithCustomScheme("help", SchemeStandard|SchemeSecure|SchemeFetchEnabled)
//
// Requests to it reach the ResourceInterceptor of the browser's
// RequestHandler, exactly like an intercepted http request; returning nil
// there fails the request instead of falling back to a network load, because
// nothing else can serve the scheme.
//
// The option MUST be passed identically to New and to ExecuteSubprocess:
// Chromium registers schemes per process, and a renderer that does not know
// the scheme refuses to parse its URLs - the page then fails with
// ERR_UNKNOWN_URL_SCHEME even though the browser process would have served
// it.
func WithCustomScheme(name string, options SchemeOptions) Option {
	return func(c *cefApp) {
		switch {
		case !schemeNamePattern.MatchString(name):
			c.options.optionErrors = append(c.options.optionErrors,
				fmt.Errorf("WithCustomScheme(%q): not a lower-case scheme name", name))
			return
		case builtinSchemes[name]:
			c.options.optionErrors = append(c.options.optionErrors,
				fmt.Errorf("WithCustomScheme(%q): built-in scheme, CEF handles it", name))
			return
		}

		for _, existing := range c.options.customSchemes {
			if existing.Name == name {
				c.options.optionErrors = append(c.options.optionErrors,
					fmt.Errorf("WithCustomScheme(%q): already registered", name))
				return
			}
		}

		c.options.customSchemes = append(c.options.customSchemes, CustomScheme{
			Name:    name,
			Options: options,
		})
	}
}

//export goOnRegisterCustomSchemes
func goOnRegisterCustomSchemes(_ *C.cef_app_t, registrar *C.cef_scheme_registrar_t) {
	app := getInstance()
	if app == nil || registrar == nil {
		return
	}

	for _, scheme := range app.options.customSchemes {
		name := C.CString(scheme.Name)
		registered := C.cSchemeRegistrarAdd(registrar, name, C.int(scheme.Options))
		C.free(unsafe.Pointer(name))

		if registered == 0 {
			// Not fatal for this process, but the scheme will not load
			// anywhere - say so rather than leave a page failing with a bare
			// ERR_UNKNOWN_URL_SCHEME.
			app.options.logger.Error().Str("scheme", scheme.Name).
				Msg("CEF refused the custom scheme registration")
		}
	}
}

// registerSchemeHandlerFactories points one request context at the shared
// factory for every custom scheme. Runs in the browser process only, after
// the context is initialized: the global context does not exist before that.
//
// A factory is what routes a custom-scheme request into the resource
// machinery at all - unlike http, an unclaimed custom scheme has no default
// loader behind it and fails outright.
func (c *cefApp) registerSchemeHandlerFactories(context *C.cef_request_context_t) {
	if context == nil || len(c.options.customSchemes) == 0 {
		return
	}

	if c.schemeHandlerFactory == nil {
		c.schemeHandlerFactory = C.cSchemeHandlerFactoryCreate()
	}

	for _, scheme := range c.options.customSchemes {
		name := C.CString(scheme.Name)
		// register_scheme_handler_factory consumes a reference - also when
		// it fails, CEF unwraps the argument before it registers anything -
		// so a failure must not release it again. The one created above
		// stays with the app and is released in freeNativeStructs.
		addRefBase(unsafe.Pointer(c.schemeHandlerFactory))
		registered := C.cRequestContextRegisterSchemeHandlerFactory(context, name, c.schemeHandlerFactory)
		C.free(unsafe.Pointer(name))

		if registered == 0 {
			c.options.logger.Error().Str("scheme", scheme.Name).
				Msg("could not register the scheme handler factory")
		}
	}
}

// clearSchemeHandlerFactories unregisters the factories of the global request
// context. Close calls it before cef_shutdown: CEF does not let go of them on
// its own, and each one holds a reference on the app's factory. Contexts from
// NewRequestContext drop theirs when they are destroyed.
func (c *cefApp) clearSchemeHandlerFactories() {
	if len(c.options.customSchemes) == 0 || c.schemeHandlerFactory == nil {
		return
	}
	if C.cClearGlobalSchemeHandlerFactories() == 0 {
		c.options.logger.Warn().Msg("could not clear the scheme handler factories")
	}
}

//export goSchemeHandlerFactoryCreate
func goSchemeHandlerFactoryCreate(
	_ *C.cef_scheme_handler_factory_t,
	cefBrowser *C.cef_browser_t,
	frame *C.cef_frame_t,
	cefRequest *C.cef_request_t,
) *C.cef_resource_handler_t {
	if cefBrowser != nil {
		defer releaseBase(unsafe.Pointer(cefBrowser))
	}
	if frame != nil {
		defer releaseBase(unsafe.Pointer(frame))
	}

	// the request wrapper owns the parameter reference; URL and method are
	// read here so the interceptor never touches the request afterwards
	r := From(unsafe.Pointer(cefRequest))
	defer r.invalidateAndRelease()

	if cefBrowser == nil {
		// no browser behind the request (cef_urlrequest, service worker) -
		// there is no RequestHandler to ask
		return nil
	}

	handler, browserInstance := requestHandlerByBrowser(cefBrowser)
	interceptor, isInterceptor := handler.(ResourceInterceptor)
	if !isInterceptor {
		return nil
	}

	requestUrl, urlErr := r.GetUrl()
	if urlErr != nil {
		return nil
	}

	fn := interceptor.InterceptResource(browserInstance, requestUrl, r.GetMethod())
	if fn == nil {
		return nil
	}

	// the create reference transfers to CEF - lookup happens via the registry
	resourceHandler := C.cResourceHandlerCreate()
	registerResourceServe(resourceHandler, fn)
	return resourceHandler
}

// requestHandlerByBrowser resolves the client owning a browser. The scheme
// handler factory is shared by all clients (it is registered per request
// context, not per browser), so unlike the resource callbacks it can only
// match through the browser it is handed.
func requestHandlerByBrowser(cefBrowser *C.cef_browser_t) (RequestHandler, Browser) {
	app := getInstance()
	if app == nil {
		return nil, nil
	}

	identifier := int(C.cBridgeGetBrowserIdentifier(cefBrowser))

	app.clientMutex.Lock()
	clients := make([]*client, 0, len(app.clients))
	for _, c := range app.clients {
		clients = append(clients, c)
	}
	app.clientMutex.Unlock()

	for _, c := range clients {
		if browserInstance := c.browserByIdentifier(identifier); browserInstance != nil {
			return c.requestHandler, browserInstance
		}
	}
	return nil, nil
}
