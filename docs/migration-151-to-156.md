# Migration: CEF 151.3.14 → CEF 156.0.27

Diese Anleitung beschreibt, was sich für Konsumenten von gomium ändert, wenn
die Bibliothek von der letzten CEF-151-Version auf die aktuelle CEF-156-Version
umgestellt wird. Das Verfahren für künftige Upgrades steht in
[`upgrading-cef.md`](upgrading-cef.md); die Hintergründe zu Crashes und
Verhalten in [`cef-internals.md`](cef-internals.md).

| | vorher | jetzt |
| --- | --- | --- |
| CEF | `151.3.14+g5d67476+chromium-151.0.7922.72` | `156.0.27+ga10dae3+chromium-156.0.8078.88` |
| Client-API | `CEF_API_VERSION_EXPERIMENTAL` | gepinnt auf `15600` (`-DCEF_API_VERSION=15600`) |
| Extensions (`WithExtension`) | vorhanden | entfernt |
| HTTP-Auth ohne Handler | CEF-Login-Dialog | Abbruch (401-Antwort) |

## Warum das Upgrade nötig war

- **Datei-Dialog im OSR-Modus stürzt ab** (bis 151.3.14). Ein Klick auf
  `<input type="file">` in einem Offscreen-Browser führte zu einer Access
  Violation in `libcef`, bevor `FileDialogHandler` gerufen wurde. Ursache:
  `CefWebContentsViewOSR::GetNativeView()` liefert eine leere View, und
  `FileSelectHelper` dereferenziert sie. In 151.3.24 und 156.0.27 nicht mehr
  reproduzierbar.
- **Extensions sind ab CEF 154 deprecated.** `CefRegisterExtension` wurde
  durch Chromium-seitige Kosten (v8::Extensions bei jeder Kontext-Erstellung)
  ausgemustert und in 156 entfernt.

## Breaking Changes für API-Nutzer

### 1. Extension-API entfernt

Entfernt sind: `WithExtension`, `Extension`, `WithExtensionCallGuard`,
`WithDefaultExtensionCallGuard`, `DenyExtensionCalls`, `ExtensionCall`,
`ExtensionCallGuard`, `SameOrigin`.

**Was du tun musst:** Aufrufe entfernen. Die JS-Seite verliert damit das
Objekt, das bisher pro Frame eingeblendet wurde. Wenn du dieses Verhalten
brauchst, bindet man die Funktionen künftig pro Kontext im
`OnContextCreated` des Render-Prozesses (`CefV8Value::CreateFunction`), ohne
JS-Quelltext zu kompilieren. Das ist nicht implementiert.

Alternative für JS-Aufrufe in die Anwendung: `ExecuteJavaScriptWithResult`
bzw. `ExecuteJavascript` aus Go heraus, und in der Gegenrichtung Browser-
Nachrichten über die Handler.

### 2. HTTP-Auth geht immer an den Client-Handler

Der Schalter `disable-chrome-login-prompt` gilt prozessweit. Dadurch gilt:

- **Mit `WithAuthHandler`**: jede Challenge (Basic, Digest, NTLM, Proxy) wird
  an den Handler gerufen. `callback.Continue(user, pass)` oder
  `callback.Cancel()` genau einmal aufrufen.
- **Ohne Handler oder mit `AuthDefault`**: die Anfrage wird abgebrochen. Die
  Seite zeigt die 401-Antwort. Es erscheint **kein** CEF-Dialog mehr.

**Was du tun musst:** Wenn deine Anwendung Auth-Seiten anzeigt, `WithAuthHandler`
setzen. Wer nie Auth braucht, ändert nichts.

### 3. Shim und Symbole

`cef_register_extension` ist aus der Symbolliste entfernt; `cef_shim.S`,
`cef_shim.c` und `cef_shim_symbols.go` sind neu generiert
(`go generate ./internal/cefshim`). Eigene Aufrufe auf CEF-Symbole müssen in
`internal/cefshim/cef_shim_symbols.txt` eingetragen werden.

### 4. Custom Schemes in beiden Prozessen

Ein Schema muss in `gomium.New` **und** in `gomium.ExecuteSubprocess`
identisch registriert werden. Sonst kennt der Renderer-Subprozess es nicht,
und die Seite bleibt leer. Der Repro in `fyne/browser/cmd/reprofile` zeigt das
Muster (`gtestScheme()`).

## Unverändert, aber zu prüfen

- **`WithSingleProcess`** ist ein No-op, Single-Process gibt es in CEF 151 und
  156 nicht mehr. Renderer laufen als Subprozesse desselben Binaries.
- **Zertifikatsfehler**: `WithIgnoreCertificateErrors(true)` bleibt möglich,
  für den echten Pfad `WithCertificateErrorHandler` verwenden.
- **Downloads** (`WithDownloadDirectory`), **PDF** (`PrintToPdf`),
  **Popups** (`WithPopupHandler`), **Focus/Keyboard/Frame-Handler**: API
  unverändert, Verhalten in 156 geprüft (siehe unten).

## Build und Umgebung

- `cefdl/cef.version` (früher `cef.version` im Root) ist die einzige Quelle für die Zielversion. Nach dem Tausch
  der Header `go clean -cache` ausführen. Der Go-Build-Cache hält sonst die
  alten C-Objekte, und das Binary meldet weiter die alte Version.
- Der Start prüft, ob die geladene `libcef` zur kompilierten Version passt
  (`cef version mismatch`). Eine falsche Runtime bricht also früh ab.
- Runtime-Dateien kommen aus der `minimal`-Distribution (`Release/` und
  `Resources/`). Die Ressourcen liegen **nicht** im `Release/`-Ordner.
- Zum Test: `icudtl.dat` und die `.pak`-Dateien fehlen sonst, und der Start
  stürzt mit `Invalid file descriptor to ICU data` ab.

## Konsumenten-Checkliste

1. `go.mod`: gomium auf den neuen Stand pinnen.
2. `WithExtension`-Aufrufe entfernen, JS-Kommunikation umbauen.
3. Auth-Seiten: `WithAuthHandler` setzen, sonst erscheint kein Dialog mehr.
4. Custom Schemes in `New` und `ExecuteSubprocess` identisch registrieren.
5. Runtime auf `156.0.27` umstellen (`minimal`-Distribution).
6. `go clean -cache` nach dem Header-Tausch.

## Stand der Verifikation (CEF 156.0.27, Windows)

| Bereich | Ergebnis |
| --- | --- |
| Browser-Start (OSR und Fenster) | ja |
| JS-Auswertung über Prozesskanal | ja |
| Datei-Dialog (OSR und Fenster) | ja, ein Klick ein Aufruf |
| HTTP-Auth (Continue, Cancel, Fyne-Dialog) | ja |
| PDF-Export | ja |
| Popup mit Nutzergeste (mit und ohne Handler) | ja |
| Download in Verzeichnis | ja |
| Custom Scheme | ja |
| Zertifikatsfehler (ohne Handler, deny, accept) | ja |
| Focus, Keyboard (echte Tasten), Frame | ja |
| Keyboard per DevTools-Injektion | **nein**: der Handler wird umgangen, Text kommt trotzdem an |

**Nicht geprüft:** macOS und Linux. Die Plattform-Header sind identisch mit den
156.0.27-Distributionen, ein Build oder Lauf dort ist aber nicht erfolgt.
