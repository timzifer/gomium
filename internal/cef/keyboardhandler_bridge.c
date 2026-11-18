#include "keyboardhandler_bridge.h"
#include "_cgo_export.h"

static int CEF_CALLBACK on_pre_key_event_trampoline(struct _cef_keyboard_handler_t* self,
                                                    struct _cef_browser_t* browser,
                                                    const cef_key_event_t* event,
                                                    cef_event_handle_t os_event,
                                                    int* is_keyboard_shortcut) {
    return goKeyboardHandlerOnPreKeyEvent(self, browser, (cef_key_event_t*)event, is_keyboard_shortcut);
}

static int CEF_CALLBACK on_key_event_trampoline(struct _cef_keyboard_handler_t* self,
                                                struct _cef_browser_t* browser,
                                                const cef_key_event_t* event,
                                                cef_event_handle_t os_event) {
    return goKeyboardHandlerOnKeyEvent(self, browser, (cef_key_event_t*)event);
}

struct _cef_keyboard_handler_t* cKeyboardHandlerCreate() { DEBUG_CALL;
    cef_keyboard_handler_t* handler = (cef_keyboard_handler_t*)calloc(1, sizeof(cef_keyboard_handler_t));
    handler -> base.size = sizeof(cef_keyboard_handler_t);
    initialize_cef_base_ref_counted((cef_base_ref_counted_t*)handler);

    handler -> on_pre_key_event = on_pre_key_event_trampoline;
    handler -> on_key_event     = on_key_event_trampoline;

    return handler;
}

void cKeyboardHandlerFree(struct _cef_keyboard_handler_t* handler) { DEBUG_CALL;
    ((cef_base_ref_counted_t*)handler)->release((cef_base_ref_counted_t*)handler);
}
