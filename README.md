# gomium

Go bindings for the [Chromium Embedded Framework](https://bitbucket.org/chromiumembedded/cef)
(CEF), built directly on the C API (`libcef`) — no C++ wrapper, no `libcef_dll_wrapper`.

Two modes of operation:

- **Offscreen rendering (OSR)** — CEF paints into a buffer, gomium hands you an
  `image.Image`. This is the mode to use when embedding a browser into Fyne, Ebiten,
  a texture, or a screenshot pipeline.
- **Native window** — CEF creates and owns a real top-level window (`WithWindow`,
  Windows and macOS; on macOS the browser can also be put into an existing `NSView`
  with `WithParentView`).

Current target: **CEF 156.0.27** (`156.0.27+ga10dae3+chromium-156.0.8078.88`), pinned in
[`cefdl/cef.version`](cefdl/cef.version). The API version is pinned to 15600 (`-DCEF_API_VERSION` in
`internal/cef/cgoflags.go`), so later CEF releases do not change the binding silently.

**Minimum version: CEF 156.0.27.** Do not run against earlier CEF releases:

- CEF 151.3.14 crashes on a click on `<input type="file">` in an offscreen
  (`WithRenderHandler`) browser. `CefWebContentsViewOSR::GetNativeView()` returns a null
  view, and Chromium's `FileSelectHelper` dereferences it before the client's
  `CefDialogHandler` is consulted. Fixed in later 151 patches and in 156.
- CEF 154 and later deprecated `CefRegisterExtension` (v8::Extension, compiled on every
  context creation). gomium no longer uses it; the feature was removed.

Verified on 156.0.27 (Windows), live in `cmd/harness` and in a Fyne host application:

- offscreen and windowed browsers start, render and shut down cleanly — with the
  multi-threaded message loop and with the external message pump
  (`WithMessagePumpWorker`)
- JavaScript evaluation returns results
- popups, downloads, custom schemes, certificate errors, keyboard input
- the file dialog reaches `FileDialogHandler`, HTTP auth challenges reach `AuthHandler`
- PDF printing writes a file; the in-process DevTools protocol answers calls

Not verified yet: the focus and frame handlers, and anything on macOS or Linux against
156.0.27 — the platform headers (`cef_types_linux.h` / `cef_types_mac.h`) are identical
to the 156.0.27 distributions, but no build or run was done there.

> **Status: pre-release (`v0.x`).** The API is not stable yet — expect breaking changes
> between minor versions. Releases are tagged (`v0.5.0` onwards), so consumers can pin a
> version instead of a commit. Windows and macOS/arm64 have both started a real browser;
> Linux has never run one. See
> [Road to a publishable package](#road-to-a-publishable-package) for the concrete gaps.

## Requirements

- Go 1.23.2 or newer
- A cgo toolchain (on Windows: MinGW-w64 / TDM-GCC — the assembly shim is GAS syntax;
  on macOS: the Xcode command line tools, since the application bootstrap is
  Objective-C)
- x86-64 or arm64. The trampoline shim (`internal/cefshim/cef_shim.S`) carries a block per
  architecture; everything else is architecture agnostic.
- ~400 MB of disk for the CEF distribution and the download cache

## Getting the CEF runtime

### In your own module

`go get` fetches the Go code, never the CEF binaries. Fetch the runtime that matches the
gomium version in your `go.mod` with the downloader, from your module's root:

```sh
go get github.com/timzifer/gomium@latest
go run github.com/timzifer/gomium/tools/cefdl -dest ./Release
```

The pinned CEF version is embedded into `cefdl` (`cefdl.PinnedVersion()`), so this works
outside a gomium checkout and always matches the headers your gomium version was built
against. `-version` and `-version-file` override it. To fetch from code instead — e.g.
on first start, next to the executable — call `gomium.FetchRuntime(ctx, dir)`.

### In a gomium checkout

The CEF binary distribution (~160 MB of DLLs, `.pak` resources and locales) is **not**
checked into the repository. Fetch it with the bundled downloader:

```sh
go generate ./...          # runs tools/cefshim and tools/cefdl
```

or directly:

```sh
go run ./tools/cefdl                 # fetch the version pinned in cefdl/cef.version
go run ./tools/cefdl -list           # show available builds for this platform
go run ./tools/cefdl -version 151    # newest build on the 151 branch
go run ./tools/cefdl -headers        # also refresh include/ (overwrites local patches)
```

`cefdl` resolves a build from the official CEF build index
(`cef-builds.spotifycdn.com`), downloads the `minimal` archive, verifies its SHA-1 and
unpacks `Release/` and `Resources/` into `./Release`. It caches the archive under
`os.UserCacheDir()/gomium-cef` and writes `Release/.cef-version`, so repeat runs are a
no-op. `./Release` is git-ignored.

The platform is picked from `GOOS`/`GOARCH` (`-platform` overrides it), so an arm64
machine gets `windowsarm64` / `linuxarm64` / `macosarm64`. The macOS distribution ships
`Chromium Embedded Framework.framework` instead of a single library — symlinks inside
the bundle are recreated as symlinks, and `LoadLibcef` takes either the bundle or the
binary inside it.

The version in `cefdl/cef.version` must match the vendored headers in `include/` — the loader
compares both before the first real call.

## Quick start

The shape of every gomium host — complete and runnable in
[`examples/osr`](examples/osr) and [`examples/window`](examples/window):

```go
// work carries message-pump workers to the main thread.
var work = make(chan gomium.Worker, 16)

func opts() []gomium.Option {
    return []gomium.Option{
        gomium.WithCachePath(filepath.Join(os.TempDir(), "myapp-cef")),
        // The message loop: gomium decides when CEF needs a pump, the host
        // runs it on the cef_initialize thread (the main loop below).
        gomium.WithMessagePumpWorker(func(_ int, w gomium.Worker) {
            select {
            case work <- w:
            default:
            }
        }),
    }
}

func main() {
    // CEF requires all its work on the thread that called cef_initialize.
    runtime.LockOSThread()

    // libcef is bound at runtime. This must run before everything else —
    // including ExecuteSubprocess, because CEF helper processes are fresh
    // instances of this same executable. Windows: libcef.dll, Linux:
    // libcef.so, macOS: the "Chromium Embedded Framework.framework" bundle.
    if err := gomium.LoadLibcef(filepath.Join("Release", "libcef.dll")); err != nil {
        log.Fatal(err)
    }

    // macOS only, before both branches: gomium.InstallMacApplication().

    // Branch off helper processes (--type=renderer, --type=gpu, ...) — before
    // flag.Parse, which would reject their Chromium switches.
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
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    if err := app.Run(ctx); err != nil { // cef_initialize
        log.Fatal(err)
    }

    client := app.NewClient(
        gomium.WithRenderHandler(gomium.NewImageRenderHandler(
            image.NewRGBA(image.Rect(0, 0, 1024, 768)),
            func(frame image.Image) { /* your frame sink */ },
        )),
    )
    // NewBrowser waits for CEF, which needs the pump: call it off the main
    // thread while main keeps pumping.
    go client.NewBrowser(gomium.WithInitialUrl("https://example.org"))

    for {
        select {
        case w := <-work:
            w()
        case <-ctx.Done():
            // Close pumps while the browsers shut down — on this thread.
            _ = app.Close()
            return
        }
    }
}
```

On Windows, `WithMultiThreadedMessageLoop(true)` is the alternative to the pump: CEF
runs its own loop thread and the host has nothing to drive. macOS has no such option.

The full API reference lives in **[docs/api.md](docs/api.md)**; the per-platform
obligations live in **[docs/](docs/)**. GoDoc comments on the symbols themselves are
the authoritative details.

## How the binding is put together

**Runtime binding instead of link-time binding.** Nothing links against `libcef` at
build time. `tools/cefshim` generates, from the single list in
`internal/cefshim/cef_shim_symbols.txt`, the three files that make up
`internal/cefshim`:

| output | role |
| --- | --- |
| `internal/cefshim/cef_shim.S` | one tail-jump trampoline per export, jumping through a pointer table — no prologue, so it is signature-neutral |
| `internal/cefshim/cef_shim.c` | the pointer table the trampolines jump through |
| `internal/cefshim/cef_shim_symbols.go` | the name list `LoadLibcef` resolves |

They live in their own package because none of them needs a CEF header: the
trampolines carry C linkage names, so the binding resolves them at link time
through the vendored CEF declarations it already includes.

The trampolines come in one block per architecture, selected by the preprocessor:
amd64 jumps straight through the RIP-relative slot, arm64 loads the slot through the
GOT into `x16` — the intra-procedure-call scratch register a veneer is allowed to
clobber — and branches. Mach-O's underscore prefix is applied by a macro, so the same
file serves ELF, PE-COFF and Mach-O. Since no argument register is touched anywhere,
the trampoline stays signature-neutral on both architectures.

`LoadLibcef` resolves every symbol first and only then binds the table, so a partial
failure leaves it fully unbound rather than half-pointing into a freed library. On
Windows the DLL is loaded with `LOAD_WITH_ALTERED_SEARCH_PATH`, so its dependencies
(`chrome_elf.dll`, `vulkan-1.dll`, `dxcompiler.dll`, …) resolve next to `libcef.dll`
and the distribution stays relocatable; on macOS and Linux the equivalent is `dlopen`
on an absolute path with `RTLD_LOCAL`, which keeps the search paths (`DYLD_*_PATH`,
`LD_LIBRARY_PATH`) from binding a different CEF build than the one that was shipped.

The payoff: a missing or mismatched CEF distribution is a startup error with a symbol
name in it, not a link error — and the same binary can be shipped without the runtime
present.

**Process model.** CEF spawns helper processes by re-executing your binary with
`--type=…`. `IsSubprocess()` / `ExecuteSubprocess()` are what keep those helpers from
running your application logic.

**Handlers.** The C callback structs are bridged per handler in `*_bridge.c`/`.h` with a
Go side next to them: life span, load, display, render, request, resource, download,
drag, find, JS dialog, context menu, cookie manager, request context, V8 / process
messages. Each bridge calls the `//export`ed Go functions of its sibling file, which is
why a handler's `.go` and its `.c` always live in the same package.

**One package for the binding, one for the API.** cgo gives every package its own
distinct set of C types, so splitting the binding by theme would mean converting
`*C.cef_browser_t` and friends through `unsafe.Pointer` at every seam — exactly where
the type checker is worth the most. The binding therefore stays a single package in
`internal/cef`, and the module root is a facade of aliases and forwards over it. The
practical effect is that `internal/` holds everything you should not depend on, and
what remains importable is the API.

The facade splits along one line: **what you implement is declared in full, what you
receive is an alias.** `go doc` does not follow a type alias, so an aliased handler
interface would hide the method set you have to implement — the 15 handler contracts
and 4 callback signatures are therefore spelled out in `contracts.go`, with paired
compile-time assertions keeping them identical to the binding. Everything the library
hands back stays an alias in `types.go`, because those types carry methods that a
redeclared type would drop, or appear in signatures where redeclaring would hand
callers a slice they cannot convert.

## Layout

| path | contents |
| --- | --- |
| `*.go` (root) | the public API: type aliases and forwards over `internal/cef` |
| `internal/cef` | the cgo binding itself — `*_bridge.c/h` and their Go siblings |
| `internal/cefshim` | generated runtime-binding shim: trampolines, pointer table, loader |
| `include/` | vendored CEF headers, patched |
| `tools/cefdl` | CEF distribution downloader (`go:generate`) |
| `tools/cefshim` | shim generator (`go:generate`) |
| `tools/cefbundle` | macOS `.app` bundle builder |
| `tools/cefdiff` | C-API diff scripts for a CEF version upgrade |
| `examples/` | minimal, copyable hosts: `osr`, `window` |
| `cmd/harness` | manual test harness, driven by `GOMIUM_*` env vars |
| `Release/` | CEF runtime — downloaded, git-ignored |
| `docs/` | API reference, per-platform guides, CEF internals and upgrade procedure |

### Examples and the test harness

- [`examples/osr`](examples/osr) — minimal offscreen host: load a page, write the frame
  to a PNG.
- [`examples/window`](examples/window) — minimal native window (Windows, macOS), ends
  when the window is closed.

Both use the external message pump driven from the main thread, and both are meant to
be copied. [`cmd/harness`](cmd/harness) is the opposite: the manual test driver with
twenty-odd `GOMIUM_*` scenarios, documented in its own README.

## Platform support

| platform | build | runtime |
| --- | --- | --- |
| windows/amd64 | yes | yes |
| windows/arm64 | trampolines are ELF/Mach-O only — PE-COFF arm64 stops at an `#error` | no |
| linux/amd64, linux/arm64 | yes | `LoadLibcef` (`dlopen`) and windowless rendering are implemented; not run on real hardware yet |
| darwin/arm64 | yes | yes — offscreen and windowed, out of a bundle built by `tools/cefbundle` |
| darwin/amd64 | yes | same code path as arm64, but only arm64 has been run |

Linux is written but unverified: it compiles and the generated assembly assembles, but
no browser has been started on it.

### Platform guides

Everything platform-specific — the toolchain, the runtime layout, the `main()`
skeleton, packaging and the failure modes worth recognising — lives in one guide per
platform:

| guide | covers |
| --- | --- |
| **[docs/windows.md](docs/windows.md)** | module handle, multi-threaded vs external message loop, DLL search path, which files from `Release\` to ship |
| **[docs/osx.md](docs/osx.md)** | `CefAppProtocol`, the AppKit event queue, `.app` bundling with `tools/cefbundle`, codesigning |
| **[docs/linux.md](docs/linux.md)** | `dlopen` loading, offscreen-only status, which files to ship — **unverified** |

The three obligations that differ most: Windows wants a module handle and can run
CEF's own loop thread; macOS needs an `NSApplication` implementing `CefAppProtocol`, an
`.app` bundle with helper bundles, and its own event queue drained on every pump tick;
Linux is offscreen-only and has never been run.

---

## Road to a publishable package

What still stands between this repository and something a third party could
`go get` and depend on. Ordered roughly by what blocks whom.

### Blocking `go get`

- [x] **Fix the module path.** `go.mod` declares `github.com/timzifer/gomium`,
      matching where the repository actually lives, so `go get` resolves it.
- [x] **Add a LICENSE.** BSD 3-Clause ([`LICENSE`](LICENSE)).
      [`THIRD_PARTY_NOTICES.md`](THIRD_PARTY_NOTICES.md) covers the vendored CEF
      headers and the binaries `cefdl` downloads; `cefdl` unpacks CEF's
      `CEF_LICENSE.txt` and Chromium's `CREDITS.html` next to the runtime, and
      `cefbundle` copies them into the `.app`.
- [x] **Tag a version.** Releases are tagged `v0.x` (`v0.5.0` onwards) — enough to
      signal instability while still being resolvable.
- [x] **Document the runtime dependency at install time.** `go get` cannot fetch the
      CEF distribution; [In your own module](#in-your-own-module) documents
      `go run github.com/timzifer/gomium/tools/cefdl`, which embeds the pinned version
      and so runs from any module, not only from this repository root.

### Correctness and confidence

- [ ] **Real tests.** `internal/cef/app_test.go` holds a single smoke test — it runs
      without a CEF distribution, since `New()` only reaches the local bridge functions
      in `app_bridge.c`, never a trampoline. `internal/cefshim` covers the shim slot
      contract, the per-architecture trampolines and generator drift, and
      `facade_test.go` covers API completeness and import resolution per platform;
      value conversion, process messages and handler lifetimes have no coverage.
- [x] **CI.** [`.github/workflows/ci.yml`](.github/workflows/ci.yml) builds, vets and
      tests on ubuntu, windows and macOS, gates `gofmt`, and sweeps file selection
      across four `GOOS` values. Nothing links against libcef and the headers are
      vendored, so no CEF distribution is fetched. Every job asserts that cgo is
      actually on before it builds — without that, a runner missing a C compiler
      produces a green run that compiled none of the binding.

      Trampoline coverage: linux/amd64 (ELF), windows/amd64 (PE-COFF) and macos-latest,
      which is arm64 (Mach-O), plus a `gcc-aarch64-linux-gnu` cross build for
      ELF/arm64. Only windows/arm64 is left, and that is the `#error` case. What CI
      still cannot do is run against a real CEF distribution — that stays `cmd/harness`
      by hand.
- [x] **Version consistency check.** `cefdl/cef.version` and the vendored `include/` headers
      must agree; `TestPinnedVersionMatchesHeaders` in `cefdl` checks the version string
      and the numeric defines of `include/cef_version.h`, in CI.
- [ ] **Race detector and leak checks.** The binding hands Go pointers across cgo
      callbacks and does its own CEF refcounting — a `-race` run and a refcount/leak
      test are the two things most likely to find real bugs here.
- [ ] **Audit `unsafe` and refcounting.** Especially the registries
      (`internal/cef/cef_process_message_registry.go`, `internal/cef/javascript_result_callback_registry.go`) and
      every handler that outlives a browser.

### Platform coverage

- [x] **Linux and macOS runtime loading.** `LoadLibcef` binds through `dlopen`/`dlsym`
      on both; the stub is left for platforms CEF publishes no builds for.
- [x] **arm64 trampolines.** `internal/cefshim/cef_shim.S` carries an amd64 and an arm64 block, the
      latter for Mach-O and ELF. PE-COFF arm64 (`windowsarm64`) still stops at an
      `#error` — it needs its own relocation syntax.
- [x] **`WithWindow` on macOS.** `internal/cef/browser_window_darwin.go` fills the macOS
      `cef_window_info_t`, with `WithParentView` for hosts that bring their own
      `NSView`. Everything except Windows and macOS still reports an error.
- [ ] **Run it on real hardware.** macOS/arm64 is done — offscreen and windowed both
      start, load, render and shut down cleanly out of a `tools/cefbundle` bundle.
      Linux has still never been run; its assembly assembles and nothing more.
- [x] **macOS packaging.** `tools/cefbundle` builds and ad-hoc signs the `.app` with
      the framework and the five helper bundles.

### Documentation

- [x] **Per-platform integration guides.** [`docs/windows.md`](docs/windows.md),
      [`docs/osx.md`](docs/osx.md), [`docs/linux.md`](docs/linux.md) — toolchain,
      `main()` skeleton, message loop, packaging and troubleshooting each.
- [x] **Package-level GoDoc.** `doc.go` covers the runtime binding, the process model
      (including the `IsSubprocess` guard every helper process runs into) and the
      threading rules.
- [x] **Translate `docs/api.md` to English.**
- [x] **Runnable `examples/`.** [`examples/osr`](examples/osr) and
      [`examples/window`](examples/window), verified on Windows against 156.0.27.
- [x] **Documented CEF upgrade procedure.** [`docs/upgrading-cef.md`](docs/upgrading-cef.md)
      — pinning a version, regenerating the C-API headers, the three diff scripts in
      `tools/cefdiff`, and why `CEF_API_VERSION` wants pinning before the next upgrade.
      [`docs/cef-internals.md`](docs/cef-internals.md) keeps the crash findings that
      explain the workarounds in `internal/cef/app_bridge.c`.
- [x] **CHANGELOG.** [`CHANGELOG.md`](CHANGELOG.md), Keep a Changelog style, with the
      pre-public history; maintained with every release.

### Repository hygiene

- [x] **Dissolved `upgrade-cef151/`** (~400 files of port notes). The durable half is
      now [`docs/cef-internals.md`](docs/cef-internals.md) and
      [`docs/upgrading-cef.md`](docs/upgrading-cef.md), the diff scripts are
      `tools/cefdiff/`; the 62 → 151 diff dumps and the second copy of the CEF headers
      are gone — the port they described is done.
- [x] **Separate the harness from the examples.** `cmd/main` is now `cmd/harness`, with
      its scenarios documented in [`cmd/harness/README.md`](cmd/harness/README.md); the
      copyable code lives in `examples/`.

### Distribution

- [x] **Document shipping an application.** Which files from `Release/` are mandatory
      and where they have to sit, per platform, in [`docs/`](docs/) — taken from CEF's
      own redistribution lists. `bootstrap.exe` / `bootstrapc.exe` turn out to be for
      client DLLs implementing `RunWinMain` / `RunConsoleMain`, so a Go executable
      never ships them.
- [ ] **Sandbox support.** `no_sandbox = 1` is hard-coded. Windows would need
      `cef_sandbox.lib` linked in, macOS entitlements and a sandboxed helper build,
      Linux the setuid `chrome-sandbox` — none of which fits the runtime-binding
      design as it stands.
- [ ] **Code signing beyond ad-hoc.** `cefbundle` signs ad-hoc, which runs locally and
      nothing else; Developer ID, hardened runtime and notarization are open, and so is
      Authenticode on Windows.
- [ ] **Offline or mirrored downloads.** `cefdl` always talks to the Spotify CDN;
      an air-gapped build needs a way to point it at a local mirror.
