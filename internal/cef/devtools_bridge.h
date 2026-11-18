#ifndef DEVTOOLS_BRIDGE_H
#define DEVTOOLS_BRIDGE_H

#include "debug.h"
#include "memory.h"
#include <stddef.h>
#include "include/capi/cef_browser_capi.h"
#include "include/capi/cef_devtools_message_observer_capi.h"
#include "include/capi/cef_registration_capi.h"

// One observer per client. Every browser of the client registers this same
// object; the callbacks identify the browser by its identifier.
struct _cef_dev_tools_message_observer_t* cDevToolsObserverCreate(void);
void cDevToolsObserverFree(struct _cef_dev_tools_message_observer_t*);

// cBridgeBrowserAddDevToolsObserver takes over one reference of observer (the
// caller adds it beforehand) and returns the registration the caller owns.
// Must run on the UI thread.
struct _cef_registration_t* cBridgeBrowserAddDevToolsObserver(struct _cef_browser_t* browser,
                                                              struct _cef_dev_tools_message_observer_t* observer);

// cBridgeBrowserSendDevToolsMessage sends one raw JSON method call. Returns
// nonzero when CEF accepted it. Must run on the UI thread.
int cBridgeBrowserSendDevToolsMessage(struct _cef_browser_t* browser, const void* message, size_t message_size);

#endif
