#ifndef BROWSER_PROCESS_HANDLER_BRIDGE_H
#define BROWSER_PROCESS_HANDLER_BRIDGE_H

#include "debug.h"
#include "memory.h"
#include <stdlib.h>
#include "include/capi/cef_browser_process_handler_capi.h"

struct _cef_browser_process_handler_t* cBrowserProcessHandlerCreate();
void cBrowserProcessHandlerFree(struct _cef_browser_process_handler_t* browser_process_handler);

#endif