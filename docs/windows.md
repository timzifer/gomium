# gomium on Windows

**Status: works on windows/amd64.** This is the platform whose runtime path has been
exercised longest. **windows/arm64 does not build** — `internal/cefshim/cef_shim.S` has ELF and Mach-O
arm64 blocks but no PE-COFF one, so it stops at an `#error`.

Windows is the least demanding of the three: helper processes are re-executions of the
same `.exe`, there is no bundle to build, and CEF can run its own message loop thread.

---

## Requirements

- A cgo toolchain — MinGW-w64 or TDM-GCC. The assembly shim is GAS syntax, so MSVC's
  assembler is not an option.
- x86-64.

## Getting the runtime

```sh
go run ./tools/cefdl        # windows64, into .\Release
```

`cefdl` flattens the distribution's `Release\` and `Resources\` into a single
`.\Release` — which is exactly the layout CEF expects next to an executable.

---

## Integration

```go
func main() {
    runtime.LockOSThread()

    // Runtime binding, before anything else - helper processes are fresh
    // instances of this same executable and go through here too.
    if err := gomium.LoadLibcef(filepath.Join("Release", "libcef.dll")); err != nil {
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

    // ... NewClient / NewBrowser
}
```

### Module handle

CEF wants the process `HINSTANCE` in `cef_main_args_t`:

```go
handle, err := gomium.GetModuleHandle(nil)
if err != nil {
    log.Fatal(err)
}
opts = append(opts, gomium.WithModuleHandle(handle))
```

This is Windows-only API — keep the call behind a `//go:build windows` file in a
cross-platform host.

### DLL search path

`LoadLibcef` opens `libcef.dll` with `LOAD_WITH_ALTERED_SEARCH_PATH` on an absolute
path, so its dependencies — `chrome_elf.dll`, `vulkan-1.dll`, `dxcompiler.dll` and the
rest — resolve from the directory holding `libcef.dll` rather than from `PATH`. The
distribution stays relocatable and a second CEF build elsewhere on the machine cannot
be bound by accident.

### Message loop

Windows is the only platform with a real choice:

| | when to use it |
| --- | --- |
| `WithMultiThreadedMessageLoop(true)` | CEF runs its own message loop thread; nothing to pump on the host side |
| `WithMessagePumpWorker(...)` | the host owns the loop (Fyne, GLFW, a game loop) - offscreen and windowed browsers |

The two are mutually exclusive — `Run` rejects the combination.

With an external pump every worker does two things: one `cef_do_message_loop_work`
and a drain of the calling thread's Win32 message queue (`PeekMessage` /
`TranslateMessage` / `DispatchMessage`). CEF's external pump never touches the Win32
queue itself, and the windows it creates on the UI thread — the browser window, but
also Chromium's hidden top-level windows in an offscreen host — would otherwise go
unserviced. `WM_QUIT` and thread messages (`hwnd == NULL`) are re-posted and left to
the host's own loop.

Workers arrive at every due time CEF asks for and at least every
`WithMessagePumpWorkerInterval` (default 1000/30 ms, as in cefclient). The interval
is also the worst-case input latency of a windowed browser when the host loop does
not dispatch Win32 messages itself. While the user drags or resizes a window,
Windows runs a modal loop inside `DispatchMessage`, and CEF is not pumped until it
ends.

With an external pump, `cef_do_message_loop_work` must run on the thread that called
`cef_initialize`; the pump worker hands the callback back and the main goroutine
executes it.

That thread is then CEF's UI thread, and the host's UI code runs on it too. Calls that
block on a UI-thread callback — `NewBrowser` (`on_after_created`), `Client.Close`
(`on_before_close`), cookie operations, `PrintToPdf`, DevTools `Call` — pump the loop
themselves while they wait when they are made on that thread. Without that they would
block the only thread that can deliver the callback and run into their timeout. A call
from inside a CEF callback cannot pump (CEF must not be pumped reentrantly) and still
times out.

### Windows

`WithWindow(title, w, h)` / `WithWindowBounds(title, rect)` let CEF create and own a
top-level `HWND` (`WS_OVERLAPPEDWINDOW | WS_CLIPCHILDREN | WS_CLIPSIBLINGS |
WS_VISIBLE`). The runtime style is pinned to Alloy, so no Chromium browser interface
comes along.

---

## Shipping an application

There is no bundle format. Ship the executable with the CEF files beside it:

```
MyApp.exe
libcef.dll
chrome_elf.dll
icudtl.dat
v8_context_snapshot.bin
chrome_100_percent.pak
chrome_200_percent.pak
resources.pak
d3dcompiler_47.dll
dxcompiler.dll
dxil.dll
libEGL.dll
libGLESv2.dll
vk_swiftshader.dll
vk_swiftshader_icd.json
vulkan-1.dll
locales\
  en-US.pak, de.pak, …
```

Per CEF's own `README.txt` for this distribution:

| component | files | required? |
| --- | --- | --- |
| CEF core library | `libcef.dll` | **yes** |
| Crash reporting | `chrome_elf.dll` | **yes** |
| Unicode data | `icudtl.dat` | **yes** |
| V8 snapshot | `v8_context_snapshot.bin` | **yes** |
| Non-localized resources | `chrome_100_percent.pak`, `chrome_200_percent.pak`, `resources.pak` | optional — web components render incorrectly without them |
| Localized resources | `locales\*.pak` | optional — only the locales you configure are needed; default `en-US` |
| Direct3D | `d3dcompiler_47.dll` | optional — GPU-accelerated canvas / 3D CSS / WebGL |
| DirectX compiler | `dxcompiler.dll`, `dxil.dll` | optional — WebGPU |
| ANGLE | `libEGL.dll`, `libGLESv2.dll` | optional — canvas / 3D CSS / WebGL |
| SwANGLE | `vk_swiftshader.dll`, `vk_swiftshader_icd.json`, `vulkan-1.dll` | optional — software rendering when the GPU is unavailable |
| License notices | `CEF_LICENSE.txt`, `CREDITS.html` | **yes** — redistribution obligation, not a runtime need; see [`THIRD_PARTY_NOTICES.md`](../THIRD_PARTY_NOTICES.md) |

Two files in `Release\` are **not** shipped:

- `libcef.lib` — the import library. gomium binds at runtime, so nothing links
  against it.
- `bootstrap.exe` / `bootstrapc.exe` — for client **DLLs** implementing `RunWinMain` /
  `RunConsoleMain` (`include/cef_sandbox_win.h`). A Go executable is not that.

### Somewhere other than next to the executable

CEF looks for the resources and locales beside the executable by default. When they
live elsewhere — a shared installation directory, or a development tree where the
binary is built into `.\build` and the runtime sits in `.\Release` — point CEF at them:

```go
gomium.WithResourcesDirPath(`C:\Program Files\MyApp\cef`),
gomium.WithLocalesDirPath(`C:\Program Files\MyApp\cef\locales`),
```

Both must be absolute; `LoadLibcef` takes the `libcef.dll` path separately.

`WithBrowserSubprocessPath` points CEF at a different executable for its helpers. Leave
it unset unless the helpers really are a separate binary — empty means "re-execute this
one", which is what `IsSubprocess()` / `ExecuteSubprocess()` are built for.

---

## Sandbox

gomium sets `no_sandbox = 1` unconditionally. Enabling the Chromium sandbox on Windows
needs `cef_sandbox.lib` linked into the executable (see `include/cef_sandbox_win.h`),
which the runtime-binding design deliberately avoids.

---

## Troubleshooting

**`gomium: loading …\libcef.dll: …`**
Either the file is missing or a dependency next to it is. `LOAD_WITH_ALTERED_SEARCH_PATH`
resolves dependencies from `libcef.dll`'s own directory — check `chrome_elf.dll` is
there.

**`gomium: … does not export: …`**
The DLL is a different CEF version than the headers in `include/`. `cefdl/cef.version` is the
pin; `go run ./tools/cefdl` fetches the matching build.

**`timed out waiting for browsers to close` on a windowed browser**
With an external pump: the workers are not executed on the thread that called
`cef_initialize`, or `Close` is called from another thread. Both drain that thread's
Win32 queue only.

**`Opening in existing browser session`**
Two CEF applications are sharing the default user-data directory and one holds the
Chromium process singleton. Give each its own `WithCachePath`.

**`multi-threaded Message loop enabled`**
`WithMultiThreadedMessageLoop(true)` and `WithMessagePumpWorker` were both set. Pick one.
