package main

import (
	"os"

	"github.com/timzifer/gomium"
	"log/slog"
)

// GOMIUM_HANDLER_TEST=1 wires log-only handlers for focus, keyboard, file
// dialog, frame and auth so the test page can be checked against them.
// GOMIUM_AUTH_TEST=1 answers auth challenges with gomium/secret; otherwise
// they fall through to CEF's dialog. The file dialog is always cancelled so a
// run never blocks on a picker.

func handlerTestEnabled() bool { return os.Getenv("GOMIUM_HANDLER_TEST") != "" }

type testFocusHandler struct{ logger *slog.Logger }

func (h testFocusHandler) OnTakeFocus(_ gomium.Browser, next bool) {
	h.logger.Info("handler-test: take focus", "next", next)
}

func (h testFocusHandler) OnSetFocus(_ gomium.Browser, source gomium.FocusSource) bool {
	h.logger.Info("handler-test: set focus", "source", int(source))
	return false
}

func (h testFocusHandler) OnGotFocus(_ gomium.Browser) {
	h.logger.Info("handler-test: got focus")
}

type testKeyboardHandler struct{ logger *slog.Logger }

func (h testKeyboardHandler) OnPreKeyEvent(_ gomium.Browser, ev gomium.KeyboardEvent) (bool, bool) {
	h.logger.Info("handler-test: pre key", "type", int(ev.Type), "wkc", ev.WindowsKeyCode, "mods", uint32(ev.Modifiers))
	return false, false
}

func (h testKeyboardHandler) OnKeyEvent(_ gomium.Browser, ev gomium.KeyboardEvent) bool {
	h.logger.Info("handler-test: key", "type", int(ev.Type), "wkc", ev.WindowsKeyCode)
	return false
}

type testFileDialogHandler struct{ logger *slog.Logger }

func (h testFileDialogHandler) OnFileDialog(_ gomium.Browser, mode gomium.FileDialogMode, title, defaultFilePath string, acceptFilters []string, callback gomium.FileDialogCallback) bool {
	h.logger.Info("handler-test: file dialog", "mode", int(mode), "title", title, "filters", acceptFilters)
	go callback.Cancel()
	return true
}

type testFrameHandler struct{ logger *slog.Logger }

func (h testFrameHandler) OnFrameCreated(_ gomium.Browser, _ gomium.CefFrame) {
	h.logger.Info("handler-test: frame created")
}

func (h testFrameHandler) OnFrameDestroyed(_ gomium.Browser, _ gomium.CefFrame) {
	h.logger.Info("handler-test: frame destroyed")
}

func (h testFrameHandler) OnFrameAttached(_ gomium.Browser, _ gomium.CefFrame, reattached bool) {
	h.logger.Info("handler-test: frame attached", "reattached", reattached)
}

func (h testFrameHandler) OnFrameDetached(_ gomium.Browser, _ gomium.CefFrame) {
	h.logger.Info("handler-test: frame detached")
}

func (h testFrameHandler) OnMainFrameChanged(_ gomium.Browser, oldFrame, newFrame gomium.CefFrame) {
	h.logger.Info("handler-test: main frame changed", "old_nil", oldFrame == nil, "new_nil", newFrame == nil)
}

type testAuthHandler struct{ logger *slog.Logger }

func (h testAuthHandler) OnAuthCredentials(_ gomium.Browser, req gomium.AuthRequest, callback gomium.AuthCallback) gomium.AuthAction {
	h.logger.Info("handler-test: auth challenge", "host", req.Host, "port", req.Port, "realm", req.Realm, "scheme", req.Scheme, "proxy", req.IsProxy)
	if os.Getenv("GOMIUM_AUTH_TEST") == "" {
		return gomium.AuthDefault
	}
	go callback.Continue("gomium", "secret")
	return gomium.AuthHandled
}
