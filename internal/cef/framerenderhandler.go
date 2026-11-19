package cef

import (
	"image"
	"image/draw"
	"sync"
)

// frameRenderHandler is the allocation-free counterpart of
// imageRenderHandler. It keeps one surface and hands it to the consumer for
// the duration of the callback instead of copying a snapshot per paint.
//
// Without a popup the surface the consumer sees is the view itself: a paint
// copies the dirty tiles in and nothing else. While CEF shows a popup the
// consumer gets a second buffer, view plus popup, kept in sync per dirty
// rect - the view stays free of popup pixels, so hiding the popup needs no
// repaint from CEF.
type frameRenderHandler struct {
	mu      sync.Mutex
	view    *image.RGBA
	out     *image.RGBA // view + popup while the popup is shown; nil otherwise
	onFrame func(frame *image.RGBA, dirty []image.Rectangle)

	popupShown bool
	popupRect  image.Rectangle
	popupImage *image.RGBA

	dirty []image.Rectangle // reused between paints
}

// NewFrameRenderHandler returns a RenderHandler that paints into one
// width x height RGBA surface it owns and calls onFrame after every change.
//
// frame and dirty are borrowed: both are valid only until onFrame returns,
// and the next paint reuses them. Copy what you need to keep - typically
// only the dirty rectangles (view coordinates) into a texture or your own
// buffer. In return the handler allocates nothing per paint, which matters
// at 30-60 fps on large or HiDPI views; NewImageRenderHandler copies the
// whole view for every callback instead.
//
// onFrame runs on the CEF UI thread with the handler's lock held: keep it
// short and do not call back into the handler (Resize) from it.
func NewFrameRenderHandler(width, height int, onFrame func(frame *image.RGBA, dirty []image.Rectangle)) RenderHandler {
	if width <= 0 || height <= 0 {
		width, height = 1, 1
	}
	return &frameRenderHandler{
		view:    image.NewRGBA(image.Rect(0, 0, width, height)),
		onFrame: onFrame,
	}
}

func (r *frameRenderHandler) GetViewRect() image.Rectangle {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.view.Bounds()
}

func (r *frameRenderHandler) CursorChanged(CursorType) {}

func (r *frameRenderHandler) DoPaint(areas []DirtyArea) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.dirty = r.dirty[:0]
	bounds := r.view.Bounds()
	for _, area := range areas {
		rect := area.Rectangle.Intersect(bounds)
		if rect.Empty() {
			continue
		}
		draw.Draw(r.view, rect, area.Buffer, rect.Min.Sub(area.Rectangle.Min), draw.Src)
		r.dirty = append(r.dirty, rect)
		if r.out != nil {
			r.composeLocked(rect)
		}
	}
	r.emitLocked()
}

// composeLocked refreshes rect of the popup buffer from the view and draws
// the popup over it where they overlap.
func (r *frameRenderHandler) composeLocked(rect image.Rectangle) {
	draw.Draw(r.out, rect, r.view, rect.Min, draw.Src)
	if r.popupImage == nil {
		return
	}
	dst := rect.Intersect(r.popupRect)
	if !dst.Empty() {
		draw.Draw(r.out, dst, r.popupImage, dst.Min.Sub(r.popupRect.Min), draw.Over)
	}
}

func (r *frameRenderHandler) emitLocked() {
	if r.onFrame == nil || len(r.dirty) == 0 {
		return
	}
	frame := r.view
	if r.out != nil {
		frame = r.out
	}
	r.onFrame(frame, r.dirty)
}

func (r *frameRenderHandler) PopupShow(show bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if show == r.popupShown {
		return
	}
	r.popupShown = show
	r.dirty = r.dirty[:0]
	if show {
		if r.out == nil || r.out.Bounds() != r.view.Bounds() {
			r.out = image.NewRGBA(r.view.Bounds())
		}
		copy(r.out.Pix, r.view.Pix)
		r.markPopupLocked()
		return
	}
	// the view never held popup pixels: switching back to it is the repaint
	r.out = nil
	if rect := r.popupRect.Intersect(r.view.Bounds()); !rect.Empty() {
		r.dirty = append(r.dirty, rect)
	}
	r.emitLocked()
}

// markPopupLocked composes the popup area and reports it dirty.
func (r *frameRenderHandler) markPopupLocked() {
	if r.out == nil {
		return
	}
	rect := r.popupRect.Intersect(r.out.Bounds())
	if rect.Empty() {
		return
	}
	r.composeLocked(rect)
	r.dirty = append(r.dirty, rect)
	r.emitLocked()
}

func (r *frameRenderHandler) PopupSize(rect image.Rectangle) {
	r.mu.Lock()
	defer r.mu.Unlock()

	old := r.popupRect
	r.popupRect = rect
	if rect.Empty() {
		r.popupImage = nil
	} else if r.popupImage == nil || r.popupImage.Bounds().Size() != rect.Size() {
		r.popupImage = image.NewRGBA(image.Rect(0, 0, rect.Dx(), rect.Dy()))
	}
	if r.out == nil {
		return
	}
	// restore what the popup covered before it moved
	r.dirty = r.dirty[:0]
	if prev := old.Intersect(r.out.Bounds()); !prev.Empty() {
		r.composeLocked(prev)
		r.dirty = append(r.dirty, prev)
	}
	r.emitLocked()
}

func (r *frameRenderHandler) DoPaintPopup(areas []DirtyArea) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.popupImage == nil {
		return
	}
	for _, area := range areas {
		draw.Draw(r.popupImage, area.Rectangle, area.Buffer, image.Point{}, draw.Src)
	}
	r.dirty = r.dirty[:0]
	r.markPopupLocked()
}

// Resize swaps the surface to the new dimensions, keeping the previous
// content where it overlaps. Call Browser.Redraw() afterwards so CEF
// repaints at the new size.
func (r *frameRenderHandler) Resize(width, height int) {
	if width <= 0 || height <= 0 {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	view := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(view, r.view.Bounds().Intersect(view.Bounds()), r.view, image.Point{}, draw.Src)
	r.view = view
	if r.out != nil {
		r.out = image.NewRGBA(view.Bounds())
		copy(r.out.Pix, view.Pix)
		if rect := r.popupRect.Intersect(view.Bounds()); !rect.Empty() {
			r.composeLocked(rect)
		}
	}
}

var (
	_ ResizableRenderHandler = (*frameRenderHandler)(nil)
	_ PopupRenderHandler     = (*frameRenderHandler)(nil)
)
