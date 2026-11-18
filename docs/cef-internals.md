# CEF internals: why the odd bits are there

Findings from porting the binding from CEF 3.3202 (Chromium 62) to CEF 151.3.14
(Chromium 151), kept because each one explains a line of code that looks removable
and is not. If you are about to delete a command-line switch, relax a lifetime rule,
or "simplify" one of the workarounds below — read the matching entry first.

The command-line switches live in `on_before_command_line_processing_trampoline`
in [`app_bridge.c`](../internal/cef/app_bridge.c). They are three *independent* fixes for three
*different* crashes, which is easy to miss because they sit in the same function.

---

## 1. `--disable-stack-profiler` — cgo frames break stack unwinding

Chromium's `StackSamplingProfiler` samples threads with `RtlVirtualUnwind`. Go frames
carry no Windows-conformant unwind info, so unwinding across them faults inside
`ntdll!RtlpUnwindPrologue` — an access violation roughly 90 s after start, **in every
process that hosts the Go runtime**.

Fix: the `disable-stack-profiler` switch plus disabling the `ThreadGroupProfiler` and
`SamplingProfilerReporting` features. All three are needed; the switch alone does not
cover the feature-driven samplers.

Symbolized against `libcef.dll.pdb` from the `release_symbols` package (5.3 GB).
Stack: `base::StackSampler::WalkStack` → `NativeUnwinderWin::TryUnwind`.

## 2. `--disable-gpu-watchdog` — the watchdog kills a healthy GPU process

Under the message pump this embedding uses, the GPU main thread never drains a Windows
message queue. The watchdog reads that as hung and kills the GPU process with
`RESULT_CODE_HUNG` (exit code 2) every ~15 s. After a few kills the browser process
gives up with `FATAL "GPU process isn't usable. Goodbye."`, which surfaces on the Go
side as a goroutine dump — Go catches the break exception.

**Diagnostic pattern:** child processes respawning on a ~15 s cadence. Any
"Network service crashed" messages are collateral damage from the same cascade, not a
separate fault. Chasing them wastes time.

Trade-off: with the watchdog off, a genuine GPU hang goes undetected. Acceptable for an
embedded webview.

## 3. `ImmersiveReadAnything` disabled — soft-navigation null deref

Chrome's "Read Anything" feature registers a page-load-metrics observer whose
`OnSoftNavigation` handler calls `tabs::TabInterface::GetFromContents`. Alloy-style
(OSR) browsers have no tab model attached to their `WebContents`, so the lookup
dereferences null and the **browser** process dies on the first soft navigation —
SPA-style `history.pushState` after a user interaction.

Reproduced by `GOMIUM_SOFTNAV_TEST` in `cmd/harness`. Unrelated to the profiler crash
above, despite sitting three lines away from it.

---

## 4. Do not release `command_line` in the CLP callback

The `command_line` parameter of `on_before_command_line_processing` wraps the
process-global command line, and CEF manages that CppToC wrapper itself. Releasing it
is a double release: it corrupts the browser process and crashes the GPU children.
Verified empirically with an isolated C repro.

This is the **exception**, not the rule. Every other refptr parameter arriving in a
callback carries a reference the receiver must release.

## 5. `cef_window_info_t.size` must be set

Since CEF 126 the struct carries a `size` field (`sizeof(cef_window_info_t)`).
Leave it zero and `cef_browser_host_create_browser` fails immediately, with no
diagnostic worth the name.

## 6. Offscreen rendering requires `CEF_RUNTIME_STYLE_ALLOY`

Chrome-style is the default since the Alloy runtime was removed in CEF 128, and it does
**not** support windowless rendering. OSR browsers must set
`runtime_style = CEF_RUNTIME_STYLE_ALLOY` in the window info — see
[`browser_window_windows.go`](../internal/cef/browser_window_windows.go).

## 7. `Request.SetUrl` in `OnBeforeResourceLoad` is a redirect, not a rewrite

Under CEF 62 this silently rewrote the request. Since the Network Service (CEF 75) it
issues a real redirect, and Chromium forbids redirecting to `data:` URLs
(`ERR_UNSAFE_REDIRECT`). A `data:` replacement therefore never reaches the renderer,
and the only symptom is a resource that quietly fails to load.

Workarounds: redirect to a local `127.0.0.1` HTTP server that serves the content, or
use a real `cef_resource_handler_t` — gomium exposes that as `ResourceInterceptor`.

## 8. API-version hash check

`internal/cef/app.go` compares the header hash against the loaded DLL's `cef_api_hash()` on `Run()`
and aborts with a clear message on mismatch — the alternative is a silent ABI mismatch
that corrupts memory. Currently unpinned
(`CEF_API_VERSION_EXPERIMENTAL`); see [upgrading-cef.md](upgrading-cef.md) for why
pinning it is worth doing before the next upgrade.

## 9. Behaviour that changed and cannot be restored

- **Extensions (`WithExtension`, guards, `SameOrigin`) were removed.** They were built on
  `cef_register_extension`, which Chromium deprecated in CEF 154 (v8::Extensions are compiled
  on every context creation and not part of the snapshot). No backport; the replacement, if
  ever needed, is per-context binding in `OnContextCreated` with `CefV8Value::CreateFunction`.
- **HTTP auth goes to the client handler, never to CEF's own dialog.** Chromium's login prompt
  answers challenges of Alloy/OSR browsers unless `disable-chrome-login-prompt` is set; then
  `GetAuthCredentials` on the client is called (`libcef/browser/net_service/login_delegate.cc`,
  CEF 8078 branch). The switch is process-wide, so a missing handler now cancels the request
  instead of showing a dialog. Verified on CEF 156.0.27 with the Fyne auth dialog.
- **`WithSingleProcess` is a no-op.** Single-process mode no longer exists. Extension
  functions run in the render subprocess, which is the same Go executable re-executed
  through the `cef_execute_process` path.
- **`load_string` was removed.** `CefFrame.LoadString` now builds a base64 `data:` URL;
  the origin parameter is ignored.
- **Frame identifiers are strings** (CEF 122), not `int64`. This reaches the public API:
  `CefFrame.Identifier() string`, `GetFrameByIdentifier(string)`. `SendProcessMessage`
  runs over frames; the browser-level variant routes through the main frame.
