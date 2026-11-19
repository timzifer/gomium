package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"net/url"
	"strings"

	"github.com/timzifer/gomium"
	"log/slog"
)

// testInterceptor exercises the ResourceInterceptor API: every .pdf request
// is answered with a generated PNG instead of hitting the network.
type testInterceptor struct {
	logger *slog.Logger
}

func (t *testInterceptor) OnProtocolExecution(_ gomium.Browser, _ *url.URL) bool { return false }
func (t *testInterceptor) OnBeforeBrowse(_ gomium.Browser, _ *url.URL) bool      { return false }
func (t *testInterceptor) OnBeforeResourceLoad(_ gomium.Browser, _ *gomium.Request, _ gomium.ResourceLoadAsyncResult) gomium.ResourceLoadResult {
	return gomium.ResourceLoadContinue
}

func (t *testInterceptor) InterceptResource(_ gomium.Browser, requestUrl *url.URL, method string) gomium.ResourceContentFn {
	if !strings.HasSuffix(requestUrl.Path, ".pdf") {
		return nil
	}
	t.logger.Info("intercepting pdf request", "uri", requestUrl.String())

	return func() *gomium.ResourceContent {
		img := image.NewRGBA(image.Rect(0, 0, 64, 64))
		for i := range img.Pix {
			switch i % 4 {
			case 0:
				img.Pix[i] = 0xE0 // red
			case 3:
				img.Pix[i] = 0xFF
			}
		}
		_ = color.Black

		buffer := bytes.NewBuffer(nil)
		if err := png.Encode(buffer, img); err != nil {
			t.logger.Warn("intercept: could not encode png", "err", err)
			return nil
		}
		t.logger.Info("intercept: serving generated png", "bytes", buffer.Len())
		return &gomium.ResourceContent{
			Data:     buffer.Bytes(),
			MimeType: "image/png",
		}
	}
}
