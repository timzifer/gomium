# Third-party notices

gomium itself is licensed under the BSD 3-Clause License, see [`LICENSE`](LICENSE).
This file lists the third-party material the repository contains and the third-party
binaries its tooling downloads.

## Chromium Embedded Framework (CEF) — vendored headers

`include/` holds the C API headers of the CEF binary distribution (version pinned in
[`cefdl/cef.version`](cefdl/cef.version)), with local patches. Each header carries its
own copyright notice; the license is:

```
Copyright (c) 2008-2020 Marshall A. Greenblatt. Portions Copyright (c)
2006-2009 Google Inc. All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are
met:

   * Redistributions of source code must retain the above copyright
notice, this list of conditions and the following disclaimer.
   * Redistributions in binary form must reproduce the above
copyright notice, this list of conditions and the following disclaimer
in the documentation and/or other materials provided with the
distribution.
   * Neither the name of Google Inc. nor the name Chromium Embedded
Framework nor the names of its contributors may be used to endorse
or promote products derived from this software without specific prior
written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
```

Two files under `include/base/internal/` (`cef_color_id_macros.inc`,
`cef_net_error_list.h`) come from Chromium and carry "Copyright The Chromium Authors",
under Chromium's BSD-style license — the same terms as above, with "The Chromium
Authors" as the copyright holder.

## CEF and Chromium binaries — downloaded, not in the repository

`tools/cefdl` (and `gomium.FetchRuntime`) download the CEF binary distribution from the
official CEF build CDN. Nothing of it is checked in. The distribution contains `libcef`
— Chromium plus CEF — and with it the code of several hundred third-party components,
under their own licenses (BSD, MIT, Apache-2.0, LGPL, MPL and others).

`cefdl` unpacks the two notice files of the distribution next to the runtime:

| file in the CEF archive | written as | content |
| --- | --- | --- |
| `LICENSE.txt` | `CEF_LICENSE.txt` | CEF's BSD license (text above) |
| `CREDITS.html` | `CREDITS.html` | licenses of Chromium and every component compiled into `libcef` |

**An application that ships the CEF runtime must ship both files with it.** On macOS,
`tools/cefbundle` copies them into `<App>.app/Contents/Resources/`; on Windows and Linux
keep them in the directory the runtime is shipped in. Making them viewable from the
application (an "About" or licenses page) is how Chromium-based products usually meet
the attribution requirements.

## Go module dependencies

Resolved by the Go toolchain from `go.mod`, not vendored:

| module | license |
| --- | --- |
| `github.com/google/uuid` | BSD-3-Clause |
| `golang.org/x/sys` | BSD-3-Clause |

Their license texts are in the respective modules; a binary built with them should
carry those notices too.
