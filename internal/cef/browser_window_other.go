//go:build !windows && !darwin

package cef

import "errors"

// applyWindowInfo reports that windowed browsers exist on Windows and macOS
// only. The offscreen path (WithRenderHandler) works everywhere.
func (b *browser) applyWindowInfo() error {
	return errors.New("windowed browsers are only implemented on windows and macos")
}

func (b *browser) releaseWindowInfo() {}
