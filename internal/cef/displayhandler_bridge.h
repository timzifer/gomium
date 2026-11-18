#ifndef DISPLAY_HANDLER_BRIDGE_H
#define DISPLAY_HANDLER_BRIDGE_H

#include "debug.h"
#include "memory.h"
#include <stdlib.h>
#include "include/capi/cef_client_capi.h"
#include "include/capi/cef_display_handler_capi.h"

struct _cef_display_handler_t* cDisplayHandlerCreate();
void cDisplayHandlerFree(struct _cef_display_handler_t*);

#endif