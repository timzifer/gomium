#ifndef KEYBOARD_HANDLER_BRIDGE_H
#define KEYBOARD_HANDLER_BRIDGE_H

#include "debug.h"
#include "memory.h"
#include <stdlib.h>
#include "include/capi/cef_client_capi.h"
#include "include/capi/cef_keyboard_handler_capi.h"

struct _cef_keyboard_handler_t* cKeyboardHandlerCreate();
void cKeyboardHandlerFree(struct _cef_keyboard_handler_t*);

#endif
