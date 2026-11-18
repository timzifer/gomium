// Package gomium is a Go binding for the Chromium Embedded Framework.
//
// This package is the entire public API. The cgo binding itself lives in
// internal/cef and the runtime shim in internal/cefshim; everything either of
// them exports is re-exported here, so there is exactly one import path to
// learn and nothing importable that is not meant to be used.
//
// The re-export follows one rule: what you implement is declared here in
// full, what you receive is an alias. So the handler contracts - [LoadHandler],
// [RequestHandler], [ContextMenuHandler] and the rest, along with callback
// signatures like [PopupHandler] - carry their method sets and doc comments,
// while [Browser], [CefFrame], [Cookie] and the other types the library hands
// back are aliases. That asymmetry exists because go doc does not follow an
// alias: for a type you have to implement, an alias would hide the very
// method set you need.
//
// # Runtime binding
//
// Nothing links against libcef at build time. Call [LoadLibcef] with the path
// to the distribution before anything else - it resolves every entry point and
// binds a table of trampolines. A missing or mismatched CEF distribution is
// therefore a startup error naming the symbol, not a link error. The
// distribution is fetched by tools/cefdl into ./Release and pinned by
// cefdl/cef.version, which must match the vendored headers in include/.
//
// # Process model
//
// CEF runs as several processes, and it creates them by re-executing your own
// binary with a --type=... argument. Every one of those helper processes runs
// your main() from the top. Guard it:
//
//	if gomium.IsSubprocess() {
//		os.Exit(gomium.ExecuteSubprocess())
//	}
//
// Miss this and the helpers run your application logic instead of CEF's, which
// usually looks like the browser never starting rather than like a crash.
// [LoadLibcef] must run in the helpers too, before [ExecuteSubprocess].
//
// # Threading
//
// CEF is thread-affine and most of it belongs to the UI thread - the thread
// that called [CefApp.Run]. On macOS that must additionally be the process's
// main thread, with an NSApplication implementing CefAppProtocol installed by
// [InstallMacApplication].
//
// Handler callbacks arrive on CEF threads, not on yours. The ones that decide
// something - CanDownload, PopupHandler, OnJsDialog, RunContextMenu,
// InterceptResource, CertificateErrorHandler - run synchronously on the UI or
// IO thread and must not block: no waiting on another goroutine, no calling
// back into blocking parts of this API. Use [CefCurrentlyOn] to assert where
// you are.
//
// # Where to read next
//
// docs/api.md is the full reference - and the place to look for method-level
// documentation of the aliased types, which go doc cannot show. docs/windows.md,
// docs/osx.md and
// docs/linux.md cover the per-platform obligations (message loop, bundling,
// which runtime files to ship), and docs/cef-internals.md records why several
// non-obvious workarounds in the binding must stay.
package gomium
