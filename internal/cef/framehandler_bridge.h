#ifndef FRAME_HANDLER_BRIDGE_H
#define FRAME_HANDLER_BRIDGE_H

#include "debug.h"
#include "memory.h"
#include <stdlib.h>
#include "include/capi/cef_client_capi.h"
#include "include/capi/cef_frame_handler_capi.h"

struct _cef_frame_handler_t* cFrameHandlerCreate();
void cFrameHandlerFree(struct _cef_frame_handler_t*);

#endif
