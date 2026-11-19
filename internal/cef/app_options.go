package cef

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"time"
)

type (
	appOptions struct {
		logger                *slog.Logger
		pumpWorker            WorkerCallback
		initializationTimeout time.Duration
		moduleHandle          uintptr
		pumpWorkerInterval    time.Duration

		// customSchemes are registered in every process from
		// goOnRegisterCustomSchemes and served through a scheme handler
		// factory in the browser process (see WithCustomScheme).
		customSchemes []CustomScheme

		// defaultBindingGuard guards the binding calls of clients without a
		// guard of their own, popups included (WithDefaultBindingGuard).
		defaultBindingGuard BindingGuard

		// bindings are the Go functions exposed to page JavaScript (WithBinding).
		// Both processes build the same list from the same options.
		bindings []bindingFunc

		// optionErrors collects what the Option functions could not report
		// themselves - Option returns nothing, so a bad argument would
		// otherwise only surface as a bare "could not initialize cef-app"
		// out of cef_initialize. New joins and returns them.
		optionErrors []error
	}
	Option         func(*cefApp)
	Worker         func()
	WorkerCallback func(delay int, worker Worker)
)

func WithMessagePumpWorkerInterval(interval time.Duration) Option {
	return func(o *cefApp) {
		o.options.pumpWorkerInterval = interval
	}
}

func WithMessagePumpWorker(pumpWorker WorkerCallback) Option {
	return func(a *cefApp) {
		a.options.pumpWorker = pumpWorker
	}
}

// WithLogger sets the logger of the app and its clients. nil switches
// logging off, which is also the default.
func WithLogger(logger *slog.Logger) Option {
	return func(c *cefApp) {
		c.options.logger = orNop(logger)
	}
}

func WithLogSeverity(severity LogSeverity) Option {
	return func(c *cefApp) {
		c.settings.LogSeverity = severity
	}
}

func WithIgnoreCertificateErrors(ignore bool) Option {
	return func(c *cefApp) {
		c.settings.IgnoreCertificateErrors = ignore
	}
}

func WithInitializationTimeout(duration time.Duration) Option {
	return func(c *cefApp) {
		c.options.initializationTimeout = duration
	}
}

func WithCachePath(cachePath string) Option {
	return func(c *cefApp) {
		c.settings.CachePath = &cachePath
	}
}

// WithRootCachePath sets the parent directory that every cache path must
// live in - the global one and all request-context paths (see
// WithContextCachePath). Defaults to the WithCachePath value.
func WithRootCachePath(rootCachePath string) Option {
	return func(c *cefApp) {
		c.settings.RootCachePath = &rootCachePath
	}
}

func WithUserAgent(agent string) Option {
	return func(c *cefApp) {
		c.settings.UserAgent = &agent
	}
}

func WithRemoteDebuggingPort(port int) Option {
	return func(c *cefApp) {
		c.settings.RemoteDebuggingPort = &port
	}
}

// WithBrowserSubprocessPath points CEF at a separate executable for its helper
// processes (renderer, gpu, utility, ...). path must be absolute.
//
// Leave it unset unless the helpers really are a different binary. Empty means
// the main executable on Windows and Linux, and on macOS it means CEF derives
// the helper from the app bundle - "Contents/Frameworks/<app> Helper.app/
// Contents/MacOS/<app> Helper", which is the layout tools/cefbundle produces.
func WithBrowserSubprocessPath(path string) Option {
	return func(c *cefApp) {
		if abs, ok := absoluteOption(c, "WithBrowserSubprocessPath", path); ok {
			c.settings.BrowserSubprocessPath = &abs
		}
	}
}

// WithResourcesDirPath points CEF at the directory holding cef.pak,
// chrome_100_percent.pak, icudtl.dat and friends. path must be absolute.
//
// Empty means CEF looks next to the executable, which is what a distribution
// laid out the usual way provides. Set it when the resources live somewhere
// else - a shared installation directory, or a development tree where the
// binary is built into ./build and the runtime sits in ./Release.
//
// Ignored on macOS: there the resources are inside the framework bundle, and
// WithFrameworkDirPath is what points at them.
func WithResourcesDirPath(path string) Option {
	return func(c *cefApp) {
		if abs, ok := absoluteOption(c, "WithResourcesDirPath", path); ok {
			c.settings.ResourcesDirPath = &abs
		}
	}
}

// WithLocalesDirPath points CEF at the directory holding the locale .pak
// files. path must be absolute.
//
// Empty means the "locales" directory next to the executable. Ignored on
// macOS for the same reason as WithResourcesDirPath.
func WithLocalesDirPath(path string) Option {
	return func(c *cefApp) {
		if abs, ok := absoluteOption(c, "WithLocalesDirPath", path); ok {
			c.settings.LocalesDirPath = &abs
		}
	}
}

// absoluteOption turns path into an absolute one for the cef_settings_t fields
// that CEF documents as "must be an absolute path". A path that cannot be
// resolved is recorded and dropped instead of being handed to CEF, where it
// would come back as an unspecific initialization failure.
func absoluteOption(c *cefApp, option, path string) (string, bool) {
	if path == "" {
		c.options.optionErrors = append(c.options.optionErrors, fmt.Errorf("%s: path is empty", option))
		return "", false
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		c.options.optionErrors = append(c.options.optionErrors, fmt.Errorf("%s(%q): %w", option, path, err))
		return "", false
	}
	return abs, true
}

// WithV8Listener registers a listener for the V8 contexts of the render
// process (see V8Listener). Like every option it has to reach both
// processes: pass it to New and to ExecuteSubprocess - the listener only
// ever runs in the render process, where the contexts live.
func WithV8Listener(listener V8Listener) Option {
	return func(c *cefApp) {
		if listener != nil {
			c.v8listeners = append(c.v8listeners, listener)
		}
	}
}
