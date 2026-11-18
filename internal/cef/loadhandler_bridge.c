#include "loadhandler_bridge.h"
#include "_cgo_export.h"

static void CEF_CALLBACK on_loading_state_change_trampoline(struct _cef_load_handler_t* self,
                                                            struct _cef_browser_t* browser,
                                                            int isLoading,
                                                            int canGoBack,
                                                            int canGoForward) {
    goLoadHandlerOnLoadingStateChange(self, browser, isLoading, canGoBack, canGoForward);
}

static void CEF_CALLBACK on_load_start_trampoline(struct _cef_load_handler_t* self,
                                                  struct _cef_browser_t* browser,
                                                  struct _cef_frame_t* frame,
                                                  cef_transition_type_t transition_type) {
    goLoadHandlerOnLoadStart(self, browser, frame);
}

static void CEF_CALLBACK on_load_end_trampoline(struct _cef_load_handler_t* self,
                                                struct _cef_browser_t* browser,
                                                struct _cef_frame_t* frame,
                                                int httpStatusCode) {
    goLoadHandlerOnLoadEnd(self, browser, frame, httpStatusCode);
}

static void CEF_CALLBACK on_load_error_trampoline(struct _cef_load_handler_t* self,
                                                  struct _cef_browser_t* browser,
                                                  struct _cef_frame_t* frame,
                                                  cef_errorcode_t errorCode,
                                                  const cef_string_t* errorText,
                                                  const cef_string_t* failedUrl) {
    goLoadHandlerOnLoadError(self, browser, frame, errorCode, (cef_string_t*)errorText, (cef_string_t*)failedUrl);
}

struct _cef_load_handler_t* cLoadHandlerCreate() { DEBUG_CALL;
    cef_load_handler_t* handler = (cef_load_handler_t*)calloc(1, sizeof(cef_load_handler_t));
    handler -> base.size = sizeof(cef_load_handler_t);
    initialize_cef_base_ref_counted((cef_base_ref_counted_t*)handler);

    handler -> on_loading_state_change = on_loading_state_change_trampoline;
    handler -> on_load_start = on_load_start_trampoline;
    handler -> on_load_end = on_load_end_trampoline;
    handler -> on_load_error = on_load_error_trampoline;

    return handler;
}

void cLoadHandlerFree(struct _cef_load_handler_t* handler) { DEBUG_CALL;
    ((cef_base_ref_counted_t*)handler)->release((cef_base_ref_counted_t*)handler);
}
