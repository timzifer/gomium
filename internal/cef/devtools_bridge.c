#include "devtools_bridge.h"
#include "_cgo_export.h"

// Only on_dev_tools_message does real work. Returning 1 marks the message as
// handled, so CEF never calls on_dev_tools_method_result or on_dev_tools_event
// for it; the Go side parses the JSON itself. The two no-op entries stay set
// because CEF calls them unconditionally when the registration is active.

static int CEF_CALLBACK on_dev_tools_message_trampoline(struct _cef_dev_tools_message_observer_t* self,
                                                        struct _cef_browser_t* browser,
                                                        const void* message,
                                                        size_t message_size) {
    return goDevToolsMessage(self, browser, (void*)message, message_size);
}

static void CEF_CALLBACK on_dev_tools_method_result_trampoline(struct _cef_dev_tools_message_observer_t* self,
                                                               struct _cef_browser_t* browser,
                                                               int message_id,
                                                               int success,
                                                               const void* result,
                                                               size_t result_size) {
    // not reached - on_dev_tools_message returns 1
    if (browser) {
        ((cef_base_ref_counted_t*)browser)->release((cef_base_ref_counted_t*)browser);
    }
}

static void CEF_CALLBACK on_dev_tools_event_trampoline(struct _cef_dev_tools_message_observer_t* self,
                                                       struct _cef_browser_t* browser,
                                                       const cef_string_t* method,
                                                       const void* params,
                                                       size_t params_size) {
    // not reached - on_dev_tools_message returns 1
    if (browser) {
        ((cef_base_ref_counted_t*)browser)->release((cef_base_ref_counted_t*)browser);
    }
}

static void CEF_CALLBACK on_dev_tools_agent_attached_trampoline(struct _cef_dev_tools_message_observer_t* self,
                                                                struct _cef_browser_t* browser) {
    goDevToolsAgentAttached(self, browser);
}

static void CEF_CALLBACK on_dev_tools_agent_detached_trampoline(struct _cef_dev_tools_message_observer_t* self,
                                                                struct _cef_browser_t* browser) {
    goDevToolsAgentDetached(self, browser);
}

struct _cef_dev_tools_message_observer_t* cDevToolsObserverCreate(void) { DEBUG_CALL;
    cef_dev_tools_message_observer_t* observer = (cef_dev_tools_message_observer_t*)calloc(1, sizeof(cef_dev_tools_message_observer_t));
    observer -> base.size = sizeof(cef_dev_tools_message_observer_t);
    initialize_cef_base_ref_counted((cef_base_ref_counted_t*)observer);

    observer -> on_dev_tools_message = on_dev_tools_message_trampoline;
    observer -> on_dev_tools_method_result = on_dev_tools_method_result_trampoline;
    observer -> on_dev_tools_event = on_dev_tools_event_trampoline;
    observer -> on_dev_tools_agent_attached = on_dev_tools_agent_attached_trampoline;
    observer -> on_dev_tools_agent_detached = on_dev_tools_agent_detached_trampoline;

    return observer;
}

void cDevToolsObserverFree(struct _cef_dev_tools_message_observer_t* observer) { DEBUG_CALL;
    ((cef_base_ref_counted_t*)observer)->release((cef_base_ref_counted_t*)observer);
}

struct _cef_registration_t* cBridgeBrowserAddDevToolsObserver(struct _cef_browser_t* browser,
                                                              struct _cef_dev_tools_message_observer_t* observer) {
    struct _cef_browser_host_t* browserHost = browser -> get_host(browser);
    struct _cef_registration_t* registration = browserHost -> add_dev_tools_message_observer(browserHost, observer);
    ((cef_base_ref_counted_t*)browserHost)->release((cef_base_ref_counted_t*)browserHost);
    return registration;
}

int cBridgeBrowserSendDevToolsMessage(struct _cef_browser_t* browser, const void* message, size_t message_size) {
    struct _cef_browser_host_t* browserHost = browser -> get_host(browser);
    int accepted = browserHost -> send_dev_tools_message(browserHost, message, message_size);
    ((cef_base_ref_counted_t*)browserHost)->release((cef_base_ref_counted_t*)browserHost);
    return accepted;
}
