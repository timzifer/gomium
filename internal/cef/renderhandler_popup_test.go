package cef

import (
	"image"
	"image/color"
	"testing"
)

// captureHandler wires an imageRenderHandler to a callback that remembers the
// last snapshot handed to the consumer.
func captureHandler(t *testing.T, w, h int) (*imageRenderHandler, **image.RGBA) {
	t.Helper()
	var last *image.RGBA
	rh := NewImageRenderHandler(image.NewRGBA(image.Rect(0, 0, w, h)), func(img image.Image) {
		last = img.(*image.RGBA)
	}).(*imageRenderHandler)
	return rh, &last
}

func fill(dst *image.RGBA, r image.Rectangle, c color.RGBA) {
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			dst.SetRGBA(x, y, c)
		}
	}
}

func TestPopupCompositedOverView(t *testing.T) {
	rh, last := captureHandler(t, 10, 10)
	white := color.RGBA{255, 255, 255, 255}
	red := color.RGBA{255, 0, 0, 255}

	fill(rh.image.(*image.RGBA), rh.image.Bounds(), white)
	rh.PopupSize(image.Rect(2, 2, 6, 6))
	rh.PopupShow(true)

	tile := image.NewRGBA(image.Rect(0, 0, 4, 4))
	fill(tile, tile.Bounds(), red)
	rh.DoPaintPopup([]DirtyArea{{Rectangle: tile.Bounds(), Buffer: tile}})

	if got := (*last).RGBAAt(3, 3); got != red {
		t.Fatalf("popup pixel not composited: got %v want %v", got, red)
	}
	if got := (*last).RGBAAt(0, 0); got != white {
		t.Fatalf("view pixel outside popup changed: got %v", got)
	}
}

func TestPopupHideRestoresView(t *testing.T) {
	rh, last := captureHandler(t, 10, 10)
	white := color.RGBA{255, 255, 255, 255}
	red := color.RGBA{255, 0, 0, 255}

	fill(rh.image.(*image.RGBA), rh.image.Bounds(), white)
	rh.PopupSize(image.Rect(2, 2, 6, 6))
	rh.PopupShow(true)
	tile := image.NewRGBA(image.Rect(0, 0, 4, 4))
	fill(tile, tile.Bounds(), red)
	rh.DoPaintPopup([]DirtyArea{{Rectangle: tile.Bounds(), Buffer: tile}})

	rh.PopupShow(false)
	if got := (*last).RGBAAt(3, 3); got != white {
		t.Fatalf("hidden popup left pixels behind: got %v", got)
	}
}

func TestPopupClippedAtViewEdge(t *testing.T) {
	rh, last := captureHandler(t, 10, 10)
	blue := color.RGBA{0, 0, 255, 255}
	green := color.RGBA{0, 255, 0, 255}

	rh.PopupSize(image.Rect(8, 8, 12, 12))
	rh.PopupShow(true)
	tile := image.NewRGBA(image.Rect(0, 0, 4, 4))
	tile.SetRGBA(0, 0, blue)
	tile.SetRGBA(1, 1, green)
	rh.DoPaintPopup([]DirtyArea{{Rectangle: tile.Bounds(), Buffer: tile}})

	// popup-local (0,0) must land on view (8,8), (1,1) on (9,9)
	if got := (*last).RGBAAt(8, 8); got != blue {
		t.Fatalf("popup origin misplaced: got %v", got)
	}
	if got := (*last).RGBAAt(9, 9); got != green {
		t.Fatalf("popup offset misplaced: got %v", got)
	}
}

func TestPopupPaintIgnoredWhileHidden(t *testing.T) {
	rh, last := captureHandler(t, 10, 10)
	red := color.RGBA{255, 0, 0, 255}

	rh.PopupSize(image.Rect(0, 0, 2, 2))
	tile := image.NewRGBA(image.Rect(0, 0, 2, 2))
	fill(tile, tile.Bounds(), red)
	rh.DoPaintPopup([]DirtyArea{{Rectangle: tile.Bounds(), Buffer: tile}})

	if last != nil && *last != nil {
		if got := (*last).RGBAAt(0, 0); got == red {
			t.Fatalf("hidden popup painted into view")
		}
	}
}
