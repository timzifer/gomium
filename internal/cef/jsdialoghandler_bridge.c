#include "jsdialoghandler_bridge.h"
#include "_cgo_export.h"

static int CEF_CALLBACK on_jsdialog_trampoline(struct _cef_jsdialog_handler_t* self,
                                               struct _cef_browser_t* browser,
                                               const cef_string_t* origin_url,
                                               cef_jsdialog_type_t dialog_type,
                                               const cef_string_t* message_text,
                                               const cef_string_t* default_prompt_text,
                                               struct _cef_jsdialog_callback_t* callback,
                                               int* suppress_message) {
    return goJsDialogHandlerOnJsDialog(self, browser,
                                       (cef_string_t*)origin_url, dialog_type,
                                       (cef_string_t*)message_text, (cef_string_t*)default_prompt_text,
                                       callback, suppress_message);
}

static int CEF_CALLBACK on_before_unload_dialog_trampoline(struct _cef_jsdialog_handler_t* self,
                                                           struct _cef_browser_t* browser,
                                                           const cef_string_t* message_text,
                                                           int is_reload,
                                                           struct _cef_jsdialog_callback_t* callback) {
    return goJsDialogHandlerOnBeforeUnloadDialog(self, browser, (cef_string_t*)message_text, is_reload, callback);
}

struct _cef_jsdialog_handler_t* cJsDialogHandlerCreate() { DEBUG_CALL;
    cef_jsdialog_handler_t* handler = (cef_jsdialog_handler_t*)calloc(1, sizeof(cef_jsdialog_handler_t));
    handler -> base.size = sizeof(cef_jsdialog_handler_t);
    initialize_cef_base_ref_counted((cef_base_ref_counted_t*)handler);

    handler -> on_jsdialog = on_jsdialog_trampoline;
    handler -> on_before_unload_dialog = on_before_unload_dialog_trampoline;

    return handler;
}

void cJsDialogHandlerFree(struct _cef_jsdialog_handler_t* handler) { DEBUG_CALL;
    ((cef_base_ref_counted_t*)handler)->release((cef_base_ref_counted_t*)handler);
}

void cJsDialogCallbackContinue(struct _cef_jsdialog_callback_t* callback,
                               int success,
                               cef_string_t* user_input) {
    callback->cont(callback, success, user_input);
}
