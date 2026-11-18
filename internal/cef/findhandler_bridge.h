#ifndef FIND_HANDLER_BRIDGE_H
#define FIND_HANDLER_BRIDGE_H

#include "debug.h"
#include "memory.h"
#include <stdlib.h>
#include "include/capi/cef_client_capi.h"
#include "include/capi/cef_find_handler_capi.h"

struct _cef_find_handler_t* cFindHandlerCreate();
void cFindHandlerFree(struct _cef_find_handler_t*);

#endif
