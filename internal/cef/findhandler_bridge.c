#include "findhandler_bridge.h"
#include "_cgo_export.h"

static void CEF_CALLBACK on_find_result_trampoline(struct _cef_find_handler_t* self,
                                                   struct _cef_browser_t* browser,
                                                   int identifier,
                                                   int count,
                                                   const cef_rect_t* selectionRect,
                                                   int activeMatchOrdinal,
                                                   int finalUpdate) {
    goFindHandlerOnFindResult(self, browser, identifier, count, activeMatchOrdinal, finalUpdate);
}

struct _cef_find_handler_t* cFindHandlerCreate() { DEBUG_CALL;
    cef_find_handler_t* handler = (cef_find_handler_t*)calloc(1, sizeof(cef_find_handler_t));
    handler -> base.size = sizeof(cef_find_handler_t);
    initialize_cef_base_ref_counted((cef_base_ref_counted_t*)handler);

    handler -> on_find_result = on_find_result_trampoline;

    return handler;
}

void cFindHandlerFree(struct _cef_find_handler_t* handler) { DEBUG_CALL;
    ((cef_base_ref_counted_t*)handler)->release((cef_base_ref_counted_t*)handler);
}
