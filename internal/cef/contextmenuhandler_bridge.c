#include "contextmenuhandler_bridge.h"
#include "_cgo_export.h"

static int CEF_CALLBACK run_context_menu_trampoline(struct _cef_context_menu_handler_t* self,
                                                    struct _cef_browser_t* browser,
                                                    struct _cef_frame_t* frame,
                                                    struct _cef_context_menu_params_t* params,
                                                    struct _cef_menu_model_t* model,
                                                    struct _cef_run_context_menu_callback_t* callback) {
    return goContextMenuHandlerRunContextMenu(self, browser, frame, params, model, callback);
}

struct _cef_context_menu_handler_t* cContextMenuHandlerCreate() { DEBUG_CALL;
    cef_context_menu_handler_t* handler = (cef_context_menu_handler_t*)calloc(1, sizeof(cef_context_menu_handler_t));
    handler -> base.size = sizeof(cef_context_menu_handler_t);
    initialize_cef_base_ref_counted((cef_base_ref_counted_t*)handler);

    handler -> run_context_menu = run_context_menu_trampoline;

    return handler;
}

void cContextMenuHandlerFree(struct _cef_context_menu_handler_t* handler) { DEBUG_CALL;
    ((cef_base_ref_counted_t*)handler)->release((cef_base_ref_counted_t*)handler);
}

int cContextMenuParamsGetX(struct _cef_context_menu_params_t* params) { return params->get_xcoord(params); }
int cContextMenuParamsGetY(struct _cef_context_menu_params_t* params) { return params->get_ycoord(params); }
int cContextMenuParamsGetTypeFlags(struct _cef_context_menu_params_t* params) { return (int)params->get_type_flags(params); }
int cContextMenuParamsGetMediaType(struct _cef_context_menu_params_t* params) { return (int)params->get_media_type(params); }
int cContextMenuParamsGetEditStateFlags(struct _cef_context_menu_params_t* params) { return (int)params->get_edit_state_flags(params); }
int cContextMenuParamsIsEditable(struct _cef_context_menu_params_t* params) { return params->is_editable(params); }
cef_string_userfree_t cContextMenuParamsGetLinkUrl(struct _cef_context_menu_params_t* params) { return params->get_link_url(params); }
cef_string_userfree_t cContextMenuParamsGetUnfilteredLinkUrl(struct _cef_context_menu_params_t* params) { return params->get_unfiltered_link_url(params); }
cef_string_userfree_t cContextMenuParamsGetSourceUrl(struct _cef_context_menu_params_t* params) { return params->get_source_url(params); }
cef_string_userfree_t cContextMenuParamsGetPageUrl(struct _cef_context_menu_params_t* params) { return params->get_page_url(params); }
cef_string_userfree_t cContextMenuParamsGetFrameUrl(struct _cef_context_menu_params_t* params) { return params->get_frame_url(params); }
cef_string_userfree_t cContextMenuParamsGetSelectionText(struct _cef_context_menu_params_t* params) { return params->get_selection_text(params); }

size_t cMenuModelGetCount(struct _cef_menu_model_t* model) { return model->get_count(model); }
int cMenuModelGetCommandIdAt(struct _cef_menu_model_t* model, size_t index) { return model->get_command_id_at(model, index); }
cef_string_userfree_t cMenuModelGetLabelAt(struct _cef_menu_model_t* model, size_t index) { return model->get_label_at(model, index); }
int cMenuModelGetTypeAt(struct _cef_menu_model_t* model, size_t index) { return (int)model->get_type_at(model, index); }
int cMenuModelIsEnabledAt(struct _cef_menu_model_t* model, size_t index) { return model->is_enabled_at(model, index); }
int cMenuModelIsCheckedAt(struct _cef_menu_model_t* model, size_t index) { return model->is_checked_at(model, index); }
struct _cef_menu_model_t* cMenuModelGetSubMenuAt(struct _cef_menu_model_t* model, size_t index) { return model->get_sub_menu_at(model, index); }

void cRunContextMenuCallbackContinue(struct _cef_run_context_menu_callback_t* callback, int command_id) {
    callback->cont(callback, command_id, (cef_event_flags_t)0);
}

void cRunContextMenuCallbackCancel(struct _cef_run_context_menu_callback_t* callback) {
    callback->cancel(callback);
}
