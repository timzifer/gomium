# Upgrading the CEF version

The procedure below is what the 3.3202 → 151.3.14 port actually used. It is written
down because the expensive part of a CEF upgrade is not the typing — it is finding out
*which* of ~180 C-API structs changed under you.

For crashes and behaviour changes the last upgrade uncovered, see
[cef-internals.md](cef-internals.md).

## 1. Pick the target and pin it

Distributions live at <https://cef-builds.spotifycdn.com/>. Put the exact version
string in [`cefdl/cef.version`](../cefdl/cef.version) — `tools/cefdl` reads it, and the API-hash
check in `internal/cef/app.go` fails loudly if the headers and the runtime disagree.
`TestPinnedVersionMatchesHeaders` (`go test ./cefdl`, runs in CI without a distribution)
fails until `include/cef_version.h` carries the same version, so pin and headers move
together.

```
go generate .                 # tools/cefdl: fetches the distribution into ./Release
go run ./tools/cefdl -headers # refresh include/ from the same distribution
```

## 2. Generate the new C-API headers

Modern CEF does **not** check the `capi/` headers into the repository — they are
generated. A distribution's `include/` is enough for building, but for *diffing* you
want the full generated tree:

```
git clone --filter=blob:none --sparse https://github.com/chromiumembedded/cef -b <branch>
cd cef && git sparse-checkout set include tools
python tools/translator.py --root-dir .
```

The translator's version assert expects a `chromium/src` layout and fails outside it.
That failure is harmless — capi generation completes anyway (161 headers plus the
`*_versions.h` files).

## 3. Diff the old and new C-API trees

Three scripts in [`../tools/cefdiff`](../tools/cefdiff):

| script | answers |
|---|---|
| `capidiff.py` | which structs and global `CEF_EXPORT` functions were added or removed |
| `sigdiff.py` | which members that exist in *both* versions changed signature |
| `settingsdiff.py` | field-level diff of `cef_settings_t`, `cef_browser_settings_t`, `cef_window_info_t` |

```
python tools/cefdiff/capidiff.py <old-capi-dir> <new-capi-dir>
python tools/cefdiff/sigdiff.py  <old-capi-dir> <new-capi-dir>
```

`settingsdiff.py` has its paths hardcoded at the top — edit them before running.

Read `sigdiff.py`'s output first: renamed-but-otherwise-identical structs (CEF 127
renamed every `cef_v8xxx_t` to `cef_v8_xxx_t`) inflate the added/removed lists in
`capidiff.py` and make the jump look larger than it is.

## 4. Work through the diff

Order that worked: headers and distribution first (mechanical), then callback
signatures and struct fields (mechanical), then the structural changes (design), then
first start and crash debugging, then verification with `cmd/harness`'s scenarios.

Crash debugging dominates the schedule, not the edits. Budget accordingly.

## 5. `CEF_API_VERSION` is pinned

CEF 131+ supports pinning the API version by defining `CEF_API_VERSION` before the
includes and checking it with `cef_api_hash(version, entry)`. gomium pins it to `15600`
(`-DCEF_API_VERSION=15600` in the cgo `CFLAGS`, `internal/cef/cgoflags.go`); the 151→156
migration introduced it. Without the pin the headers default to
`CEF_API_VERSION_EXPERIMENTAL`, where every CEF release is free to change the ABI under
the binding.

On an upgrade, keep the pin unless you need API that only a newer version exposes:
a newer CEF still serves the pinned version, which keeps the upgrade a version bump
rather than a port. Raising it is a deliberate step — bump the define, regenerate what
depends on it, and read the changed structs in the headers.

## 6. New symbols need a shim entry

The binding resolves libcef at runtime, so any newly used `cef_*` export must be added
to `internal/cefshim/cef_shim_symbols.txt` followed by `go generate ./internal/cefshim`
— otherwise it links against the unbound backstop and aborts on first call. See the
runtime-binding section in the [README](../README.md).
