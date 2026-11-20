# gomium on macOS

**Status: works.** Offscreen and windowed browsers both start, load, render and shut
down on darwin/arm64, out of a bundle built by [`tools/cefbundle`](../tools/cefbundle).
darwin/amd64 compiles and shares every code path, but has not been run.

macOS is the platform with the most host obligations. Three of them have no equivalent
on Windows or Linux, and skipping any one of them fails in a way that does not name
its cause:

| obligation | what happens without it |
| --- | --- |
| `InstallMacApplication()` before CEF starts | CEF aborts — it will not run under an `NSApplication` that does not implement `CefAppProtocol` |
| An `.app` bundle with helper bundles | `cef_initialize` fails; CEF cannot launch helpers out of the main executable |
| Draining `NSApp`'s event queue | the window paints once and then ignores mouse, keyboard and its own close |

---

## Requirements

- Xcode command line tools (`xcode-select --install`) — the application bootstrap is
  Objective-C, compiled by cgo.
- arm64 or x86-64. `internal/cefshim/cef_shim.S` carries a Mach-O arm64 block; nothing else is
  architecture specific.

## Getting the runtime

```sh
go run ./tools/cefdl        # macosarm64 / macosx64 by GOARCH, into ./Release
```

The macOS distribution ships `Chromium Embedded Framework.framework` rather than a
single library. Everything CEF needs is inside it — `icudtl.dat`,
`v8_context_snapshot.bin`, the `.pak` resources and the locales all live in
`Resources/`, the ANGLE and SwiftShader libraries in `Libraries/`. There is no
separate resources or locales directory to ship, and `WithResourcesDirPath` /
`WithLocalesDirPath` are ignored here.

`LoadLibcef` accepts either the bundle or the binary inside it:

```
Release/Chromium Embedded Framework.framework
Release/Chromium Embedded Framework.framework/Chromium Embedded Framework
```

---

## Integration

```go
func main() {
    // AppKit and CEF both want the real main thread.
    runtime.LockOSThread()

    // Runtime binding, before anything else - helper processes are fresh
    // instances of this same executable and go through here too.
    framework := filepath.Join("Release", "Chromium Embedded Framework.framework")
    if bundle, fromBundle, _, err := gomium.MacBundlePaths(); err == nil {
        _ = bundle
        framework = fromBundle          // running from inside a .app
    }
    if err := gomium.LoadLibcef(framework); err != nil {
        log.Fatal(err)
    }

    // CEF drives the shared NSApplication through CefAppProtocol and refuses
    // to run under one that does not implement it. This must happen in the
    // browser process AND in every helper, so it belongs above the
    // IsSubprocess branch.
    if err := gomium.InstallMacApplication(); err != nil {
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

    // Browser process only, after cef_initialize. This is where cefclient
    // would enter [NSApp run].
    gomium.FinishMacApplicationLaunch()
    gomium.InstallMacDefaultMenu("MyApp")
    gomium.ActivateMacApplication()      // windowed hosts only

    // ... NewClient / NewBrowser, then drain the pump on this thread.
}
```

`opts()` must install an external message pump — see [Message loop](#message-loop).

### Application object

`InstallMacApplication()` creates an `NSApplication` subclass implementing
`CefAppProtocol` (`isHandlingSendEvent` / `setHandlingSendEvent:` / a `sendEvent:`
that saves and restores the flag around nested loops). AppKit keeps the first
`NSApplication` instance for the lifetime of the process, so this has to run before
anything else touches `NSApp`.

A host that already runs its own Cocoa application object — Fyne, Ebiten, an
Objective-C application — implements the protocol on **its** class instead. Calling
`InstallMacApplication()` then only reports whether that class conforms; if it does
not, the call fails and there is no way to fix it at runtime.

### Message loop

`multi_threaded_message_loop` does not exist on macOS. `WithMessagePumpWorker` is the
only option, and `cef_do_message_loop_work` must run on the thread that called
`cef_initialize`:

```go
gomium.WithMessagePumpWorker(func(delay int, worker gomium.Worker) {
    go func() {
        if delay > 0 {
            time.Sleep(time.Duration(delay) * time.Millisecond)
        }
        select {
        case pumpChannel <- worker:   // drained by the main goroutine
        default:                      // a missed tick beats a leaked goroutine
        }
    }()
}),
```

```go
for {
    select {
    case <-ctx.Done():
        return
    case work := <-pumpChannel:
        work()          // on the main thread
    }
}
```

### The AppKit event queue

CEF's message loop and `NSApp`'s event queue are two different things. `cefclient`
keeps `-[NSApplication run]` going and only decides *when* to call
`CefDoMessageLoopWork` from inside it. gomium owns the loop instead, so it drains
`NSApp` itself on every pump tick — `nextEventMatchingMask:` with `distantPast`,
bounded, main-thread guarded, in `internal/cef/app_darwin.m`.

What stays with the host are the calls AppKit would otherwise get from `-run`:

| call | when | why |
| --- | --- | --- |
| `FinishMacApplicationLaunch()` | once, after `Run` | nothing else calls `finishLaunching`; without it the process stays half-launched |
| `InstallMacDefaultMenu(name)` | once | macOS dispatches ⌘C / ⌘V / ⌘A through the menu bar — with no main menu they never reach the page |
| `ActivateMacApplication()` | windowed hosts only | a binary started from a terminal is not the active application, so its first window opens behind the terminal |

All three do nothing under a host that runs its own event loop, and all three are
browser-process only.

### Windows and views

`WithWindow(title, w, h)` lets CEF create and own the `NSWindow`.
`WithParentView(view)` puts the browser into an `NSView` the host already has — the
bounds are then read as the frame inside that view, and the window title is ignored.
Both pin the runtime style to Alloy, so no Chromium browser interface comes along.

### Paths

`WithFrameworkDirPath`, `WithMainBundlePath` and the cross-platform
`WithBrowserSubprocessPath` fill the macOS fields of `cef_settings_t`. **None of them
is needed inside a bundle built by `tools/cefbundle`** — CEF finds the framework and
the helper at their conventional places. Set them when the framework lives somewhere
else, or when the executable is not inside the bundle it belongs to.

`MacBundlePaths()` derives bundle, framework and helper paths from the running
executable and understands both layouts CEF launches (browser and helper), so the same
code works in either process. It returns an error outside a bundle, which is the
normal case during development — fall back to `./Release` then.

---

## Shipping an application

```sh
go run ./tools/cefdl                              # Release/…framework
go build -o build/myapp ./cmd/myapp
go run ./tools/cefbundle -bin build/myapp -cef Release \
    -out build/MyApp.app -name MyApp -id com.example.myapp
```

Flags: `-version`, `-min-os` (default `11.0`) and `-sign` (default `-`, ad-hoc; empty
skips signing).

`cefbundle` also copies the license notices `cefdl` put into the `-cef` directory
(`CEF_LICENSE.txt`, `CREDITS.html`) into `Contents/Resources/`. Shipping them is a
redistribution obligation (see [`THIRD_PARTY_NOTICES.md`](../THIRD_PARTY_NOTICES.md)); a
`-cef` directory from an older `cefdl` run lacks them, and `cefbundle` warns.

The result, ad-hoc signed:

```
MyApp.app/Contents/
  Info.plist
  MacOS/MyApp
  Frameworks/
    Chromium Embedded Framework.framework/
    MyApp Helper.app/Contents/{Info.plist,MacOS/MyApp Helper}
    MyApp Helper (Alerts).app/…
    MyApp Helper (GPU).app/…
    MyApp Helper (Plugin).app/…
    MyApp Helper (Renderer).app/…
```

Every helper executable is a copy of the same binary — `IsSubprocess()` routes all of
them into `ExecuteSubprocess`. Only the plain `MyApp Helper.app` is mandatory; the
four variants mirror cefclient's layout, cost a copy each and leave room to give one
process type different entitlements later. The helper `Info.plist` files carry
`LSUIElement`, so renderer processes stay out of the Dock.

`Info.plist` deliberately has no `NSPrincipalClass`: `InstallMacApplication()` puts the
`CefAppProtocol` class in place before AppKit would read it.

### Why the bundle is staged in a scratch directory

`codesign` rejects a bundle carrying `com.apple.FinderInfo`. A file provider — iCloud
Drive over `~/Documents`, for one — restores that attribute on every `.app` directory
within a second of it being stripped, so signing in place is a race that cannot be won:

```
MyApp.app: resource fork, Finder information, or similar detritus not allowed
```

`cefbundle` therefore assembles and signs in `os.MkdirTemp`, which no provider watches,
and moves the finished bundle into `-out` afterwards. An attribute stamped on an
already-signed bundle no longer invalidates it.

### Signing for distribution

Ad-hoc signatures run locally and nothing more. Distribution needs a Developer ID
identity (`-sign "Developer ID Application: …"`), the hardened runtime, and
notarization — none of which `cefbundle` does yet. `codesign -vv MyApp.app` should
report *valid on disk* and *satisfies its Designated Requirement* before you go
further.

One caveat: CEF's own README notes that Xcode 26 wants a **versioned** framework
directory (`Versions/A` plus relative symlinks), while the distribution ships a flat
one. `cefbundle` copies the framework as it finds it and preserves symlinks, so both
shapes survive the copy, but it does not convert between them.

### The Windows-only files

`bootstrap.exe` / `bootstrapc.exe` in the Windows distribution are for client **DLLs**
implementing `RunWinMain` / `RunConsoleMain` (`include/cef_sandbox_win.h`). They have
no macOS counterpart and nothing to do with a Go executable.

### The keychain ("Chromium Safe Storage")

Chromium encrypts cookies and stored credentials on disk (OSCrypt) with a key it keeps
in the login keychain as **Chromium Safe Storage**. The keychain grants access per code
identity, and an ad-hoc signature (`cefbundle -sign -`) is a new identity on every
build. Each rebuilt bundle therefore raises a keychain prompt, several at once, one per
process that reaches for the key, and until somebody answers:

- the network service blocks, so the first page never finishes loading,
- `Close` reaches `on_before_close`, then Chromium's own *Teardown watchdog expired;
  recording dump and terminating* ends the process with exit code 2.

`WithMockKeychain(true)` (pass it to `New` and to `ExecuteSubprocess`) switches Chromium
to a fixed key (`--use-mock-keychain`): no prompt, nothing blocks. The price is that the
profile on disk is no longer protected; anyone who can read `WithCachePath` can decrypt
its cookies. `cmd/harness` and both examples turn it on.

An application that ships signed with a stable identity (Developer ID) should leave it
off. The user then allows access once ("Always Allow"), and the grant survives updates
as long as the designated requirement stays the same.

---

## Sandbox

gomium sets `no_sandbox = 1` unconditionally. Enabling the CEF sandbox on macOS needs
entitlements and a helper build that gomium does not produce today.

---

## Troubleshooting

**`the NSApplication of this process does not implement CefAppProtocol`**
Something created `NSApp` before `InstallMacApplication()` ran, or the host's own
application class does not conform. AppKit keeps the first instance forever — the fix
is at the call order, not at runtime.

**`could not initialize cef-app`**
Almost always the bundle layout. Check that
`Contents/Frameworks/Chromium Embedded Framework.framework` and
`Contents/Frameworks/<App> Helper.app/Contents/MacOS/<App> Helper` both exist, and run
with `WithLogSeverity(gomium.LOGSEVERITY_VERBOSE)` — CEF's own diagnosis is more
specific than the return value.

**`timed out waiting for browsers to close`**
The pump is not reaching the main thread. `cef_do_message_loop_work` — and with it the
`NSApp` drain — has to run on the `cef_initialize` thread; a pump worker that calls it
from a goroutine leaves the AppKit queue untouched.

**Keychain prompts for "Chromium Safe Storage", the first page never loads, exit code 2**
A new ad-hoc signed build asking for the OSCrypt key, see
[The keychain](#the-keychain-chromium-safe-storage). Use `WithMockKeychain(true)` during
development.

**The window opens behind the terminal, or ⌘C does nothing**
`ActivateMacApplication()` and `InstallMacDefaultMenu()` were not called.

**`cef version mismatch`**
`Release/` and the vendored `include/` disagree. `go run ./tools/cefdl` fetches the
version pinned in [`cefdl/cef.version`](../cefdl/cef.version).

---

## Verifying a build

```sh
codesign -vv build/MyApp.app                      # valid on disk
./build/MyApp.app/Contents/MacOS/MyApp            # from a real desktop session
ps -ax | grep "MyApp Helper"                      # renderer/gpu helpers present
curl -s localhost:6696/json/version               # with WithRemoteDebuggingPort
lsappinfo front                                   # windowed: the app is frontmost
```

AppKit needs a real desktop session — a window will not appear over SSH.
