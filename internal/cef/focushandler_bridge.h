#ifndef FOCUS_HANDLER_BRIDGE_H
#define FOCUS_HANDLER_BRIDGE_H

#include "debug.h"
#include "memory.h"
#include <stdlib.h>
#include "include/capi/cef_client_capi.h"
#include "include/capi/cef_focus_handler_capi.h"

struct _cef_focus_handler_t* cFocusHandlerCreate();
void cFocusHandlerFree(struct _cef_focus_handler_t*);

#endif
