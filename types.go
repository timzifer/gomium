package gomium

import (
	cef "github.com/timzifer/gomium/internal/cef"
)

// The types you receive, aliased so that composite literals, type assertions
// and interface implementations written against these names keep working - a
// defined type would break all three, and for the types carrying methods
// (Request, ExceptionError, the Value structs) it would drop the methods
// outright.
//
// The cost is documentation: `go doc gomium.Browser` prints the alias and its
// doc comment but not the method set, because go doc does not follow an alias
// - to an internal package or any other. The types a caller has to implement
// are spelled out in contracts.go for exactly that reason; these are the ones
// the library hands you, and their methods are documented in docs/api.md and
// in the declarations in internal/cef.

type ArrayValue = cef.ArrayValue

type BooleanValue = cef.BooleanValue

type Browser = cef.Browser

type BrowserNotInitialized = cef.BrowserNotInitialized

type BrowserOption = cef.BrowserOption

type CefApp = cef.CefApp

type CefBrowser = cef.CefBrowser

type CefFrame = cef.CefFrame

type CefProcessId = cef.CefProcessId

type CefTask = cef.CefTask

type CefTaskRunner = cef.CefTaskRunner

type CefThreadId = cef.CefThreadId

type CefV8Exception = cef.CefV8Exception

type CertificateAction = cef.CertificateAction

// CertificateErrorCallback finishes an async certificate decision.
// Continue must be called exactly once; safe from any goroutine.
type CertificateErrorCallback = cef.CertificateErrorCallback

type Client = cef.Client

type ClientOption = cef.ClientOption

type (
	DevTools      = cef.DevTools
	DevToolsError = cef.DevToolsError
)

var (
	ErrDevToolsDetached = cef.ErrDevToolsDetached
	ErrDevToolsClosed   = cef.ErrDevToolsClosed
)

// ContextMenuCallback finishes a context menu the handler took over.
// Exactly one of Continue or Cancel must be called; safe from any
// goroutine. Continue executes the given command as if the user picked
// it from Chromium's own menu (e.g. MenuIdCopy).
type ContextMenuCallback = cef.ContextMenuCallback

type ContextMenuEditFlags = cef.ContextMenuEditFlags

// ContextMenuItem mirrors one entry of the default Chromium menu.
type ContextMenuItem = cef.ContextMenuItem

type ContextMenuMediaType = cef.ContextMenuMediaType

// ContextMenuParams is a snapshot of the right-click context. It stays
// valid after the callback returned - it carries no CEF references.
type ContextMenuParams = cef.ContextMenuParams

type ContextMenuTypeFlags = cef.ContextMenuTypeFlags

// Cookie mirrors cef_cookie_t. Creation and LastAccess are informational
// and populated by the store when reading; they are ignored on SetCookie.
// A nil Expires makes a session cookie.
type Cookie = cef.Cookie

// CookieManager reads and writes the browser cookie store. All calls
// block until the store confirmed the operation, so they must not be
// invoked from CEF callback goroutines that the UI thread waits on.
// Persistence across restarts requires WithCachePath on the app.
type CookieManager = cef.CookieManager

type CookiePriority = cef.CookiePriority

type CookieSameSite = cef.CookieSameSite

type CursorType = cef.CursorType

type DateValue = cef.DateValue

type DirtyArea = cef.DirtyArea

type DispatchWith = cef.DispatchWith

type Dispatcher = cef.Dispatcher

type DoubleValue = cef.DoubleValue

// DownloadControl cancels, pauses or resumes a running download. Only
// valid during the OnDownloadUpdated call it was handed to.
type DownloadControl = cef.DownloadControl

// DownloadItem is a snapshot of a download's state. It stays valid after
// the callback returned - it carries no CEF references.
type DownloadItem = cef.DownloadItem

type Event = cef.Event

type ExceptionError = cef.ExceptionError

type Exported = cef.Exported

type ExportedFunction = cef.ExportedFunction

type Exporter = cef.Exporter

// FindResult reports the state of a Browser.Find search. Multiple
// updates may arrive per search; FinalUpdate marks the last one.
type FindResult = cef.FindResult

// FocusSource tells where a focus change originated.
type FocusSource = cef.FocusSource

// KeyboardEvent is the Go view of a key event seen by a KeyboardHandler.
type KeyboardEvent = cef.KeyboardEvent

// FileDialogMode is the base mode of a file dialog.
type FileDialogMode = cef.FileDialogMode

// AuthRequest describes an HTTP or proxy auth challenge.
type AuthRequest = cef.AuthRequest

// AuthCallback finishes an auth challenge taken over by an AuthHandler.
type AuthCallback = cef.AuthCallback

// AuthAction tells CEF how an AuthHandler treated a challenge.
type AuthAction = cef.AuthAction

// AuthHandler answers HTTP and proxy auth challenges with an own login form.
type AuthHandler = cef.AuthHandler

// FrameHandler observes frame lifecycle. Frames are valid only during a call.
type FrameHandler = cef.FrameHandler

// ScreenInfo describes the screen a render handler's view lives on (DIP).
type ScreenInfo = cef.ScreenInfo

type FunctionValue = cef.FunctionValue

type IntValue = cef.IntValue

type JsDialogAction = cef.JsDialogAction

// JsDialogCallback finishes a dialog the handler took over. Continue must
// be called exactly once; userInput is only meaningful for prompt
// dialogs. Safe to call from any goroutine.
type JsDialogCallback = cef.JsDialogCallback

type JsDialogType = cef.JsDialogType

type KeyEventModifier = cef.KeyEventModifier

type KeyEventModifiers = cef.KeyEventModifiers

type KeyEventType = cef.KeyEventType

type LogSeverity = cef.LogSeverity

type MenuItemType = cef.MenuItemType

type MouseButton = cef.MouseButton

type MouseButtonDirection = cef.MouseButtonDirection

type NullValue = cef.NullValue

type ObjectValue = cef.ObjectValue

type Option = cef.Option

// BindingCall describes one call of a binding from page JavaScript. See
// BindingGuard.
type BindingCall = cef.BindingCall

// BindingGuard decides whether a binding call may run. It is set per browser
// with WithBindingGuard. false rejects the call.
type BindingGuard = cef.BindingGuard

// CustomScheme is a scheme the embedder serves itself, registered with
// WithCustomScheme and answered from a ResourceInterceptor.
type CustomScheme = cef.CustomScheme

// PdfPrintOptions keeps Chromium's defaults for everything not listed.
type PdfPrintOptions = cef.PdfPrintOptions

// PopupRequest carries the information about a popup the page wants to open
// (window.open, target="_blank", shift-click, ...).
type PopupRequest = cef.PopupRequest

// PopupView is the offscreen view of a popup (see WithPopupView).
type PopupView = cef.PopupView

// PopupViewFactory builds the PopupView for a popup (see WithPopupView).
type PopupViewFactory = cef.PopupViewFactory

type ProcessMessage = cef.ProcessMessage

// Request wraps a cef_request_t. It is only valid during the callback it was
// handed to (for OnBeforeResourceLoad with ResourceLoadContinueAsync: until
// Continue or Cancel is called). After that every method degrades to a safe
// zero value instead of touching freed memory.
type Request = cef.Request

// RequestContext is an isolated browsing profile: its own cookies,
// localStorage, cache and credentials. Browsers created with the same
// context share storage; browsers with different contexts are fully
// separated from each other and from the global context.
type RequestContext = cef.RequestContext

type RequestContextOption = cef.RequestContextOption

// ResourceContent is a complete in-memory response served for an
// intercepted resource request. MimeType is the media type, e.g. "text/html";
// a charset parameter is allowed and passed on separately.
type ResourceContent = cef.ResourceContent

// ResourceContentFn produces the response content. It runs on its own
// goroutine and may block (downloads, rendering). Returning nil aborts
// the request with ERR_ABORTED.
type ResourceContentFn = cef.ResourceContentFn

type ResourceLoadAsyncResult = cef.ResourceLoadAsyncResult

// SchemeOptions is the bitmask describing how Chromium treats a custom
// scheme (see the Scheme* constants and WithCustomScheme).
type SchemeOptions = cef.SchemeOptions

type ResourceLoadResult = cef.ResourceLoadResult

type ResultCallback = cef.ResultCallback

type StringValue = cef.StringValue

type UintValue = cef.UintValue

type UndefinedValue = cef.UndefinedValue

type V8Context = cef.V8Context

type Value = cef.Value

type ValueType = cef.ValueType

// WindowOpenDisposition describes where a popup wants to open.
type WindowOpenDisposition = cef.WindowOpenDisposition

type Worker = cef.Worker
