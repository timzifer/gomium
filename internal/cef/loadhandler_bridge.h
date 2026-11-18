#ifndef LOAD_HANDLER_BRIDGE_H
#define LOAD_HANDLER_BRIDGE_H

#include "debug.h"
#include "memory.h"
#include <stdlib.h>
#include "include/capi/cef_client_capi.h"
#include "include/capi/cef_load_handler_capi.h"

struct _cef_load_handler_t* cLoadHandlerCreate();
void cLoadHandlerFree(struct _cef_load_handler_t*);

#endif
