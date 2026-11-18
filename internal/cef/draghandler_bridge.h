#ifndef DRAG_HANDLER_BRIDGE_H
#define DRAG_HANDLER_BRIDGE_H

#include "debug.h"
#include "memory.h"
#include <stdlib.h>
#include "include/capi/cef_client_capi.h"
#include "include/capi/cef_drag_handler_capi.h"
#include "include/internal/cef_string_list.h"

struct _cef_drag_handler_t* cDragHandlerCreate();
void cDragHandlerFree(struct _cef_drag_handler_t*);

// cef_drag_data_t is only ever handed to us inside on_drag_enter, so the two
// accessors we need live here instead of in a wrapper of their own.
int cBridgeDragDataIsFile(struct _cef_drag_data_t*);
int cBridgeDragDataFileNames(struct _cef_drag_data_t*, cef_string_list_t);

#endif
