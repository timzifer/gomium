# cmd/harness — manual test harness

**Not an example.** This is the driver gomium is tested with against a real CEF
distribution: one program, a pile of scenarios behind environment variables, and logging
turned all the way up. To learn how to embed gomium, start from
[`examples/osr`](../../examples/osr) and [`examples/window`](../../examples/window).

CI cannot run any of this — it needs the CEF runtime in `./Release`
(`go run ./tools/cefdl`). Run it from the repository root:

```sh
go build -o build/harness.exe ./cmd/harness
GOMIUM_BINDING_TEST=1 GOMIUM_RUN_SECONDS=15 ./build/harness.exe
```

On macOS CEF starts only out of an application bundle. `-agent` keeps the offscreen
runs out of the Dock; bundle without it for `GOMIUM_WINDOW_TEST`:

```sh
go build -o build/harness ./cmd/harness
go run ./tools/cefbundle -bin build/harness -cef Release -out build/Harness.app -agent
GOMIUM_BINDING_TEST=1 GOMIUM_RUN_SECONDS=15 build/Harness.app/Contents/MacOS/Harness
```

Building with `-race` and running the scenarios below is the race-detector check; a `GORACE="log_path=build/race"` keeps one report file per process (helpers
included).

## Scenarios

Without any variable, the harness renders its default start page offscreen with the
external message pump for 20 s and writes every paint to `shots/<µs>.jpg` (create
`shots/` first).

| variable | what it does |
| --- | --- |
| `GOMIUM_RUN_SECONDS=n` | run for n seconds instead of 20 |
| `GOMIUM_TEST_URL=url` | initial URL |
| `GOMIUM_DEBUG_PORT=n` | remote debugging port (default 6696) |
| `GOMIUM_MT_LOOP=1` | Windows: CEF's multi-threaded message loop instead of the external pump |
| `GOMIUM_WINDOW_TEST=1` | native window instead of offscreen; file-drop page on `127.0.0.1:18776` shows dropped names (JS) and paths (Go) side by side |
| `GOMIUM_BINDING_TEST=1` | `WithBinding`: result, error and an iframe the guard rejects (`BINDING_*` log lines) |
| `GOMIUM_BINDING_POPUP_TEST=1` | binding in a `WithPopupView` popup, rendered to `shots/popup-*.jpg`, auto-closed after 5 s |
| `GOMIUM_POPUP_NO_GUARD=1` | with the above: popup without its own guard, falls back to `WithDefaultBindingGuard` |
| `GOMIUM_POPUP_ABORT=1` | with the above: closes the opener while CEF builds the popup, so CEF aborts it |
| `GOMIUM_V8_LISTENER_TEST=1` | `WithV8Listener` injects a script through its frame (`V8_LISTENER_INJECTED` log lines) |
| `GOMIUM_POPUP_TEST=1` | trusted click on `target=_blank`, popup routed into the opener |
| `GOMIUM_DOWNLOAD_TEST=1` | download handler against a local attachment response |
| `GOMIUM_COOKIE_TEST=1` | cookie manager: set, read, delete |
| `GOMIUM_CONTEXT_TEST=1` | two incognito request contexts: cookie isolation, a browser in context A |
| `GOMIUM_DIALOG_TEST=1` | alert/confirm/prompt answered from Go, context menu intercepted |
| `GOMIUM_NAV_TEST=1` | navigation sequence: `location.href`, link click, `window.open` |
| `GOMIUM_SOFTNAV_TEST=1` | SPA soft navigations — regression test for the CEF 151 profiler crash |
| `GOMIUM_INTERCEPT_TEST=1` | request handler: `.pdf` requests served by the resource interceptor |
| `GOMIUM_SCHEME_TEST=1` | custom scheme: the start page itself is served through `WithCustomScheme` |
| `GOMIUM_HANDLER_TEST=1` | focus, keyboard, file dialog, frame and auth handlers that only log |
| `GOMIUM_AUTH_TEST=1` | with the above: auth challenges answered with fixed credentials |
| `GOMIUM_RESIZE_STRESS=1` | resizes the offscreen surface continuously, like a dragged parent window |
| `GOMIUM_DEVTOOLS_TEST=1` or `-devtools` | `WithDevTools` probe: attach, evaluate, load events, cross-site navigation, renderer crash |

## Reference ledger (`refdebug`)

Built with `-tags refdebug`, gomium checks the reference counts of every struct it
allocates: a reference operation on a freed struct panics with the stack that freed it,
and `Close` fails with the allocation stack of each struct still alive after
`cef_shutdown` - the harness then exits with code 3. `refdebug.sh` runs the offscreen
scenarios against such a build and fails on any finding; CI does the same on Windows.

```sh
go build -tags refdebug -o build/harness-refdebug.exe ./cmd/harness
cmd/harness/refdebug.sh build/harness-refdebug.exe            # all scenarios
cmd/harness/refdebug.sh build/harness-refdebug.exe POPUP SCHEME
```

The ledger covers gomium's own structs (handlers, clients, callbacks), not the CEF
objects gomium receives - a missing release of a browser or frame stays invisible.

### Debug runtime

CEF's debug build covers the other side: its DCHECKs turn an over-release into a crash
and check at `cef_shutdown` that no wrapper object is left - a leaked browser shows up
as `DCHECK failed: all_.empty()` in `browser_context.cc`. `cefdl -debug` unpacks it
(Debug/ of the 350 MiB standard archive) in place of the release runtime; `refdebug.sh`
scans CEF's log lines (`:FATAL:`, `Check failed`) along with the ledger findings. The
debug libcef is slow, so give the scenarios more time:

```sh
go run ./tools/cefdl -debug -dest Release     # a plain go run ./tools/cefdl switches back
GOMIUM_RUN_SECONDS=30 GOMIUM_SCENARIO_GRACE=120 cmd/harness/refdebug.sh build/harness-refdebug.exe
```

After a failed DCHECK the process can hang in its exit path for good - on Windows a
plain `taskkill /F` does not end it, only one from an elevated shell does.
`refdebug.sh` abandons it at the deadline and goes on, so the leftover process has to
be killed as administrator (`taskkill /F /IM harness-refdebug.exe`). CI runs this nightly
(`.github/workflows/cef-debug.yml`, also startable by hand).

The harness uses its own profile, `%TEMP%/gomium-test-profile`, so it never collides with
another CEF application's process singleton.
