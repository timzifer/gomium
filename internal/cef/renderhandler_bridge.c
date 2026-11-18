#include "renderhandler_bridge.h"
#include "_cgo_export.h"

static void CEF_CALLBACK on_paint_trampoline(struct _cef_render_handler_t* self,
                               struct _cef_browser_t* browser,
                               cef_paint_element_type_t type,
                               size_t dirtyRectsCount,
                               cef_rect_t const* dirtyRects,
                               const void* buffer,
                               int width,
                               int height) {
    goRenderHandlerOnPaint(self, browser, type, dirtyRectsCount, (cef_rect_t*)dirtyRects, (void*)buffer, width, height);
}

static void CEF_CALLBACK on_popup_show_trampoline(struct _cef_render_handler_t* self,
                                    struct _cef_browser_t* browser,
                                    int show) {
    goRenderHandlerOnPopupShow(self, browser, show);
}

static void CEF_CALLBACK on_popup_size_trampoline(struct _cef_render_handler_t* self,
                                    struct _cef_browser_t* browser,
                                    const cef_rect_t* rect) {
    goRenderHandlerOnPopupSize(self, browser, (cef_rect_t*)rect);
}

static int CEF_CALLBACK get_screen_info_trampoline(struct _cef_render_handler_t* self,
                                                   struct _cef_browser_t* browser,
                                                   cef_screen_info_t* screen_info) {
    return goRenderHandlerGetScreenInfo(self, browser, screen_info);
}

static int CEF_CALLBACK get_screen_point_trampoline(struct _cef_render_handler_t* self,
                                                    struct _cef_browser_t* browser,
                                                    int viewX,
                                                    int viewY,
                                                    int* screenX,
                                                    int* screenY) {
    return goRenderHandlerGetScreenPoint(self, browser, viewX, viewY, screenX, screenY);
}

static void CEF_CALLBACK get_view_rect_trampoline(struct _cef_render_handler_t* self,
                                              struct _cef_browser_t* browser,
                                              cef_rect_t* rect) {
    // the rect MUST be non-empty since CEF made get_view_rect void
    if (!goRenderHandlerGetViewRect(self, browser, rect) || rect->width <= 0 || rect->height <= 0) {
        rect->x = 0;
        rect->y = 0;
        rect->width = 800;
        rect->height = 600;
    }
}

struct _cef_render_handler_t* cRenderHandlerCreate() { DEBUG_CALL;
    cef_render_handler_t* handler = (cef_render_handler_t*)calloc(1, sizeof(cef_render_handler_t));
    handler -> base.size = sizeof(cef_render_handler_t);
    initialize_cef_base_ref_counted((cef_base_ref_counted_t*)handler);

    handler->get_view_rect         = get_view_rect_trampoline;
    handler->on_paint              = on_paint_trampoline;
    handler->on_popup_show         = on_popup_show_trampoline;
    handler->on_popup_size         = on_popup_size_trampoline;
    handler->get_screen_info       = get_screen_info_trampoline;
    handler->get_screen_point      = get_screen_point_trampoline;

    return handler;
}
void cRenderHandlerFree(struct _cef_render_handler_t* handler) { DEBUG_CALL;
    ((cef_base_ref_counted_t*)handler)->release((cef_base_ref_counted_t*)handler);
}

void cBridgeSetCefRectDimensions(cef_rect_t * rect, int x, int y, int width, int height) {
      rect->x = x;
      rect->y = y;
      rect->width = width;
      rect->height = height;
}
