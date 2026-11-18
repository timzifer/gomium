#ifndef FILE_DIALOG_HANDLER_BRIDGE_H
#define FILE_DIALOG_HANDLER_BRIDGE_H

#include "debug.h"
#include "memory.h"
#include <stdlib.h>
#include "include/capi/cef_client_capi.h"
#include "include/capi/cef_dialog_handler_capi.h"

struct _cef_dialog_handler_t* cFileDialogHandlerCreate();
void cFileDialogHandlerFree(struct _cef_dialog_handler_t*);

void cBridgeFileDialogContinue(struct _cef_file_dialog_callback_t* cb, cef_string_list_t paths);
void cBridgeFileDialogCancel(struct _cef_file_dialog_callback_t* cb);

#endif
