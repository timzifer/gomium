package main

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/timzifer/gomium"
)

// GOMIUM_HANDLER_TEST=1 wires log-only handlers for focus, keyboard, file
// dialog, frame and auth so the test page can be checked against them.
// GOMIUM_AUTH_TEST=1 answers auth challenges with gomium/secret; otherwise
// they fall through to CEF's dialog. The file dialog is always cancelled so a
// run never blocks on a picker.

func handlerTestEnabled() bool { return os.Getenv("GOMIUM_HANDLER_TEST") != "" }

type testFocusHandler struct{ logger zerolog.Logger }

func (h testFocusHandler) OnTakeFocus(_ gomium.Browser, next bool) {
	h.logger.Info().Bool("next", next).Msg("handler-test: take focus")
}

func (h testFocusHandler) OnSetFocus(_ gomium.Browser, source gomium.FocusSource) bool {
	h.logger.Info().Int("source", int(source)).Msg("handler-test: set focus")
	return false
}

func (h testFocusHandler) OnGotFocus(_ gomium.Browser) {
	h.logger.Info().Msg("handler-test: got focus")
}

type testKeyboardHandler struct{ logger zerolog.Logger }

func (h testKeyboardHandler) OnPreKeyEvent(_ gomium.Browser, ev gomium.KeyboardEvent) (bool, bool) {
	h.logger.Info().Int("type", int(ev.Type)).Int("wkc", ev.WindowsKeyCode).Uint32("mods", uint32(ev.Modifiers)).Msg("handler-test: pre key")
	return false, false
}

func (h testKeyboardHandler) OnKeyEvent(_ gomium.Browser, ev gomium.KeyboardEvent) bool {
	h.logger.Info().Int("type", int(ev.Type)).Int("wkc", ev.WindowsKeyCode).Msg("handler-test: key")
	return false
}

type testFileDialogHandler struct{ logger zerolog.Logger }

func (h testFileDialogHandler) OnFileDialog(_ gomium.Browser, mode gomium.FileDialogMode, title, defaultFilePath string, acceptFilters []string, callback gomium.FileDialogCallback) bool {
	h.logger.Info().Int("mode", int(mode)).Str("title", title).Strs("filters", acceptFilters).Msg("handler-test: file dialog")
	go callback.Cancel()
	return true
}

type testFrameHandler struct{ logger zerolog.Logger }

func (h testFrameHandler) OnFrameCreated(_ gomium.Browser, _ gomium.CefFrame) {
	h.logger.Info().Msg("handler-test: frame created")
}

func (h testFrameHandler) OnFrameDestroyed(_ gomium.Browser, _ gomium.CefFrame) {
	h.logger.Info().Msg("handler-test: frame destroyed")
}

func (h testFrameHandler) OnFrameAttached(_ gomium.Browser, _ gomium.CefFrame, reattached bool) {
	h.logger.Info().Bool("reattached", reattached).Msg("handler-test: frame attached")
}

func (h testFrameHandler) OnFrameDetached(_ gomium.Browser, _ gomium.CefFrame) {
	h.logger.Info().Msg("handler-test: frame detached")
}

func (h testFrameHandler) OnMainFrameChanged(_ gomium.Browser, oldFrame, newFrame gomium.CefFrame) {
	h.logger.Info().Bool("old_nil", oldFrame == nil).Bool("new_nil", newFrame == nil).Msg("handler-test: main frame changed")
}

type testAuthHandler struct{ logger zerolog.Logger }

func (h testAuthHandler) OnAuthCredentials(_ gomium.Browser, req gomium.AuthRequest, callback gomium.AuthCallback) gomium.AuthAction {
	h.logger.Info().Str("host", req.Host).Int("port", req.Port).Str("realm", req.Realm).Str("scheme", req.Scheme).Bool("proxy", req.IsProxy).Msg("handler-test: auth challenge")
	if os.Getenv("GOMIUM_AUTH_TEST") == "" {
		return gomium.AuthDefault
	}
	go callback.Continue("gomium", "secret")
	return gomium.AuthHandled
}
