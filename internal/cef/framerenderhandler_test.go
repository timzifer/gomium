package cef

import (
	"bytes"
	"image"
	"image/color"
	"testing"
)

func tileOf(w, h int, c color.RGBA) *image.RGBA {
	t := image.NewRGBA(image.Rect(0, 0, w, h))
	fill(t, t.Bounds(), c)
	return t
}

// renderPair drives the snapshot handler and the frame handler through the
// same calls; after each step the frame handler's view must equal the
// snapshot.
type renderPair struct {
	t        *testing.T
	snap     *imageRenderHandler
	last     **image.RGBA
	frame    *frameRenderHandler
	frameImg *image.RGBA
	dirty    []image.Rectangle
}

func newRenderPair(t *testing.T, w, h int) *renderPair {
	p := &renderPair{t: t}
	p.snap, p.last = captureHandler(t, w, h)
	p.frame = NewFrameRenderHandler(w, h, func(f *image.RGBA, d []image.Rectangle) {
		// copy: both are borrowed
		p.frameImg = image.NewRGBA(f.Bounds())
		copy(p.frameImg.Pix, f.Pix)
		p.dirty = append([]image.Rectangle(nil), d...)
	}).(*frameRenderHandler)
	return p
}

func (p *renderPair) both(step string, fn func(h interface {
	RenderHandler
	PopupRenderHandler
})) {
	p.t.Helper()
	fn(p.snap)
	fn(p.frame)
	if *p.last == nil || p.frameImg == nil {
		p.t.Fatalf("%s: no frame delivered (snapshot %v, frame %v)", step, *p.last != nil, p.frameImg != nil)
	}
	if !bytes.Equal((*p.last).Pix, p.frameImg.Pix) {
		p.t.Fatalf("%s: frame handler differs from the snapshot handler", step)
	}
}

func TestFrameRenderHandlerMatchesSnapshotHandler(t *testing.T) {
	white := color.RGBA{255, 255, 255, 255}
	red := color.RGBA{255, 0, 0, 255}
	blue := color.RGBA{0, 0, 255, 255}
	type handler = interface {
		RenderHandler
		PopupRenderHandler
	}
	p := newRenderPair(t, 20, 20)

	p.both("full paint", func(h handler) {
		h.DoPaint([]DirtyArea{{Rectangle: image.Rect(0, 0, 20, 20), Buffer: tileOf(20, 20, white)}})
	})
	p.both("partial paint", func(h handler) {
		h.DoPaint([]DirtyArea{{Rectangle: image.Rect(3, 4, 7, 9), Buffer: tileOf(4, 5, blue)}})
	})
	if len(p.dirty) != 1 || p.dirty[0] != image.Rect(3, 4, 7, 9) {
		t.Fatalf("dirty = %v, want the painted rect", p.dirty)
	}
	p.both("popup shown", func(h handler) {
		h.PopupSize(image.Rect(5, 5, 15, 15))
		h.PopupShow(true)
		h.DoPaintPopup([]DirtyArea{{Rectangle: image.Rect(0, 0, 10, 10), Buffer: tileOf(10, 10, red)}})
	})
	p.both("view painted under the popup", func(h handler) {
		h.DoPaint([]DirtyArea{{Rectangle: image.Rect(0, 0, 20, 20), Buffer: tileOf(20, 20, blue)}})
	})
	p.both("popup moved", func(h handler) {
		h.PopupSize(image.Rect(10, 10, 20, 20))
		h.DoPaintPopup([]DirtyArea{{Rectangle: image.Rect(0, 0, 10, 10), Buffer: tileOf(10, 10, red)}})
	})
	p.both("popup hidden", func(h handler) {
		h.PopupShow(false)
	})
	if p.frame.out != nil {
		t.Fatal("popup buffer kept after hide")
	}
}

func TestFrameRenderHandlerPaintsWithoutAllocating(t *testing.T) {
	rh := NewFrameRenderHandler(1920, 1080, func(*image.RGBA, []image.Rectangle) {})
	areas := []DirtyArea{{Rectangle: image.Rect(100, 100, 116, 116), Buffer: tileOf(16, 16, color.RGBA{1, 2, 3, 255})}}
	rh.DoPaint(areas) // grows the dirty slice once
	if n := testing.AllocsPerRun(100, func() { rh.DoPaint(areas) }); n != 0 {
		t.Fatalf("DoPaint allocates %.0f times per call", n)
	}
}

func benchmarkPaint(b *testing.B, rh RenderHandler) {
	areas := []DirtyArea{{Rectangle: image.Rect(100, 100, 116, 116), Buffer: tileOf(16, 16, color.RGBA{1, 2, 3, 255})}}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rh.DoPaint(areas)
	}
}

// A blinking cursor on a 1080p view: one 16x16 dirty rect per paint.
func BenchmarkImageRenderHandlerSmallDirty1080p(b *testing.B) {
	benchmarkPaint(b, NewImageRenderHandler(image.NewRGBA(image.Rect(0, 0, 1920, 1080)), func(image.Image) {}))
}

func BenchmarkFrameRenderHandlerSmallDirty1080p(b *testing.B) {
	benchmarkPaint(b, NewFrameRenderHandler(1920, 1080, func(*image.RGBA, []image.Rectangle) {}))
}
