#ifndef RESOURCE_HANDLER_BRIDGE_H
#define RESOURCE_HANDLER_BRIDGE_H

#include "debug.h"
#include "memory.h"
#include <stdlib.h>
#include "include/capi/cef_resource_handler_capi.h"
#include "include/capi/cef_response_capi.h"

struct _cef_resource_handler_t* cResourceHandlerCreate();

void cBridgeResponseSet(struct _cef_response_t* response, const cef_string_t* mime, const cef_string_t* charset, int status);
void cBridgeCallbackCont(struct _cef_callback_t* cb);
void cBridgeCallbackCancel(struct _cef_callback_t* cb);

#endif
