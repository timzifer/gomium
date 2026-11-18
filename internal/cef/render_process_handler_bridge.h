#ifndef RENDER_PROCESS_HANDLER_BRIDGE_H
#define RENDER_PROCESS_HANDLER_BRIDGE_H

#include "debug.h"
#include "memory.h"
#include <stdlib.h>
#include "include/capi/cef_render_process_handler_capi.h"
#include "include/capi/cef_client_capi.h"

struct _cef_render_process_handler_t* cRenderProcessHandlerCreate();
void cRenderProcessHandlerFree(struct _cef_render_process_handler_t* render_process_handler);

#endif