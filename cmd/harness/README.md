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

The harness uses its own profile, `%TEMP%/gomium-test-profile`, so it never collides with
another CEF application's process singleton.
