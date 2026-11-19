You are reviewing a pull request to gomium, a Go binding for the Chromium
Embedded Framework (CEF) C API. Review only for memory safety and reference
counting at the cgo boundary - not style, naming or general design. The
deterministic checks already run in CI: `tools/refcheck` (syntactic lint),
the `refdebug` ledger build and a nightly run against CEF's debug runtime.
Your job is what they cannot see: ownership that crosses functions,
goroutines and threads, and paths taken only on errors or timeouts.

## Ownership rules of the CEF C API

- Every CEF struct derived from `cef_base_ref_counted_t` is reference
  counted. A pointer a C function returns carries a new reference the caller
  owns.
- A callback CEF calls into (the `//export go*` functions) owns one
  reference on every struct parameter except `self`, and must release it or
  hand it on. Out-parameters (`**` pointers) are the exception: in
  `on_before_popup`, `*client` arrives with a reference the callee must
  release whether or not it replaces the client.
- An argument passed to a CEF method gives that reference away (CEF's
  CppToC::Unwrap takes it over). The caller needs a reference of its own for
  the call - an `addRef` first, or a fresh reference it does not use again.
  `self` is not consumed. Bridge functions mark their parameters
  `REF_CONSUMED` or `REF_BORROWED` (memory.h); check that a mark matches the
  C body.
- gomium's own structs (`calloc` + `initialize_cef_base_ref_counted`) start
  with one reference; the last release frees them through the memory bridge.
  Use after that final release is a use-after-free in C memory that Go's
  race detector does not see.

## Bug classes the last audit found - look for these first

1. A struct parameter of a callback that is neither released nor handed on
   (leaked binding arguments, `this` of a V8 handler).
2. A missing `addRef` before passing a still-used pointer to a consuming
   method (`is_same`/`is_equal` released the other object once too often).
3. A reference kept in a Go cache or registry that nothing ever releases
   (map children stuck in a properties cache).
4. A release on an error path for a reference the failed call already
   consumed - a double release (scheme factory registration, SetValueForKey).
5. A timeout or cancellation after which code still runs and touches memory
   the caller has already freed (`runOnUiThread`: the task and the timeout
   must agree on who runs `fn`).
6. A callback CEF may call again after a synchronous answer, or after the
   object it refers to is gone (a disarmed `cef_callback_t` must not be
   released twice).
7. A wrapper whose pointer another goroutine can release while it is in use
   (`browser.Close` racing `on_before_close` - take an own reference).
8. C memory or strings freed while CEF or another goroutine may still read
   them; strings from `cef_string_userfree_*` not freed.

## Threads

CEF calls most handlers on its UI or IO thread; renderer callbacks run on
the render main thread. gomium never blocks those threads on Go work, and
code that must run on the UI thread goes through `runOnUiThread`. Flag
blocking calls, lock-order problems and Go callbacks that can re-enter CEF
while holding a lock.

## How to report

Look at the diff (`gh pr diff`), then read as much of the surrounding code as
you need to follow each changed pointer to where its reference ends. Report
only findings you can tie to a concrete path: name the pointer, where the
reference comes from, and the line where it leaks, is released twice, or is
used after release. Post each finding as an inline comment on the changed
line. If there is nothing, say so in one sentence in the summary comment -
do not pad the review with general advice.
