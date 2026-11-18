package cef

/*
#include <string.h>
#include "browser_client_bridge.h"
#include "lifespanhandler_bridge.h"
#include "renderhandler_bridge.h"
*/
import "C"
import (
	"image"
	"image/draw"
	"sync"
	"unsafe"
)

type (
	RenderHandler interface {
		GetViewRect() image.Rectangle
		DoPaint(areas []DirtyArea)
		CursorChanged(cursorType CursorType)
	}

	// ResizableRenderHandler is implemented by render handlers whose surface
	// can change size at runtime. After calling Resize, trigger
	// Browser.Redraw() so CEF picks up the new view rect.
	ResizableRenderHandler interface {
		RenderHandler
		Resize(width, height int)
	}

	// PopupRenderHandler is an optional extension of RenderHandler for the
	// popup widget CEF uses for <select> dropdowns (and similar). CEF reports
	// the popup as a separate surface in view coordinates. A RenderHandler
	// that does not implement this interface never sees popups - they are
	// dropped. A handler that does implement it owns the presentation: the
	// default image handler composites the popup into its image, while a
	// custom handler (e.g. one drawing the list with Fyne) can render it
	// however it likes. Mouse input stays with CEF: forward clicks in view
	// coordinates and Chromium routes them into the popup.
	PopupRenderHandler interface {
		// PopupShow is called when CEF shows (true) or hides (false) the popup.
		PopupShow(show bool)
		// PopupSize is called with the popup's position and size in view
		// coordinates whenever CEF moves or resizes it.
		PopupSize(rect image.Rectangle)
		// DoPaintPopup receives popup pixels. Each area's Rectangle is in
		// popup-local coordinates (origin = popup top-left), Buffer is an
		// RGBA tile of the same size.
		DoPaintPopup(areas []DirtyArea)
	}

	// ScreenInfo describes the screen the view lives on. Rect and
	// AvailableRect are in DIP (view) coordinates.
	ScreenInfo struct {
		DeviceScaleFactor float32
		Rect              image.Rectangle
		AvailableRect     image.Rectangle
	}

	// ScreenInfoRenderHandler is an optional extension of RenderHandler for
	// HiDPI-aware hosts. Without it CEF assumes scale 1 and the view rect as
	// the screen, which renders blurry on scaled displays. Returning ok=false
	// from either method keeps CEF's default for that query.
	ScreenInfoRenderHandler interface {
		GetScreenInfo() (info ScreenInfo, ok bool)
		// GetScreenPoint maps a view-local point to screen coordinates
		// (device pixels on Windows/Linux).
		GetScreenPoint(viewX, viewY int) (screenX, screenY int, ok bool)
	}

	imageRenderHandler struct {
		mu       sync.Mutex
		image    draw.Image
		callback func(image.Image)

		popupShown bool
		popupRect  image.Rectangle
		popupImage *image.RGBA
	}
	DirtyArea struct {
		Rectangle image.Rectangle
		Buffer    image.Image
	}
	CursorType int
)

// Explicitly mapped to cef_cursor_type_t so inserted enum values in future
// CEF versions cannot silently shift the constants.
const (
	CursorTypePOINTER                  = CursorType(C.CT_POINTER)
	CursorTypeCROSS                    = CursorType(C.CT_CROSS)
	CursorTypeHAND                     = CursorType(C.CT_HAND)
	CursorTypeIBEAM                    = CursorType(C.CT_IBEAM)
	CursorTypeWAIT                     = CursorType(C.CT_WAIT)
	CursorTypeHELP                     = CursorType(C.CT_HELP)
	CursorTypeEASTRESIZE               = CursorType(C.CT_EASTRESIZE)
	CursorTypeNORTHRESIZE              = CursorType(C.CT_NORTHRESIZE)
	CursorTypeNORTHEASTRESIZE          = CursorType(C.CT_NORTHEASTRESIZE)
	CursorTypeNORTHWESTRESIZE          = CursorType(C.CT_NORTHWESTRESIZE)
	CursorTypeSOUTHRESIZE              = CursorType(C.CT_SOUTHRESIZE)
	CursorTypeSOUTHEASTRESIZE          = CursorType(C.CT_SOUTHEASTRESIZE)
	CursorTypeSOUTHWESTRESIZE          = CursorType(C.CT_SOUTHWESTRESIZE)
	CursorTypeWESTRESIZE               = CursorType(C.CT_WESTRESIZE)
	CursorTypeNORTHSOUTHRESIZE         = CursorType(C.CT_NORTHSOUTHRESIZE)
	CursorTypeEASTWESTRESIZE           = CursorType(C.CT_EASTWESTRESIZE)
	CursorTypeNORTHEASTSOUTHWESTRESIZE = CursorType(C.CT_NORTHEASTSOUTHWESTRESIZE)
	CursorTypeNORTHWESTSOUTHEASTRESIZE = CursorType(C.CT_NORTHWESTSOUTHEASTRESIZE)
	CursorTypeCOLUMNRESIZE             = CursorType(C.CT_COLUMNRESIZE)
	CursorTypeROWRESIZE                = CursorType(C.CT_ROWRESIZE)
	CursorTypeMIDDLEPANNING            = CursorType(C.CT_MIDDLEPANNING)
	CursorTypeEASTPANNING              = CursorType(C.CT_EASTPANNING)
	CursorTypeNORTHPANNING             = CursorType(C.CT_NORTHPANNING)
	CursorTypeNORTHEASTPANNING         = CursorType(C.CT_NORTHEASTPANNING)
	CursorTypeNORTHWESTPANNING         = CursorType(C.CT_NORTHWESTPANNING)
	CursorTypeSOUTHPANNING             = CursorType(C.CT_SOUTHPANNING)
	CursorTypeSOUTHEASTPANNING         = CursorType(C.CT_SOUTHEASTPANNING)
	CursorTypeSOUTHWESTPANNING         = CursorType(C.CT_SOUTHWESTPANNING)
	CursorTypeWESTPANNING              = CursorType(C.CT_WESTPANNING)
	CursorTypeMOVE                     = CursorType(C.CT_MOVE)
	CursorTypeVERTICALTEXT             = CursorType(C.CT_VERTICALTEXT)
	CursorTypeCELL                     = CursorType(C.CT_CELL)
	CursorTypeCONTEXTMENU              = CursorType(C.CT_CONTEXTMENU)
	CursorTypeALIAS                    = CursorType(C.CT_ALIAS)
	CursorTypePROGRESS                 = CursorType(C.CT_PROGRESS)
	CursorTypeNODROP                   = CursorType(C.CT_NODROP)
	CursorTypeCOPY                     = CursorType(C.CT_COPY)
	CursorTypeNONE                     = CursorType(C.CT_NONE)
	CursorTypeNOTALLOWED               = CursorType(C.CT_NOTALLOWED)
	CursorTypeZOOMIN                   = CursorType(C.CT_ZOOMIN)
	CursorTypeZOOMOUT                  = CursorType(C.CT_ZOOMOUT)
	CursorTypeGRAB                     = CursorType(C.CT_GRAB)
	CursorTypeGRABBING                 = CursorType(C.CT_GRABBING)
	CursorTypeCUSTOM                   = CursorType(C.CT_CUSTOM)
)

//export goGetRenderHandler
func goGetRenderHandler(ptr *C.cef_client_t) (result *C.cef_render_handler_t) {

	c := getClient(ptr)
	if c == nil {
		return nil
	}

	c.Lock()
	defer c.Unlock()

	// only offer a render handler when the user actually configured one -
	// windowed browsers must not advertise OSR support
	if c.renderHandler == nil || c.renderHandlerPtr == nil {
		return nil
	}
	addRefBase(unsafe.Pointer(c.renderHandlerPtr))
	return c.renderHandlerPtr
}

func renderHandler(handler *C.cef_render_handler_t) RenderHandler {
	app := getInstance()
	if app == nil {
		return nil
	}

	app.clientMutex.Lock()
	defer app.clientMutex.Unlock()

	for _, c := range app.clients {
		c.Lock()
		match := c.renderHandlerPtr == handler
		h := c.renderHandler
		c.Unlock()
		if match {
			return h
		}
	}

	return nil
}

//export goRenderHandlerGetViewRect
func goRenderHandlerGetViewRect(
	handler *C.cef_render_handler_t,
	cefBrowser *C.cef_browser_t,
	rect *C.cef_rect_t,
) C.int {
	defer releaseBase(unsafe.Pointer(cefBrowser))

	result := C.int(0)

	if h := renderHandler(handler); h != nil {
		imgRect := h.GetViewRect()

		*rect = C.cef_rect_t{}
		C.cBridgeSetCefRectDimensions(rect, (C.int)(imgRect.Min.X), (C.int)(imgRect.Min.Y), (C.int)(imgRect.Dx()), (C.int)(imgRect.Dy()))

		result = C.int(1)
	}

	return result
}

//export goRenderHandlerOnPaint
func goRenderHandlerOnPaint(
	handler *C.cef_render_handler_t,
	cefBrowser *C.cef_browser_t,
	t C.cef_paint_element_type_t,
	dirtyRectsCount C.size_t,
	dirtyRects *C.cef_rect_t,
	buffer unsafe.Pointer,
	width C.int,
	height C.int,
) {
	defer releaseBase(unsafe.Pointer(cefBrowser))

	h := renderHandler(handler)
	if h == nil {
		return
	}

	w, ht := int(width), int(height)
	if w <= 0 || ht <= 0 || buffer == nil || dirtyRectsCount == 0 {
		return
	}
	dirtyAreas := copyDirtyTiles(buffer, w, ht, dirtyRects, int(dirtyRectsCount))
	if len(dirtyAreas) == 0 {
		return
	}

	switch t {
	case C.PET_VIEW:
		h.DoPaint(dirtyAreas)
	case C.PET_POPUP:
		// popup paints use popup-local coordinates and must not touch the
		// main surface. Only handlers that opted in via PopupRenderHandler
		// get them; everyone else keeps the historical drop.
		if ph, ok := h.(PopupRenderHandler); ok {
			ph.DoPaintPopup(dirtyAreas)
		}
	}
}

// copyDirtyTiles copies the dirty rectangles out of a CEF buffer into
// standalone RGBA tiles. CEF hands us a premultiplied BGRA buffer with
// upper-left origin; the bytes are copied row-wise, swapping B and R - no
// alpha conversion may happen here. Rectangles are in buffer pixels.
func copyDirtyTiles(buffer unsafe.Pointer, w, ht int, dirtyRects *C.cef_rect_t, count int) []DirtyArea {
	const bytesPerPixel = 4
	src := unsafe.Slice((*byte)(buffer), bytesPerPixel*w*ht)
	rects := unsafe.Slice(dirtyRects, count)

	dirtyAreas := make([]DirtyArea, 0, len(rects))
	for _, dirty := range rects {
		x, y := int(dirty.x), int(dirty.y)
		dw, dh := int(dirty.width), int(dirty.height)
		if x < 0 || y < 0 || dw <= 0 || dh <= 0 || x+dw > w || y+dh > ht {
			continue
		}

		tile := image.NewRGBA(image.Rect(0, 0, dw, dh))
		for row := 0; row < dh; row++ {
			srcOffset := ((y+row)*w + x) * bytesPerPixel
			dstOffset := row * tile.Stride
			copy(tile.Pix[dstOffset:dstOffset+dw*bytesPerPixel], src[srcOffset:srcOffset+dw*bytesPerPixel])
		}
		for pixelIdx := 0; pixelIdx < len(tile.Pix); pixelIdx += bytesPerPixel {
			// BGRA => RGBA
			tile.Pix[pixelIdx], tile.Pix[pixelIdx+2] = tile.Pix[pixelIdx+2], tile.Pix[pixelIdx]
		}

		dirtyAreas = append(dirtyAreas, DirtyArea{
			Rectangle: image.Rect(x, y, x+dw, y+dh),
			Buffer:    tile,
		})
	}
	return dirtyAreas
}

//export goRenderHandlerOnPopupShow
func goRenderHandlerOnPopupShow(handler *C.cef_render_handler_t, cefBrowser *C.cef_browser_t, show C.int) {
	defer releaseBase(unsafe.Pointer(cefBrowser))

	if ph, ok := renderHandler(handler).(PopupRenderHandler); ok {
		ph.PopupShow(show != 0)
	}
}

//export goRenderHandlerOnPopupSize
func goRenderHandlerOnPopupSize(handler *C.cef_render_handler_t, cefBrowser *C.cef_browser_t, rect *C.cef_rect_t) {
	defer releaseBase(unsafe.Pointer(cefBrowser))

	if rect == nil {
		return
	}
	if ph, ok := renderHandler(handler).(PopupRenderHandler); ok {
		ph.PopupSize(image.Rect(int(rect.x), int(rect.y), int(rect.x+rect.width), int(rect.y+rect.height)))
	}
}

func (r *imageRenderHandler) CursorChanged(cursorType CursorType) {
}

func (r *imageRenderHandler) DoPaint(areas []DirtyArea) {
	r.mu.Lock()

	for _, area := range areas {
		// tiles are (0,0)-based - the source point must be the zero point,
		// not the dirty rect's position
		draw.Draw(r.image, area.Rectangle, area.Buffer, image.Point{}, draw.Src)
	}

	snapshot := r.snapshotLocked()
	callback := r.callback
	r.mu.Unlock()

	if callback != nil {
		callback(snapshot)
	}
}

// snapshotLocked builds a fresh copy of the view with the popup composited on
// top when it is shown. The consumer never observes the live surface
// mid-update, and hiding the popup leaves no stale pixels behind because the
// overlay is rebuilt from the view every time. Caller holds r.mu.
func (r *imageRenderHandler) snapshotLocked() image.Image {
	bounds := r.image.Bounds()
	snapshot := image.NewRGBA(bounds)
	draw.Draw(snapshot, bounds, r.image, bounds.Min, draw.Src)

	if r.popupShown && r.popupImage != nil {
		dst := r.popupRect.Intersect(bounds)
		if !dst.Empty() {
			// source point is the popup-local position matching dst.Min
			sp := dst.Min.Sub(r.popupRect.Min)
			draw.Draw(snapshot, dst, r.popupImage, sp, draw.Over)
		}
	}
	return snapshot
}

func (r *imageRenderHandler) PopupShow(show bool) {
	r.mu.Lock()
	r.popupShown = show
	snapshot := r.snapshotLocked()
	callback := r.callback
	r.mu.Unlock()

	if callback != nil {
		callback(snapshot)
	}
}

func (r *imageRenderHandler) PopupSize(rect image.Rectangle) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.popupRect = rect
	if rect.Empty() {
		r.popupImage = nil
		return
	}
	if r.popupImage == nil || r.popupImage.Bounds().Size() != rect.Size() {
		r.popupImage = image.NewRGBA(image.Rect(0, 0, rect.Dx(), rect.Dy()))
	}
}

func (r *imageRenderHandler) DoPaintPopup(areas []DirtyArea) {
	r.mu.Lock()

	if r.popupImage == nil {
		r.mu.Unlock()
		return
	}
	for _, area := range areas {
		draw.Draw(r.popupImage, area.Rectangle, area.Buffer, image.Point{}, draw.Src)
	}

	var snapshot image.Image
	if r.popupShown {
		snapshot = r.snapshotLocked()
	}
	callback := r.callback
	r.mu.Unlock()

	if snapshot != nil && callback != nil {
		callback(snapshot)
	}
}

func (r *imageRenderHandler) GetViewRect() image.Rectangle {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.image.Bounds()
}

// Resize swaps the surface to the new dimensions, keeping the previous
// content where it overlaps. Call Browser.Redraw() afterwards so CEF
// repaints at the new size.
func (r *imageRenderHandler) Resize(width, height int) {
	if width <= 0 || height <= 0 {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	newImage := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(newImage, r.image.Bounds().Intersect(newImage.Bounds()), r.image, image.Point{}, draw.Src)
	r.image = newImage
}

func NewImageRenderHandler(img draw.Image, imageChanged func(image.Image)) RenderHandler {
	return &imageRenderHandler{
		image:    img,
		callback: imageChanged,
	}
}

var (
	_ ResizableRenderHandler = (*imageRenderHandler)(nil)
	_ PopupRenderHandler     = (*imageRenderHandler)(nil)
)

//export goRenderHandlerGetScreenInfo
func goRenderHandlerGetScreenInfo(handler *C.cef_render_handler_t, cefBrowser *C.cef_browser_t, info *C.cef_screen_info_t) C.int {
	defer releaseBase(unsafe.Pointer(cefBrowser))

	if info == nil {
		return 0
	}
	sh, ok := renderHandler(handler).(ScreenInfoRenderHandler)
	if !ok {
		return 0
	}
	si, ok := sh.GetScreenInfo()
	if !ok {
		return 0
	}
	info.device_scale_factor = C.float(si.DeviceScaleFactor)
	info.depth = 32
	info.depth_per_component = 8
	info.is_monochrome = 0
	info.rect = C.cef_rect_t{x: C.int(si.Rect.Min.X), y: C.int(si.Rect.Min.Y), width: C.int(si.Rect.Dx()), height: C.int(si.Rect.Dy())}
	info.available_rect = C.cef_rect_t{x: C.int(si.AvailableRect.Min.X), y: C.int(si.AvailableRect.Min.Y), width: C.int(si.AvailableRect.Dx()), height: C.int(si.AvailableRect.Dy())}
	return 1
}

//export goRenderHandlerGetScreenPoint
func goRenderHandlerGetScreenPoint(handler *C.cef_render_handler_t, cefBrowser *C.cef_browser_t, viewX, viewY C.int, screenX, screenY *C.int) C.int {
	defer releaseBase(unsafe.Pointer(cefBrowser))

	if screenX == nil || screenY == nil {
		return 0
	}
	sh, ok := renderHandler(handler).(ScreenInfoRenderHandler)
	if !ok {
		return 0
	}
	sx, sy, ok := sh.GetScreenPoint(int(viewX), int(viewY))
	if !ok {
		return 0
	}
	*screenX = C.int(sx)
	*screenY = C.int(sy)
	return 1
}
