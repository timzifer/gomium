#ifndef LIFE_SPAN_HANDLER_BRIDGE_H
#define LIFE_SPAN_HANDLER_BRIDGE_H

#include "debug.h"
#include "memory.h"
#include <stdlib.h>
#include "include/capi/cef_client_capi.h"
#include "include/capi/cef_life_span_handler_capi.h"

struct _cef_life_span_handler_t* cLifeSpanHandlerCreate();
void cLifeSpanHandlerFree(struct _cef_life_span_handler_t*);

#endif