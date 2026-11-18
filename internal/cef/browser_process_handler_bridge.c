#include "browser_process_handler_bridge.h"
#include "_cgo_export.h"

static void CEF_CALLBACK on_context_initialized_trampoline(struct _cef_browser_process_handler_t* self) {
    goOnContextInitialized(self);
}

static void CEF_CALLBACK on_schedule_message_pump_work_trampoline(struct _cef_browser_process_handler_t* self, int64_t delay_ms) {
    goOnScheduleMessagePumpWork(self, delay_ms);
}

struct _cef_browser_process_handler_t* cBrowserProcessHandlerCreate() { DEBUG_CALL;
    cef_browser_process_handler_t* browser_process_handler = (cef_browser_process_handler_t*)calloc(1, sizeof(cef_browser_process_handler_t));
    browser_process_handler->base.size = sizeof(cef_browser_process_handler_t);
    initialize_cef_base_ref_counted((cef_base_ref_counted_t*)browser_process_handler);

    browser_process_handler->on_context_initialized = on_context_initialized_trampoline;
    browser_process_handler->on_schedule_message_pump_work = on_schedule_message_pump_work_trampoline;

    return browser_process_handler;
}
void cBrowserProcessHandlerFree(struct _cef_browser_process_handler_t* browser_process_handler) { DEBUG_CALL;
    ((cef_base_ref_counted_t*)browser_process_handler)->release((cef_base_ref_counted_t*)browser_process_handler);
}
