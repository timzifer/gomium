#ifndef REQUEST_CONTEXT_BRIDGE_H
#define REQUEST_CONTEXT_BRIDGE_H

#include "debug.h"
#include "memory.h"
#include <stdlib.h>
#include "include/capi/cef_request_context_capi.h"

struct _cef_cookie_manager_t* cRequestContextGetCookieManager(struct _cef_request_context_t* context);
int cRequestContextIsGlobal(struct _cef_request_context_t* context);
int cRequestContextIsSame(struct _cef_request_context_t* context, struct _cef_request_context_t* other);
cef_string_userfree_t cRequestContextGetCachePath(struct _cef_request_context_t* context);

#endif
