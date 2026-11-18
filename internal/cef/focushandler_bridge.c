#include "focushandler_bridge.h"
#include "_cgo_export.h"

static void CEF_CALLBACK on_take_focus_trampoline(struct _cef_focus_handler_t* self,
                                                  struct _cef_browser_t* browser,
                                                  int next) {
    goFocusHandlerOnTakeFocus(self, browser, next);
}

static int CEF_CALLBACK on_set_focus_trampoline(struct _cef_focus_handler_t* self,
                                                struct _cef_browser_t* browser,
                                                cef_focus_source_t source) {
    return goFocusHandlerOnSetFocus(self, browser, source);
}

static void CEF_CALLBACK on_got_focus_trampoline(struct _cef_focus_handler_t* self,
                                                 struct _cef_browser_t* browser) {
    goFocusHandlerOnGotFocus(self, browser);
}

struct _cef_focus_handler_t* cFocusHandlerCreate() { DEBUG_CALL;
    cef_focus_handler_t* handler = (cef_focus_handler_t*)calloc(1, sizeof(cef_focus_handler_t));
    handler -> base.size = sizeof(cef_focus_handler_t);
    initialize_cef_base_ref_counted((cef_base_ref_counted_t*)handler);

    handler -> on_take_focus = on_take_focus_trampoline;
    handler -> on_set_focus  = on_set_focus_trampoline;
    handler -> on_got_focus  = on_got_focus_trampoline;

    return handler;
}

void cFocusHandlerFree(struct _cef_focus_handler_t* handler) { DEBUG_CALL;
    ((cef_base_ref_counted_t*)handler)->release((cef_base_ref_counted_t*)handler);
}
