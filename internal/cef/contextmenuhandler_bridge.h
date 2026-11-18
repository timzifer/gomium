#ifndef CONTEXT_MENU_HANDLER_BRIDGE_H
#define CONTEXT_MENU_HANDLER_BRIDGE_H

#include "debug.h"
#include "memory.h"
#include <stdlib.h>
#include "include/capi/cef_client_capi.h"
#include "include/capi/cef_context_menu_handler_capi.h"
#include "include/capi/cef_menu_model_capi.h"

struct _cef_context_menu_handler_t* cContextMenuHandlerCreate();
void cContextMenuHandlerFree(struct _cef_context_menu_handler_t*);

// cef_context_menu_params_t accessors
int cContextMenuParamsGetX(struct _cef_context_menu_params_t* params);
int cContextMenuParamsGetY(struct _cef_context_menu_params_t* params);
int cContextMenuParamsGetTypeFlags(struct _cef_context_menu_params_t* params);
int cContextMenuParamsGetMediaType(struct _cef_context_menu_params_t* params);
int cContextMenuParamsGetEditStateFlags(struct _cef_context_menu_params_t* params);
int cContextMenuParamsIsEditable(struct _cef_context_menu_params_t* params);
cef_string_userfree_t cContextMenuParamsGetLinkUrl(struct _cef_context_menu_params_t* params);
cef_string_userfree_t cContextMenuParamsGetUnfilteredLinkUrl(struct _cef_context_menu_params_t* params);
cef_string_userfree_t cContextMenuParamsGetSourceUrl(struct _cef_context_menu_params_t* params);
cef_string_userfree_t cContextMenuParamsGetPageUrl(struct _cef_context_menu_params_t* params);
cef_string_userfree_t cContextMenuParamsGetFrameUrl(struct _cef_context_menu_params_t* params);
cef_string_userfree_t cContextMenuParamsGetSelectionText(struct _cef_context_menu_params_t* params);

// cef_menu_model_t accessors
size_t cMenuModelGetCount(struct _cef_menu_model_t* model);
int cMenuModelGetCommandIdAt(struct _cef_menu_model_t* model, size_t index);
cef_string_userfree_t cMenuModelGetLabelAt(struct _cef_menu_model_t* model, size_t index);
int cMenuModelGetTypeAt(struct _cef_menu_model_t* model, size_t index);
int cMenuModelIsEnabledAt(struct _cef_menu_model_t* model, size_t index);
int cMenuModelIsCheckedAt(struct _cef_menu_model_t* model, size_t index);
struct _cef_menu_model_t* cMenuModelGetSubMenuAt(struct _cef_menu_model_t* model, size_t index);

void cRunContextMenuCallbackContinue(struct _cef_run_context_menu_callback_t* callback, int command_id);
void cRunContextMenuCallbackCancel(struct _cef_run_context_menu_callback_t* callback);

#endif
