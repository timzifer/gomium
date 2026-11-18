#include "lifespanhandler_bridge.h"
#include "_cgo_export.h"

static void CEF_CALLBACK on_after_created_trampoline(struct _cef_life_span_handler_t* self, struct _cef_browser_t* browser) {
    goOnAfterCreated(self, browser);
}

static int CEF_CALLBACK do_close_trampoline(struct _cef_life_span_handler_t* self, struct _cef_browser_t* browser) {
    return goDoClose(self, browser);
}

static void CEF_CALLBACK on_before_close_trampoline(struct _cef_life_span_handler_t* self, struct _cef_browser_t* browser) {
    goOnBeforeClose(self, browser);
}

static int CEF_CALLBACK on_before_popup_trampoline(struct _cef_life_span_handler_t* self,
                                                   struct _cef_browser_t* browser,
                                                   struct _cef_frame_t* frame,
                                                   int popup_id,
                                                   const cef_string_t* target_url,
                                                   const cef_string_t* target_frame_name,
                                                   cef_window_open_disposition_t target_disposition,
                                                   int user_gesture,
                                                   const cef_popup_features_t* popupFeatures,
                                                   struct _cef_window_info_t* windowInfo,
                                                   struct _cef_client_t** client,
                                                   struct _cef_browser_settings_t* settings,
                                                   struct _cef_dictionary_value_t** extra_info,
                                                   int* no_javascript_access) {
    return goOnBeforePopup(self, browser, frame,
                           (cef_string_t*)target_url, (cef_string_t*)target_frame_name,
                           target_disposition, user_gesture,
                           (cef_window_info_t*)windowInfo, (cef_client_t**)client);
}

struct _cef_life_span_handler_t* cLifeSpanHandlerCreate() { DEBUG_CALL;
    cef_life_span_handler_t* handler = (cef_life_span_handler_t*)calloc(1, sizeof(cef_life_span_handler_t));
    handler -> base.size = sizeof(cef_life_span_handler_t);
    initialize_cef_base_ref_counted((cef_base_ref_counted_t*)handler);

    handler -> on_after_created = on_after_created_trampoline;
    handler -> do_close = do_close_trampoline;
    handler -> on_before_close = on_before_close_trampoline;
    handler -> on_before_popup = on_before_popup_trampoline;

    return handler;
}
void cLifeSpanHandlerFree(struct _cef_life_span_handler_t* handler) { DEBUG_CALL;
    ((cef_base_ref_counted_t*)handler)->release((cef_base_ref_counted_t*)handler);
}
