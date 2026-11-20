# Changelog

All notable changes to gomium are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow
[Semantic Versioning](https://semver.org/spec/v2.0.0.html) — while gomium is at `v0.x`,
a minor version may break the API, and so, in the past, did one patch version
(see 0.2.2).

0.5.0 is the first public release. The versions below it were tagged in the private
repository gomium grew up in; their tags were not carried over, so they cannot be
fetched — they are listed for the record of what changed when.

## [Unreleased]

### Added

- CI job `refdebug (macos)`: the offscreen harness scenarios on `macos-latest` (arm64), out of
  a `cefbundle` bundle, on every pull request (#40).
- `WithMockKeychain(bool)` (macOS): Chromium encrypts cookies and stored credentials with
  a fixed key instead of the "Chromium Safe Storage" keychain item. An ad-hoc signed
  bundle is a new code identity on every build, and each one raised a keychain prompt
  per process; until it was answered the network service blocked, page loads stalled and
  shutdown ended in Chromium's teardown watchdog (exit 2). `cmd/harness` and both
  examples turn it on. It leaves the profile on disk unprotected — see `docs/osx.md`.

### Fixed

- macOS: every gomium host asked the user to allow notifications, once per bundle
  identifier - so every rebuilt test bundle again. Chrome connects the browser process
  and the "Helper (Alerts)" process to the notification system at startup; gomium routes
  no web notifications to the host, so the `NativeNotifications` and
  `SystemNotifications` features are now off on macOS.
- macOS: every helper process (GPU, utility, renderer) showed up in the Dock.
  `InstallMacApplication`, which runs in the helpers too, forced the regular activation
  policy over their `LSUIElement`. The policy is now left alone there;
  `ActivateMacApplication` promotes a process only when AppKit started it without a Dock
  entry (a binary outside a bundle).

## [0.7.0] - 2026-11-20

### Added

- `NewFrameRenderHandler`: an OSR render handler that paints into one surface it owns
  and lends it, with the dirty rectangles, to the callback. Nothing is allocated per
  paint; `NewImageRenderHandler` copies the full view for every callback (a 16x16 paint
  on a 1080p view: 1.9 ms and 8.3 MB vs. 0.1 µs and 0 B). The harness runs it as the
  `FRAME` scenario (#23).
- CI runs the unit tests with `-race` (Linux), and a nightly workflow (`race.yml`) runs
  the harness scenarios as a `-race` build; `refdebug.sh` fails a scenario on
  `WARNING: DATA RACE`. The first run found no race (#25).

### Changed

- **Breaking:** logging uses `log/slog` instead of zerolog (#22). `WithLogger`,
  `SetDebugLogger` and `SetMemoryLogger` take a `*slog.Logger`; `nil` switches logging
  off. The trace output of the reference counting and the noisiest app steps logs at the
  new `LevelTrace` (below `slog.LevelDebug`). zerolog is no longer a dependency. The
  reference-count trace builds its attributes only when the logger takes `LevelTrace` —
  before, every `add_ref`/`release` formatted the pointer even with logging off.
- `ScreenInfoRenderHandler`, `FocusHandler`, `KeyboardHandler`, `FileDialogHandler`,
  `AuthHandler` and `FrameHandler` are declared in the root package instead of aliased,
  so `go doc` shows their methods; `BindingGuard` and `PopupViewFactory` are func types
  of their own. Existing implementations and func literals keep compiling (#21).

### Removed

- **Breaking:** internal plumbing that had leaked into the root API (#19):
  - `From(unsafe.Pointer) *Request` — a raw-pointer constructor nothing outside the
    binding could use correctly.
  - `WithSingleProcess` — a no-op since CEF 151.
  - The V8 value wrappers `Value`, `ObjectValue`, `ArrayValue`, `BooleanValue`,
    `DateValue`, `DoubleValue`, `FunctionValue`, `IntValue`, `NullValue`,
    `StringValue`, `UintValue`, `UndefinedValue`, and `CreateObject`, `Exporter`,
    `NewExporter`. No public signature took or returned them; results reach callers
    as `Exported`.
  - Process messages: `RegisterMessageIdentifier`, `ProcessMessageFactory`,
    `ProcessMessage`, `Dispatcher`, `DispatchWith`, and `SendProcessMessage` on
    `CefBrowser` / `CefFrame`. A caller could not build a `ProcessMessage`, so none of
    it was usable outside the binding.
- The `github.com/davecgh/go-spew` dependency; only the harness used it (#27).

### Fixed

- A binding call still running when `CefApp.Close` reaches `cef_shutdown` no longer
  keeps its `cef_frame_t` reference alive across the shutdown. `Close` releases the
  references of such calls before `cef_shutdown`; their answers are dropped as before
  (#18).
- Without `-tags refdebug`, an `add_ref` on a struct gomium had already freed brought it
  back into the reference map, so the next `release` freed it a second time. It now
  logs a warning and leaves the pointer alone (#17).
- The final `release` of a gomium struct no longer clears the resource, cookie and PDF
  side registries while holding the global reference-count lock (#17).
- `SetMemoryLogger` raced with the reference-count callbacks (#17).
- Linux: browsers start. Chromium's zygote is the Go executable and refuses to fork a
  multi-threaded process, so every GPU and renderer launch failed and the browser
  exited with "GPU process isn't usable"; gomium now appends `--no-zygote` on Linux.
  A `refdebug (linux)` CI job runs the harness scenarios on Ubuntu (#24).
- DevTools: events without a handler are no longer queued, the queue of undelivered
  events is capped at 4096 (further events are dropped with a warning), and a drained
  queue releases its backing array (#26).

## [0.6.0] - 2026-11-19

### Added

- `-tags refdebug`: a reference ledger for every struct gomium allocates. A reference
  operation on a freed struct panics with the stack that freed it, and `CefApp.Close`
  fails with the allocation stacks of structs still alive after `cef_shutdown`. CI
  runs the harness scenarios with it on Windows (`cmd/harness/refdebug.sh`).
- `tools/refcheck`: a lint for the reference counting at the cgo boundary, run in CI.
  Callback parameters must be released or handed off, and a C call that passes a
  reference on to CEF needs one of its own; bridge parameters say which with
  `REF_CONSUMED` / `REF_BORROWED`.
- `cefdl.Options.Debug` / `cefdl -debug`: unpacks CEF's debug runtime (Debug/ of the
  standard archive) instead of the release one. A nightly workflow runs the harness
  scenarios against it, where CEF's DCHECKs catch over-releases and wrapper objects
  still alive at `cef_shutdown` - the CEF-side counterpart of the refdebug ledger.
- CEF review workflow: an agent review of pull requests touching `internal/cef`, with
  the prompt of the unsafe audit (`.github/claude-review/cef-audit.md`). Optional and
  advisory; it runs once `ANTHROPIC_API_KEY` or `CLAUDE_CODE_OAUTH_TOKEN` is set.
- `WithV8Listener`: registers a V8 context listener in both processes. Before, a host
  could not reach the render process at all — `AddV8Listener` on the browser-process
  app never fires, since V8 contexts exist only in the renderer.

### Changed

- Binding guards and bound functions run on a goroutine of their own instead of the
  browser UI thread, and calls from the page may run concurrently — a bound function
  must be safe for that. `BindingCall.FrameURL` and `MainFrame` are still taken when
  the call arrives.
- With the external message pump, `CefApp.Close` called off the thread that
  initialized CEF returns an error instead of pumping and shutting CEF down on the
  wrong thread.

### Fixed

- V8 listeners run synchronously on the renderer main thread, before any page script
  of the new context, instead of on a goroutine where CEF rejected every call on the
  frame and browser they receive.
- A bound function that waits for the UI thread itself (cookie manager, DevTools
  `Call`, `PrintToPdf`) no longer blocks until its timeout, and a slow one no longer
  freezes every browser while it runs.
- `WithPopupView` popups no longer leak clients: a popup CEF aborts after
  `on_before_popup` allowed it (`on_before_popup_aborted`) now closes the client
  prepared for it, as does closing the opener's client while the popup is still being
  built. Every popup - also one routed into the opener, cancelled, or opened
  natively - leaked one reference to the opener's `cef_client_t`, which kept it from
  ever being freed.
- The factory of `WithCustomScheme` outlived `cef_shutdown`: CEF keeps the factories
  of the global request context, so `Close` now clears them first.
- A frame whose new V8 context is created before the old one is released keeps the new
  one; before, the late release dropped it and evaluations in that frame failed with
  "no v8 context".
- Binding calls the browser never answered are rejected when their context is
  released, instead of holding the promise and the context until the renderer exits.
- `NewBrowser` no longer returns a timeout error for a browser that arrived while the
  timeout fired; the browser was registered all the same and lived on unseen.

## [0.5.0] - 2026-11-18

### Added

- `LICENSE` (BSD 3-Clause) and `THIRD_PARTY_NOTICES.md`.
- `examples/osr` and `examples/window`: minimal, copyable hosts with the external
  message pump.
- `cefdl` unpacks the distribution's license notices next to the runtime:
  `CEF_LICENSE.txt` and `CREDITS.html` (Chromium and its third-party components). Both
  must ship with an application; `tools/cefbundle` copies them into
  `Contents/Resources/`. An existing runtime directory gets them with `cefdl -force`.

### Changed

- `docs/api.md` is in English now, like the rest of the documentation.
- The manual test harness moved from `cmd/main` to `cmd/harness`, with its scenarios
  documented in `cmd/harness/README.md`.
- **Breaking for the CEF upgrade workflow:** the pinned CEF version moved from
  `cef.version` in the repository root to `cefdl/cef.version`. It is embedded into the
  `cefdl` package and available as `cefdl.PinnedVersion()`.

### Fixed

- `go run github.com/timzifer/gomium/tools/cefdl` works from a module that depends on
  gomium; it no longer reads `./cef.version` from the working directory.
- Reference counting and memory safety:
  - a UI-thread call that timed out (`ZoomLevel`, `IsAudioMuted`,
    `ExecuteDevToolsMethod`, DevTools `Call`) no longer runs later on memory its
    caller had already freed;
  - binding calls no longer leak the JavaScript `this` value and every argument;
    binding results that are maps no longer leak their children;
  - `Equals` on `CefBrowser`, `V8Context` and `CefTaskRunner` (and the value
    wrappers) no longer releases the argument once too often;
  - a request handler that keeps the `OnBeforeResourceLoad` callback but answers
    synchronously no longer reaches a released callback, and one that called
    `Continue` before returning no longer releases it twice;
  - `Browser.Close` holds its own reference while it closes;
  - a failed custom-scheme registration or `ObjectValue.SetValueForKey` no longer
    releases its argument twice; a process message to a browser without a main frame
    is released instead of leaked.

## Pre-public history

### 0.4.3 - 2026-11-18

#### Fixed

- With an external message pump, blocking calls made on the UI thread (`NewBrowser`,
  `Client.Close`, cookie access, `PrintToPdf`, DevTools `Call`) pump the message loop
  themselves instead of waiting for a callback only that thread could deliver.
  `NewBrowser` no longer runs into its 5 s timeout in a Fyne host.

### 0.4.2 - 2026-11-18

#### Fixed

- External message pump (`WithMessagePumpWorker`) on Windows drains the Win32 message
  queue of the `cef_initialize` thread. Windowed browsers no longer go "not responding"
  and close cleanly; offscreen hosts service Chromium's hidden windows.
- The pump honours `delay_ms` of `OnScheduleMessagePumpWork` and pumps at least every
  1000/30 ms (was 250 ms), as cefclient does.
- Workers queued by the host before `Run` finished or after `Close` began are no-ops;
  reentrant pumping is detected and rescheduled.

#### Changed

- Default of `WithMessagePumpWorkerInterval` is 1000/30 ms. Workers and `Close` must run
  on the thread that called `cef_initialize`.

### 0.4.1 - 2026-11-12

#### Added

- Test harness: DevTools probe that crashes the renderer, popup dialog test page for
  bindings in popups. No library changes.

### 0.4.0 - 2026-11-12

#### Added

- In-process DevTools protocol per browser (`WithDevTools`): method calls, events and
  attach, without a remote debugging port.

#### Removed

- **Breaking:** the raw CEF value helpers left the public API — `NewValue`, `FreeValue`,
  `NewListValue`, `FreeListValue`, `BinaryValueFromBuffer` and the `CefValue`,
  `CefListValue`, `CefBinaryValue` types.

### 0.3.1 - 2026-11-11

#### Changed

- `WithBinding` accepts variadic Go functions; surplus JavaScript arguments arrive as the
  final slice parameter.

### 0.3.0 - 2026-11-11

#### Added

- `WithBinding(path, fn)`: Go functions exposed to page JavaScript per frame, returning
  promises. Signatures are checked when the binding is created.
- `WithBindingGuard` and `WithDefaultBindingGuard`: every binding call is checked
  against its frame; without a guard, calls are rejected. `SameOrigin` guard helper.
- `WithPopupView`: allowed popups render offscreen with a client of their own instead of
  opening a native window.

### 0.2.2 - 2026-11-11

#### Added

- Optional handler escape hatches: focus, keyboard, file dialog, frames, auth, popups and
  screen info.
- Migration guide CEF 151.3.14 → 156.0.27 (`docs/migration-151-to-156.md`).

#### Changed

- CEF 156.0.27 is the target and the minimum version; the API version is pinned to 15600.

#### Fixed

- HTTP auth challenges reach the client's `AuthHandler` instead of Chrome's own login
  prompt. Without an auth handler the request is cancelled.

#### Removed

- **Breaking:** the extension API (`CefRegisterExtension`, deprecated since CEF 154):
  `WithExtension`, `Extension`, `WithExtensionCallGuard`,
  `WithDefaultExtensionCallGuard`, `DenyExtensionCalls`, `SameOrigin`. Bindings
  (0.3.0) replace it.

### 0.2.1 - 2026-11-10

#### Fixed

- `""`, `0` and `false` survive the trip between renderer and browser process in
  extension calls instead of arriving as `null`.

### 0.2.0 - 2026-11-09

#### Added

- Guards for extension calls: `ExtensionCall`, `ExtensionCallGuard`,
  `WithExtensionCallGuard`, `WithDefaultExtensionCallGuard`, `DenyExtensionCalls`,
  `SameOrigin`.

### 0.1.0 - 2026-09-14

First tagged release.

#### Added

- Go bindings on the CEF C API, bound at runtime (`LoadLibcef`) — no link-time dependency
  on libcef.
- Offscreen rendering and native windows (`WithWindow`) on Windows and macOS, with
  native file-drop paths.
- CEF 151, macOS/arm64 support (arm64 trampolines, `dlopen` loader, AppKit pump, `.app`
  bundling via `tools/cefbundle`).
- Custom schemes (`WithCustomScheme`), runtime download of the CEF distribution
  (`FetchRuntime`, package `cefdl`, `tools/cefdl`).
- CI on Ubuntu, Windows and macOS.

[Unreleased]: https://github.com/timzifer/gomium/compare/v0.7.0...HEAD
[0.7.0]: https://github.com/timzifer/gomium/compare/v0.6.0...v0.7.0
[0.6.0]: https://github.com/timzifer/gomium/compare/v0.5.0...v0.6.0
[0.5.0]: https://github.com/timzifer/gomium/releases/tag/v0.5.0
