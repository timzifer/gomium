#ifndef REQUEST_BRIDGE_H
#define REQUEST_BRIDGE_H

#include "debug.h"
#include "memory.h"
#include <stdlib.h>

#include "include/capi/cef_request_capi.h"

int cBridgeRequestIsReadOnly(struct _cef_request_t* request);
cef_string_utf8_t* cBridgeRequestGetUrl(struct _cef_request_t* request);
cef_string_utf8_t* cBridgeRequestGetMethod(struct _cef_request_t* request);
cef_string_utf8_t* cBridgeRequestGetReferrer(struct _cef_request_t* request);
void cBridgeRequestSetUrl(struct _cef_request_t* request, const cef_string_t* url);

#endif