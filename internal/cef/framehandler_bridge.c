#include "framehandler_bridge.h"
#include "_cgo_export.h"

static void CEF_CALLBACK on_frame_created_trampoline(struct _cef_frame_handler_t* self,
                                                     struct _cef_browser_t* browser,
                                                     struct _cef_frame_t* frame) {
    goFrameHandlerOnFrameCreated(self, browser, frame);
}

static void CEF_CALLBACK on_frame_destroyed_trampoline(struct _cef_frame_handler_t* self,
                                                       struct _cef_browser_t* browser,
                                                       struct _cef_frame_t* frame) {
    goFrameHandlerOnFrameDestroyed(self, browser, frame);
}

static void CEF_CALLBACK on_frame_attached_trampoline(struct _cef_frame_handler_t* self,
                                                      struct _cef_browser_t* browser,
                                                      struct _cef_frame_t* frame,
                                                      int reattached) {
    goFrameHandlerOnFrameAttached(self, browser, frame, reattached);
}

static void CEF_CALLBACK on_frame_detached_trampoline(struct _cef_frame_handler_t* self,
                                                      struct _cef_browser_t* browser,
                                                      struct _cef_frame_t* frame) {
    goFrameHandlerOnFrameDetached(self, browser, frame);
}

static void CEF_CALLBACK on_main_frame_changed_trampoline(struct _cef_frame_handler_t* self,
                                                          struct _cef_browser_t* browser,
                                                          struct _cef_frame_t* old_frame,
                                                          struct _cef_frame_t* new_frame) {
    goFrameHandlerOnMainFrameChanged(self, browser, old_frame, new_frame);
}

struct _cef_frame_handler_t* cFrameHandlerCreate() { DEBUG_CALL;
    cef_frame_handler_t* handler = (cef_frame_handler_t*)calloc(1, sizeof(cef_frame_handler_t));
    handler -> base.size = sizeof(cef_frame_handler_t);
    initialize_cef_base_ref_counted((cef_base_ref_counted_t*)handler);

    handler -> on_frame_created      = on_frame_created_trampoline;
    handler -> on_frame_destroyed    = on_frame_destroyed_trampoline;
    handler -> on_frame_attached     = on_frame_attached_trampoline;
    handler -> on_frame_detached     = on_frame_detached_trampoline;
    handler -> on_main_frame_changed = on_main_frame_changed_trampoline;

    return handler;
}

void cFrameHandlerFree(struct _cef_frame_handler_t* handler) { DEBUG_CALL;
    ((cef_base_ref_counted_t*)handler)->release((cef_base_ref_counted_t*)handler);
}
