#ifndef JSDIALOG_HANDLER_BRIDGE_H
#define JSDIALOG_HANDLER_BRIDGE_H

#include "debug.h"
#include "memory.h"
#include <stdlib.h>
#include "include/capi/cef_client_capi.h"
#include "include/capi/cef_jsdialog_handler_capi.h"

struct _cef_jsdialog_handler_t* cJsDialogHandlerCreate();
void cJsDialogHandlerFree(struct _cef_jsdialog_handler_t*);

void cJsDialogCallbackContinue(struct _cef_jsdialog_callback_t* callback,
                               int success,
                               cef_string_t* user_input);

#endif
