package cef

/*
#include "browser_process_handler_bridge.h"
#include "render_process_handler_bridge.h"
#include "v8.h"
#include "app_bridge.h"
#include "scheme_bridge.h"
#include <stdlib.h>
#include <string.h>
*/
import "C"
import (
	"context"
	"errors"
	"fmt"
	"math"
	"reflect"

	//	_ "github.com/ianlancetaylor/cgosymbolizer"
	"github.com/rs/zerolog"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

type (
	cefApp struct {
		options  appOptions
		settings appSettings

		ptr      *C.cef_app_t
		mainArgs *C.cef_main_args_t

		browserProcessHandler *C.cef_browser_process_handler_t
		renderProcessHandler  *C.cef_render_process_handler_t

		clientMutex sync.Mutex
		clients     map[unsafe.Pointer]*client

		v8listenerLock sync.Mutex
		v8listeners    []V8Listener

		schemeHandlerFactory *C.cef_scheme_handler_factory_t

		contextInitializationFinished chan struct{}
		shutdownChannel               chan struct{}
		shutdownOnce                  sync.Once
		pumpWakeup                    chan struct{}

		// external message pump state, see message_pump.go
		pumpMtx       sync.Mutex
		pumpDue       time.Time // earliest requested work, zero if none
		pumpReady     atomic.Bool
		pumpActive    atomic.Bool
		pumpReentered atomic.Bool

		initialized bool
		closed      bool

		// cefMu guards work that finishes on a goroutine of its own and then
		// has to call into CEF (binding answers): it runs under the read lock
		// and only while cefDown is false. Close takes the write lock before
		// cef_shutdown, so nothing reaches a CEF that is gone.
		cefMu   sync.RWMutex
		cefDown bool

		workerWG sync.WaitGroup
	}

	CefApp interface {
		io.Closer

		Run(ctx context.Context) error
		NewClient(options ...ClientOption) Client

		AddV8Listener(listener V8Listener)
	}

	V8Listener interface {
		V8ContextCreated(CefBrowser, CefFrame)
		V8ContextReleased(CefBrowser, CefFrame)
	}

	v8listener struct {
		created, released func(CefBrowser, CefFrame)
	}
)

func (v v8listener) V8ContextCreated(c CefBrowser, frame CefFrame) {
	if v.created != nil {

		v.created(c, frame)
	}
}

func (v v8listener) V8ContextReleased(c CefBrowser, frame CefFrame) {
	if v.released != nil {
		v.released(c, frame)

	}
}

func V8ListenerFuncs(created, released func(b CefBrowser, f CefFrame)) V8Listener {
	return v8listener{
		created:  created,
		released: released,
	}
}

func (c *cefApp) AddV8Listener(listener V8Listener) {

	c.v8listenerLock.Lock()
	defer c.v8listenerLock.Unlock()

	c.v8listeners = append(c.v8listeners, listener)
}

var (
	CefAppAlreadyExistsErr = errors.New("cefApp already exists")
	mtx                    sync.Mutex
	instance               *cefApp
	appRunning             bool
)

func (c *cefApp) Close() error {
	mtx.Lock()
	if c.closed {
		mtx.Unlock()
		return nil
	}
	c.closed = true
	initialized := c.initialized
	mtx.Unlock()

	// Close all clients (and their browsers) while the message pump is still
	// running - browser teardown needs cef_do_message_loop_work to progress.
	// The clients stay registered until closeInternal removes them, because
	// the close callbacks (do_close, on_before_close) still need to resolve
	// their client.
	c.options.logger.Debug().Msg("closing clients")
	c.clientMutex.Lock()
	clients := make([]*client, 0, len(c.clients))
	for _, theClient := range c.clients {
		clients = append(clients, theClient)
	}
	c.clientMutex.Unlock()

	// with an external message pump, Close is expected to run on the thread
	// that called cef_initialize - pump the loop ourselves while waiting for
	// the browsers to finish closing
	var pump func()
	if initialized && c.options.pumpWorker != nil {
		pump = c.doMessageWork
	}

	for _, theClient := range clients {
		if closeErr := theClient.closeInternal(pump); closeErr != nil {
			c.options.logger.Warn().Err(closeErr).Msg("failed to close client")
		}
	}

	// Now stop the pump worker and shut CEF down. Workers the host still has
	// queued turn into no-ops from here on.
	c.shutdownOnce.Do(func() { close(c.shutdownChannel) })
	c.workerWG.Wait()
	c.pumpReady.Store(false)

	c.cefMu.Lock()
	c.cefDown = true
	c.cefMu.Unlock()

	if initialized {
		C.cef_shutdown()
	}

	mtx.Lock()
	c.freeNativeStructs()
	instance = nil
	appRunning = false
	mtx.Unlock()
	return nil
}

// freeNativeStructs releases the go-owned references on all calloc'd CEF
// structs. Must be called with mtx held.
func (c *cefApp) freeNativeStructs() {
	if nil != c.ptr {
		C.cAppFree(c.ptr)
		c.ptr = nil
	}

	if nil != c.browserProcessHandler {
		C.cBrowserProcessHandlerFree(c.browserProcessHandler)
		c.browserProcessHandler = nil
	}

	if nil != c.renderProcessHandler {
		C.cRenderProcessHandlerFree(c.renderProcessHandler)
		c.renderProcessHandler = nil
	}

	if nil != c.schemeHandlerFactory {
		C.cSchemeHandlerFactoryFree(c.schemeHandlerFactory)
		c.schemeHandlerFactory = nil
	}

	if nil != c.mainArgs {
		c.freeOperatingSystemSpecificMainArgs()
		C.free(unsafe.Pointer(c.mainArgs))
		c.mainArgs = nil
	}
}

var _ CefApp = (*cefApp)(nil)

// getInstance returns the current app instance or nil. Callback code paths
// must tolerate nil - a panic inside a cgo callback aborts the process.
func getInstance() *cefApp {
	mtx.Lock()
	defer mtx.Unlock()

	return instance
}

func Get() (CefApp, bool) {
	mtx.Lock()
	defer mtx.Unlock()

	if instance == nil {
		return nil, appRunning
	}
	return instance, appRunning
}

func New(opts ...Option) (CefApp, error) {
	mtx.Lock()
	defer mtx.Unlock()

	if instance != nil {
		return nil, CefAppAlreadyExistsErr
	}

	app := &cefApp{
		ptr: C.cAppCreate(),

		contextInitializationFinished: make(chan struct{}, 1),
		shutdownChannel:               make(chan struct{}),
		pumpWakeup:                    make(chan struct{}, 1),

		clients: map[unsafe.Pointer]*client{},

		browserProcessHandler: C.cBrowserProcessHandlerCreate(),
		renderProcessHandler:  C.cRenderProcessHandlerCreate(),

		options: appOptions{
			logger:                zerolog.Nop(),
			initializationTimeout: 500 * time.Millisecond,
			pumpWorkerInterval:    defaultPumpWorkerInterval,
		},
	}

	for _, opt := range opts {
		opt(app)
	}

	// Option is a plain func with no return value, so anything an option
	// rejected is parked in optionErrors and reported here - before the
	// bad value reaches cef_initialize, which only ever says "could not
	// initialize cef-app".
	if len(app.options.optionErrors) > 0 {
		err := errors.Join(app.options.optionErrors...)
		app.freeNativeStructs()
		return nil, err
	}

	app.options.logger.Debug().Str("ptr", fmt.Sprintf("0x%x", unsafe.Pointer(app.ptr))).Msg("cef-app created")
	app.options.logger.Debug().Str("ptr", fmt.Sprintf("0x%x", unsafe.Pointer(app.renderProcessHandler))).Msg("cef-render-process-handler created")
	app.options.logger.Debug().Str("ptr", fmt.Sprintf("0x%x", unsafe.Pointer(app.browserProcessHandler))).Msg("cef-browser-process-handler created")

	app.mainArgs = (*C.cef_main_args_t)(C.calloc(1, C.sizeof_cef_main_args_t))
	app.initializeOperatingSystemSpecificMainArgs()

	instance = app
	return app, nil
}

func (c *cefApp) Run(ctx context.Context) error {
	// note: the global mtx must not be held across cef_initialize or while
	// waiting for the context initialization - CEF callbacks arriving during
	// that window need getInstance().

	if err := c.verifyAndConfigureLibrary(); err != nil {
		return err
	}

	if c.settings.IgnoreCertificateErrors {
		C.cAppSetIgnoreCertificateErrors(C.int(1))
	}

	c.options.logger.Debug().Any("settings", c.settings).Msg("resulting settings computed")
	cSettings := C.cef_settings_t{}
	c.settings.apply(&cSettings)
	defer c.settings.clear(&cSettings)

	c.executeProcess()

	mtx.Lock()
	appRunning = true
	mtx.Unlock()

	if c.options.pumpWorker != nil {
		if c.settings.MultiThreadedMessageLoop {
			return errors.New("multi-threaded Message loop enabled")
		}
		c.options.logger.Debug().Msg("setting external_message_pump to 1 as we have a pump-worker")
		cSettings.external_message_pump = C.int(1)
		c.options.logger.Debug().Msg("setting multi_threaded_message_loop to 0")
		cSettings.multi_threaded_message_loop = C.int(0)

	}
	cSettings.windowless_rendering_enabled = C.int(1)

	initializationStarted := time.Now()
	// cef_initialize consumes a reference on the app struct.
	addRefBase(unsafe.Pointer(c.ptr))
	result := int(C.cef_initialize(c.mainArgs, &cSettings, c.ptr, nil))
	if result != 1 {
		err := errors.New("could not initialize cef-app")
		c.options.logger.Error().Err(err).Int("result", result).Send()
		return err
	}

	mtx.Lock()
	c.initialized = true
	mtx.Unlock()

	// The scheduler starts only now: before cef_initialize returned there is
	// nothing to pump, and if it fails there never will be. Requests CEF made
	// during initialization wait in pumpWakeup.
	var pumpTick <-chan time.Time
	if c.options.pumpWorker != nil {
		c.pumpReady.Store(true)
		c.workerWG.Add(1)
		go c.runPumpScheduler(ctx)

		// on_context_initialized is a UI-thread task. With an external pump
		// that thread is this one, and the host only starts running its
		// workers once Run returned - so pump here while waiting.
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()
		pumpTick = ticker.C
	}

	c.options.logger.Trace().Msg("cef-app initializing; waiting for completion...")

	initTimeout := time.NewTimer(c.options.initializationTimeout)
	defer initTimeout.Stop()
wait:
	for {
		select {
		case <-c.shutdownChannel:
			c.options.logger.Warn().Msg("shutdown while waiting for init")
			return nil
		case <-ctx.Done():
			c.options.logger.Warn().Msg("context cancelled while waiting for init")
			return ctx.Err()
		case <-c.contextInitializationFinished:
			break wait
		case <-pumpTick:
			c.doMessageWork()
		case <-initTimeout.C:
			err := errors.New("timed out while waiting for app-context")
			c.options.logger.Error().Err(err).Send()
			return err
		}
	}

	c.options.logger.Debug().Dur("duration", time.Since(initializationStarted)).Msg("cef-app initialized")

	// A custom scheme is announced in every process (see
	// goOnRegisterCustomSchemes), but only the browser process can serve it -
	// and only through a factory on the request context, which does not exist
	// before initialization. Browsers with their own context are covered in
	// NewRequestContext.
	if len(c.options.customSchemes) > 0 {
		if globalContext := C.cef_request_context_get_global_context(); globalContext != nil {
			c.registerSchemeHandlerFactories(globalContext)
			releaseBase(unsafe.Pointer(globalContext))
		} else {
			c.options.logger.Error().Msg("no global request context - custom schemes stay unserved")
		}
	}

	if c.settings.RemoteDebuggingPort != nil {
		c.options.logger.Debug().Msgf("Remote-Debugging-Port at: ws://localhost:%d", *c.settings.RemoteDebuggingPort)
	}

	return nil
}

// verifyLoadedLibraryVersion makes sure the loaded libcef matches the
// version this binary was compiled against - struct layouts differ between
// CEF versions and any regular call into a mismatched library crashes.
// cef_version_info is the only call that is safe across versions, so this
// must run before everything else that touches libcef.
func (c *cefApp) verifyLoadedLibraryVersion() error {
	if C.cAppRuntimeVersionMatches() != 0 {
		return nil
	}
	var buf [128]C.char
	C.cAppRuntimeCefVersion(&buf[0], C.size_t(len(buf)))
	err := fmt.Errorf("cef version mismatch: loaded libcef reports %s, binary was built against %s",
		C.GoString(&buf[0]), C.GoString(C.cAppCompiledCefVersion()))
	c.options.logger.Error().Err(err).Send()
	return err
}

// verifyAndConfigureLibrary runs the version check and then configures the
// client API version via cef_api_hash. CEF 151 refuses every versioned
// wrapper call with FATAL "called with invalid version -1" if the API
// version was never configured, so this MUST happen before the first call
// that passes a handler struct into libcef - in particular before
// cef_execute_process on BOTH the browser and the helper process path.
func (c *cefApp) verifyAndConfigureLibrary() error {
	if err := libcefRuntimeReady(); err != nil {
		c.options.logger.Error().Err(err).Send()
		return err
	}
	if err := c.verifyLoadedLibraryVersion(); err != nil {
		return err
	}

	expectedHash := C.GoString(C.cAppExpectedApiHash())
	actualHashPtr := C.cAppConfigureApiVersion()
	if actualHashPtr == nil {
		err := errors.New("cef api hash: libcef returned no hash for the configured api version")
		c.options.logger.Error().Str("expected", expectedHash).Err(err).Send()
		return err
	}
	actualHash := C.GoString(actualHashPtr)
	if expectedHash != actualHash {
		err := errors.New("cef api hash mismatch: libcef.dll does not match the headers this binary was built against")
		c.options.logger.Error().Str("expected", expectedHash).Str("actual", actualHash).Err(err).Send()
		return err
	}
	return nil
}

// withCEF runs fn unless CEF has been shut down (or is being shut down). For
// goroutines that finish late: a binding answer after Close is dropped, along
// with the references it would release - they died with CEF.
func (c *cefApp) withCEF(fn func()) {
	c.cefMu.RLock()
	defer c.cefMu.RUnlock()
	if !c.cefDown {
		fn()
	}
}

// doMessageWork performs one iteration of CEF's message loop plus the native
// event queue CEF does not service itself with an external pump: Win32 on
// Windows (MessagePumpExternal never calls PeekMessage), AppKit on macOS
// (gomium never enters -[NSApplication run]). It is the Worker the host
// receives and must run on the thread that called cef_initialize.
//
// A worker that reaches the host before cef_initialize returned or after
// Close began shutting CEF down does nothing: hosts queue workers (a channel,
// a main-thread dispatch), and one left over from the last tick would
// otherwise call into a CEF that is gone.
//
// CEF must not be pumped reentrantly. A nested call - a CEF callback that ends
// up in the host's loop again - is dropped and answered with an immediate
// reschedule once the outer pass is done, like cefclient's reentrancy
// detection.
func (c *cefApp) doMessageWork() {
	if !c.pumpReady.Load() {
		return
	}
	if !c.pumpActive.CompareAndSwap(false, true) {
		c.pumpReentered.Store(true)
		return
	}

	C.cef_do_message_loop_work()
	more := pumpPlatformEvents()
	c.pumpActive.Store(false)

	if c.pumpReentered.Swap(false) || more {
		c.scheduleWork(0)
	}
}

func (c *cefApp) executeProcess() {
	c.options.logger.Trace().Msg("executing cef-process...")

	// cef_execute_process consumes a reference on the app struct.
	addRefBase(unsafe.Pointer(c.ptr))
	exitCode := int(C.cef_execute_process(c.mainArgs, c.ptr, nil))
	if exitCode == -1 {
		c.options.logger.Debug().Msg("cef-execute-process returned with -1 for the main browser process")
		return
	}
	c.options.logger.Error().Int("exit-code", exitCode).Msg("cef-execute-process returned")
	os.Exit(exitCode)
}

func (c *cefApp) onWebKitInitialized() {
	c.options.logger.Debug().Msg("web-kit initialized")
}

func (c *cefApp) v8ContextCreated(c2 cefBrowser, frame cefFrame) {

	c.v8listenerLock.Lock()
	defer c.v8listenerLock.Unlock()

	for _, listener := range c.v8listeners {
		listener.V8ContextCreated(c2, frame)
	}
}

func (c *cefApp) v8ContextReleased(c2 cefBrowser, frame cefFrame) {

	c.v8listenerLock.Lock()
	defer c.v8listenerLock.Unlock()

	for _, listener := range c.v8listeners {
		listener.V8ContextReleased(c2, frame)
	}
}

func castValue(val any) (Value, error) {
	switch v := val.(type) {
	case nil:
		return &NullValue{value{ptr: C.cef_v8_value_create_null()}}, nil
	case time.Time:
		cTime := C.cef_time_t{}
		// cef_time_from_doublet expects seconds since epoch
		C.cef_time_from_doublet(C.double(float64(v.UnixMilli())/1e3), &cTime)
		var baseTime C.cef_basetime_t
		C.cef_time_to_basetime(&cTime, &baseTime)
		return &DateValue{value{ptr: C.cef_v8_value_create_date(baseTime)}}, nil
	case float32:
		return &DoubleValue{value{ptr: C.cef_v8_value_create_double(C.double(v))}}, nil
	case float64:
		return &DoubleValue{value{ptr: C.cef_v8_value_create_double(C.double(v))}}, nil
	case int:
		if v > math.MaxInt32 || v < math.MinInt32 {
			return &DoubleValue{value{ptr: C.cef_v8_value_create_double(C.double(v))}}, nil
		}
		return &IntValue{value{ptr: C.cef_v8_value_create_int(C.int32_t(v))}}, nil
	case uint:
		if v > math.MaxUint32 {
			return &DoubleValue{value{ptr: C.cef_v8_value_create_double(C.double(v))}}, nil
		}
		return &UintValue{value{ptr: C.cef_v8_value_create_uint(C.uint32_t(v))}}, nil
	case bool:
		intValue := C.int(0)
		if v {
			intValue = C.int(1)
		}
		return &BooleanValue{value{ptr: C.cef_v8_value_create_bool(intValue)}}, nil
	case string:
		var charStr *C.char = C.CString(v)
		var cefStr C.cef_string_t
		defer C.free(unsafe.Pointer(charStr))
		C.cef_string_from_utf8(charStr, C.strlen(charStr), &cefStr)
		defer C.cef_string_clear(&cefStr)
		return &StringValue{value{ptr: C.cef_v8_value_create_string(&cefStr)}}, nil
	default:
		reflected := reflect.ValueOf(val)

		if reflected.Kind() == reflect.Slice {
			v8Ptr := C.cef_v8_value_create_array(C.int(reflected.Len()))

			for idx := range reflected.Len() {
				item := reflected.Index(idx).Interface()
				if subValue, subErr := castValue(item); subErr != nil {
					releaseBase(unsafe.Pointer(v8Ptr))
					return nil, subErr
				} else {
					// set_value_byindex consumes the sub-value reference
					C.js_set_value_by_index(v8Ptr, C.int(idx), subValue.native())
				}
			}
			return &ArrayValue{value: value{ptr: v8Ptr}, values: map[int]Value{}}, nil
		}

		if reflected.Kind() == reflect.Map && reflected.Type().Key().Kind() == reflect.String {
			obj := CreateObject()
			for _, key := range reflected.MapKeys() {
				item := reflected.MapIndex(key).Interface()
				subValue, subErr := castValue(item)
				if subErr != nil {
					_ = obj.Close()
					return nil, subErr
				}
				// Hand the child's own reference to the object, like the
				// array branch does with set_value_byindex. SetValueForKey
				// would add a reference and keep the wrapper in properties -
				// and nothing closes those once the root is resolved into a
				// promise, so every child of a map result leaked.
				cKey := C.CString(key.String())
				ok := C.setValueByKey(obj.ptr, cKey, subValue.native()) != 0
				C.free(unsafe.Pointer(cKey))
				if !ok {
					_ = obj.Close()
					return nil, fmt.Errorf("could not assign value to key %q", key.String())
				}
			}
			return obj, nil
		}

		return nil, fmt.Errorf("unknown value type: %T", val)
	}
}

// callArgument turns the i-th JS argument into the value for parameter i of
// fn. null and undefined have no Go value of their own; reflect.Call would
// panic on the invalid Value, so they become the zero value of the parameter
// ("" for a string, nil for any). For a variadic fn every argument from the
// last parameter on takes the element type.
func callArgument(fn reflect.Type, i int, arg *Exported) reflect.Value {
	native := arg.Native()
	if native != nil {
		return reflect.ValueOf(native)
	}
	if fn.IsVariadic() && i >= fn.NumIn()-1 {
		return reflect.Zero(fn.In(fn.NumIn() - 1).Elem())
	}
	return reflect.Zero(fn.In(i))
}
