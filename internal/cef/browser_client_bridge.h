#ifndef BROWSER_CLIENT_BRIDGE_H
#define BROWSER_CLIENT_BRIDGE_H

#include "debug.h"
#include "memory.h"
#include <stdlib.h>
#include "include/capi/cef_client_capi.h"
#include "include/capi/cef_frame_capi.h"
#include "include/capi/cef_v8_capi.h"
#include "include/capi/cef_task_capi.h"

struct _cef_client_t* cClientCreate();
void cClientFree(struct _cef_client_t* client);

int cBridgeGetBrowserIdentifier(struct _cef_browser_t* browser);
void cBridgeBrowserStopLoad(struct _cef_browser_t *browser);

cef_string_utf8_t * cBridgeCefSourceToString(cef_string_t * source);

#endif