#ifndef REQUEST_HANDLER_BRIDGE_H
#define REQUEST_HANDLER_BRIDGE_H

#include "debug.h"
#include "memory.h"
#include <stdlib.h>
#include "include/capi/cef_request_handler_capi.h"
#include "include/capi/cef_resource_request_handler_capi.h"
#include "resourcehandler_bridge.h"

struct _cef_request_handler_t* cRequestHandlerCreate();
void cRequestHandlerFree(struct _cef_request_handler_t* handler);

struct _cef_resource_request_handler_t* cResourceRequestHandlerCreate();
void cResourceRequestHandlerFree(struct _cef_resource_request_handler_t* handler);

void cBridgeContinueResourceLoad(struct _cef_callback_t* callback);
void cBridgeCancelResourceLoad(struct _cef_callback_t* callback);

void cBridgeAuthCallbackContinue(struct _cef_auth_callback_t* callback, const cef_string_t* username, const cef_string_t* password);
void cBridgeAuthCallbackCancel(struct _cef_auth_callback_t* callback);

#endif
