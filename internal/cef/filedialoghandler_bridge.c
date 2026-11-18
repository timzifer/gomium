#include "filedialoghandler_bridge.h"
#include "_cgo_export.h"

static int CEF_CALLBACK on_file_dialog_trampoline(struct _cef_dialog_handler_t* self,
                                                  struct _cef_browser_t* browser,
                                                  cef_file_dialog_mode_t mode,
                                                  const cef_string_t* title,
                                                  const cef_string_t* default_file_path,
                                                  cef_string_list_t accept_filters,
                                                  cef_string_list_t accept_extensions,
                                                  cef_string_list_t accept_descriptions,
                                                  struct _cef_file_dialog_callback_t* callback) {
    return goFileDialogHandlerOnFileDialog(self, browser, mode, (cef_string_t*)title,
                                           (cef_string_t*)default_file_path, accept_filters, callback);
}

void cBridgeFileDialogContinue(struct _cef_file_dialog_callback_t* cb, cef_string_list_t paths) {
    cb->cont(cb, paths);
}

void cBridgeFileDialogCancel(struct _cef_file_dialog_callback_t* cb) {
    cb->cancel(cb);
}

struct _cef_dialog_handler_t* cFileDialogHandlerCreate() { DEBUG_CALL;
    cef_dialog_handler_t* handler = (cef_dialog_handler_t*)calloc(1, sizeof(cef_dialog_handler_t));
    handler -> base.size = sizeof(cef_dialog_handler_t);
    initialize_cef_base_ref_counted((cef_base_ref_counted_t*)handler);

    handler -> on_file_dialog = on_file_dialog_trampoline;

    return handler;
}

void cFileDialogHandlerFree(struct _cef_dialog_handler_t* handler) { DEBUG_CALL;
    ((cef_base_ref_counted_t*)handler)->release((cef_base_ref_counted_t*)handler);
}
