# gomium on Linux

> **Status: offscreen, verified on linux/amd64 in CI.** The `refdebug (linux)` job runs
> every harness scenario on Ubuntu under `xvfb-run` against CEF 156 — bindings, custom
> schemes, popups, cookies, DevTools, both render handlers — with the refdebug ledger.
> linux/arm64 compiles but has not run. Offscreen rendering is the only mode:
> `WithWindow` returns `windowed browsers are only implemented on windows and macos`.

### No zygote

On Linux Chromium normally spawns renderers and the GPU process from a *zygote*, a
pre-initialised helper that forks. With gomium the zygote is the Go executable, and the
Go runtime always runs several threads; Chromium refuses to fork a multi-threaded
process. The child launches fail (`GPU process launch failed: error_code=1002`) and
the browser exits with `GPU process isn't usable`. gomium therefore always appends
`--no-zygote` on Linux, and the browser process starts every child by fork+exec. That
needs `--no-sandbox`, which gomium sets anyway (`no_sandbox = 1`).

---

## Requirements

- gcc or clang with cgo, and `libdl` (linked via `#cgo linux LDFLAGS: -ldl`).
- x86-64 or arm64 — `internal/cefshim/cef_shim.S` has an ELF arm64 block alongside the amd64 one.
- The shared libraries Chromium expects at runtime (X11/GTK stack). CEF's distribution
  does not carry them. On Ubuntu 24.04 the CI job installs `xvfb libnss3 libgbm1
  libasound2t64 libatk1.0-0t64 libatk-bridge2.0-0t64 libcups2t64 libxkbcommon0
  libxcomposite1 libxdamage1 libxrandr2 libpango-1.0-0 libcairo2`
  (`.github/workflows/ci.yml`); without a display server run under `xvfb-run -a`.

## Getting the runtime

```sh
go run ./tools/cefdl        # linux64 / linuxarm64 by GOARCH, into ./Release
```

`cefdl` flattens the distribution's `Release/` and `Resources/` into a single
`./Release`, which is the layout CEF expects next to an executable.

---

## Integration

```go
func main() {
    runtime.LockOSThread()

    // Runtime binding, before anything else - helper processes are fresh
    // instances of this same executable and go through here too.
    if err := gomium.LoadLibcef(filepath.Join("Release", "libcef.so")); err != nil {
        log.Fatal(err)
    }

    if gomium.IsSubprocess() {
        if err := gomium.ExecuteSubprocess(opts()...); err != nil {
            log.Fatal(err)
        }
        return
    }

    app, err := gomium.New(opts()...)
    if err != nil {
        log.Fatal(err)
    }
    defer app.Close()

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    if err := app.Run(ctx); err != nil {
        log.Fatal(err)
    }

    client := app.NewClient(
        gomium.WithRenderHandler(gomium.NewImageRenderHandler(
            image.NewRGBA(image.Rect(0, 0, 1024, 768)),
            func(frame image.Image) { /* your frame sink */ },
        )),
    )
    // ... client.NewBrowser(gomium.WithInitialUrl(...))
}
```

### Library loading

`LoadLibcef` resolves the path to an absolute one and calls `dlopen` with
`RTLD_LAZY | RTLD_LOCAL`. `RTLD_LOCAL` keeps everything outside the package from
resolving against libcef — every entry point goes through the shim table — and the
absolute path keeps `LD_LIBRARY_PATH` from binding a different CEF build than the one
shipped next to the executable.

Every symbol is resolved before any of them is bound, so a partial failure leaves the
table fully unbound and names the missing exports rather than crashing later.

### Message loop

`WithMultiThreadedMessageLoop` is Windows-only in gomium today, even though CEF
supports it on Linux as well. The external pump is therefore the only option:

```go
gomium.WithMessagePumpWorker(func(delay int, worker gomium.Worker) {
    go func() {
        if delay > 0 {
            time.Sleep(time.Duration(delay) * time.Millisecond)
        }
        select {
        case pumpChannel <- worker:   // drained by the main goroutine
        default:
        }
    }()
}),
```

`cef_do_message_loop_work` must run on the thread that called `cef_initialize`, so the
main goroutine — the one holding `runtime.LockOSThread` — executes what arrives on the
channel.

### Offscreen only

`internal/cef/browser_window_other.go` reports an error for any windowed browser, so every Linux
host goes through `WithRenderHandler`.

A native-window Linux path would mean filling in the Linux `cef_window_info_t`
(`include/internal/cef_types_linux.h`). It carries `window_name`, `bounds`,
`runtime_style` and `windowless_rendering_enabled` like the others, but no window
style, and its `parent_window` / `window` handles are X11 window ids (`unsigned long`)
rather than an `HWND` or an `NSView`.

---

## Shipping an application

There is no bundle format. Ship the executable with the CEF files beside it — including
the two license notices `cefdl` unpacks (`CEF_LICENSE.txt`, `CREDITS.html`), which are a
redistribution obligation; see [`THIRD_PARTY_NOTICES.md`](../THIRD_PARTY_NOTICES.md):

```
myapp
CEF_LICENSE.txt
CREDITS.html
libcef.so
icudtl.dat
v8_context_snapshot.bin
chrome_100_percent.pak
chrome_200_percent.pak
resources.pak
libEGL.so
libGLESv2.so
libvk_swiftshader.so
libvulkan.so.1
vk_swiftshader_icd.json
locales/
  en-US.pak, de.pak, …
```

Per CEF's own `README.txt` for this distribution:

| component | files | required? |
| --- | --- | --- |
| CEF core library | `libcef.so` | **yes** |
| Unicode data | `icudtl.dat` | **yes** |
| V8 snapshot | `v8_context_snapshot.bin` | **yes** |
| Non-localized resources | `chrome_100_percent.pak`, `chrome_200_percent.pak`, `resources.pak` | optional — web components render incorrectly without them |
| Localized resources | `locales/*.pak` | optional — the locale is picked from `LANGUAGE`, `LC_ALL`, `LC_MESSAGES`, `LANG`, defaulting to `en-US` |
| ANGLE | `libEGL.so`, `libGLESv2.so` | optional — canvas / 3D CSS / WebGL |
| SwANGLE | `libvk_swiftshader.so`, `libvulkan.so.1`, `vk_swiftshader_icd.json` | optional — software rendering when the GPU is unavailable |

`chrome-sandbox` is in the distribution but not in CEF's redistribution list, and
gomium sets `no_sandbox = 1` unconditionally — so it is not needed. Using it would mean
installing it setuid root, which the current settings do not ask for.

Note the Linux distribution has **no** `chrome_elf` equivalent; crash reporting is
inside `libcef.so`.

### Somewhere other than next to the executable

```go
gomium.WithResourcesDirPath("/opt/myapp/cef"),
gomium.WithLocalesDirPath("/opt/myapp/cef/locales"),
```

Both must be absolute. `LoadLibcef` takes the `libcef.so` path separately, so the
library does not have to sit with the resources.

`WithBrowserSubprocessPath` points CEF at a different executable for its helpers. Leave
it unset unless the helpers really are a separate binary — empty means "re-execute this
one", which is what `IsSubprocess()` / `ExecuteSubprocess()` are built for.

---

## Known gaps

- **Never run.** Everything below "it compiles" is untested.
- **No windowed browser.** `cef_window_info_t` is unfilled for Linux.
- **No `WithMultiThreadedMessageLoop`.** CEF supports it here; gomium does not expose
  it outside Windows.
- **Headless is unexplored.** Whether OSR works without an X display, and which
  `--ozone-platform` / `--disable-gpu` switches are needed, has not been tried.

---

## Troubleshooting

**`gomium: loading /…/libcef.so: … cannot open shared object file`**
Usually a *dependency* of `libcef.so`, not `libcef.so` itself. `ldd Release/libcef.so`
names what is missing; those are distribution packages, not CEF files.

**`gomium: … does not export: …`**
The `.so` is a different CEF version than the headers in `include/`. `cefdl/cef.version` is
the pin; `go run ./tools/cefdl` fetches the matching build.

**`Opening in existing browser session`**
Two CEF applications sharing the default user-data directory; one holds the Chromium
process singleton. Give each its own `WithCachePath`.
