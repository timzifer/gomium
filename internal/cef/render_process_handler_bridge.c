#include "render_process_handler_bridge.h"
#include "_cgo_export.h"

static void CEF_CALLBACK on_web_kit_initialized_trampoline(struct _cef_render_process_handler_t* self) {
    goOnWebKitInitialized(self);
}

static int CEF_CALLBACK on_process_message_received_trampoline(struct _cef_render_process_handler_t* self, struct _cef_browser_t* browser, struct _cef_frame_t* frame, cef_process_id_t source_process, struct _cef_process_message_t* message) {
    return goOnRenderProcessMessageReceived(self, browser, frame, source_process, message);
}

static void CEF_CALLBACK on_context_created_trampoline(struct _cef_render_process_handler_t* self, struct _cef_browser_t* browser, struct _cef_frame_t* frame, struct _cef_v8_context_t* context) {
    goOnRenderProcessContextCreated(self, browser, frame, context);
}

static void CEF_CALLBACK on_context_released_trampoline(struct _cef_render_process_handler_t* self, struct _cef_browser_t* browser, struct _cef_frame_t* frame, struct _cef_v8_context_t* context) {
    goOnRenderProcessContextReleased(self, browser, frame, context);
}

struct _cef_render_process_handler_t* cRenderProcessHandlerCreate() { DEBUG_CALL;
    cef_render_process_handler_t* render_process_handler = (cef_render_process_handler_t*)calloc(1, sizeof(cef_render_process_handler_t));
    render_process_handler->base.size = sizeof(cef_render_process_handler_t);
    initialize_cef_base_ref_counted((cef_base_ref_counted_t*)render_process_handler);

    render_process_handler->on_web_kit_initialized = on_web_kit_initialized_trampoline;
    render_process_handler->on_process_message_received = on_process_message_received_trampoline;
    render_process_handler->on_context_created = on_context_created_trampoline;
    render_process_handler->on_context_released = on_context_released_trampoline;

    return render_process_handler;
}
void cRenderProcessHandlerFree(struct _cef_render_process_handler_t* render_process_handler) { DEBUG_CALL;
    ((cef_base_ref_counted_t*)render_process_handler)->release((cef_base_ref_counted_t*)render_process_handler);
}
