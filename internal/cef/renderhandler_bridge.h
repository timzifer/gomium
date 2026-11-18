#ifndef RENDER_HANDLER_BRIDGE_H
#define RENDER_HANDLER_BRIDGE_H

#include "debug.h"
#include "memory.h"
#include <stdlib.h>
#include "include/capi/cef_render_handler_capi.h"

struct _cef_render_handler_t* cRenderHandlerCreate();
void cRenderHandlerFree(struct _cef_render_handler_t*);

void cBridgeSetCefRectDimensions(cef_rect_t * rect, int x, int y, int width, int height);

#endif