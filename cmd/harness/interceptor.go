package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"net/url"
	"strings"

	"github.com/rs/zerolog"
	"github.com/timzifer/gomium"
)

// testInterceptor exercises the ResourceInterceptor API: every .pdf request
// is answered with a generated PNG instead of hitting the network.
type testInterceptor struct {
	logger zerolog.Logger
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
	t.logger.Info().Str("uri", requestUrl.String()).Msg("intercepting pdf request")

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
			t.logger.Warn().Err(err).Msg("intercept: could not encode png")
			return nil
		}
		t.logger.Info().Int("bytes", buffer.Len()).Msg("intercept: serving generated png")
		return &gomium.ResourceContent{
			Data:     buffer.Bytes(),
			MimeType: "image/png",
		}
	}
}
