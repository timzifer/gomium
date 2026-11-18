package main

import (
	"errors"
	"net/url"
	"os"
	"strings"

	"github.com/rs/zerolog"
	"github.com/timzifer/gomium"
)

// GOMIUM_SCHEME_TEST=1 exercises WithCustomScheme end to end: the page, its
// stylesheet and an image all come from Go through the ResourceInterceptor,
// with nothing on disk and no server behind them.
//
// What the run has to show in the log:
//   - "load end" with status 200 and no "load error"
//   - the page's console lines: the stylesheet applied (proving the relative
//     path resolved against the scheme's origin) and isSecureContext=true
//     (proving SchemeSecure arrived)
const (
	schemeTestName = "gomium"
	schemeTestHost = "content"
	schemeTestURL  = schemeTestName + "://" + schemeTestHost + "/index.html"
)

func schemeTestEnabled() bool { return os.Getenv("GOMIUM_SCHEME_TEST") != "" }

// schemeTestOptions must reach New AND ExecuteSubprocess - a renderer that
// does not know the scheme refuses to parse its URLs.
func schemeTestOptions() []gomium.Option {
	if !schemeTestEnabled() {
		return nil
	}
	return []gomium.Option{
		gomium.WithCustomScheme(schemeTestName, gomium.SchemeStandard|gomium.SchemeSecure),
	}
}

// GOMIUM_BINDING_TEST=1: bindings for the page in bindingPage. Both processes
// get the same options, so the renderer knows the same functions.
func bindingTestOptions() []gomium.Option {
	if os.Getenv("GOMIUM_BINDING_TEST") == "" && os.Getenv("GOMIUM_BINDING_POPUP_TEST") == "" {
		return nil
	}
	opts := []gomium.Option{
		gomium.WithBinding("mde.ping", func() (string, error) { return "pong", nil }),
		gomium.WithBinding("mde.add", func(a, b float64) (float64, error) { return a + b, nil }),
		gomium.WithBinding("mde.fail", func() error { return errors.New("boom") }),
		// waits for a UI-thread callback itself: before bindings left the UI
		// thread this blocked until the cookie manager's timeout
		gomium.WithBinding("mde.cookieCount", func() (int, error) {
			cookies, err := gomium.GlobalCookieManager().CookiesForUrl("https://example.org/", true)
			return len(cookies), err
		}),
	}
	// popup mode: browsers without a guard of their own get the default, which
	// rejects everything, so the popup's result shows which guard it got
	if os.Getenv("GOMIUM_BINDING_POPUP_TEST") != "" {
		opts = append(opts, gomium.WithDefaultBindingGuard(func(gomium.BindingCall) bool { return false }))
	}
	return opts
}

// bindingTestGuard allows binding calls from the top frame of the test page
// only. The iframe on the same page is rejected, which proves the guard.
func bindingTestGuard(call gomium.BindingCall) bool {
	return call.MainFrame && call.FrameURL != nil && call.FrameURL.Port() == "18779"
}

const schemeTestPage = `<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<title>gomium scheme test</title>
<link rel="stylesheet" href="style.css">
</head>
<body>
<h1>served from Go</h1>
<script>
window.addEventListener('load', function () {
  var color = getComputedStyle(document.querySelector('h1')).color;
  console.log('scheme-test: heading color is ' + color + ' (want rgb(0, 128, 0))');
  console.log('scheme-test: isSecureContext=' + window.isSecureContext);
  console.log('scheme-test: origin=' + window.location.origin);
});
</script>
</body>
</html>`

const schemeTestStyle = `h1 { color: rgb(0, 128, 0); }`

// schemeTestHandler answers every request of the custom scheme and lets
// everything else pass.
type schemeTestHandler struct {
	logger zerolog.Logger
}

func (schemeTestHandler) OnProtocolExecution(_ gomium.Browser, _ *url.URL) bool { return false }
func (schemeTestHandler) OnBeforeBrowse(_ gomium.Browser, _ *url.URL) bool      { return false }
func (schemeTestHandler) OnBeforeResourceLoad(_ gomium.Browser, _ *gomium.Request, _ gomium.ResourceLoadAsyncResult) gomium.ResourceLoadResult {
	return gomium.ResourceLoadContinue
}

func (h schemeTestHandler) InterceptResource(_ gomium.Browser, requestUrl *url.URL, method string) gomium.ResourceContentFn {
	if requestUrl.Scheme != schemeTestName {
		return nil
	}
	h.logger.Info().Str("uri", requestUrl.String()).Str("method", method).Msg("scheme-test: serving")

	var content *gomium.ResourceContent
	switch strings.TrimPrefix(requestUrl.Path, "/") {
	case "index.html":
		content = &gomium.ResourceContent{
			Data:     []byte(schemeTestPage),
			MimeType: "text/html; charset=utf-8",
		}
	case "style.css":
		content = &gomium.ResourceContent{
			Data:     []byte(schemeTestStyle),
			MimeType: "text/css; charset=utf-8",
		}
	default:
		h.logger.Warn().Str("uri", requestUrl.String()).Msg("scheme-test: nothing behind this path")
		return nil
	}

	return func() *gomium.ResourceContent { return content }
}
