#include "requesthandler_bridge.h"
#include "_cgo_export.h"

static int CEF_CALLBACK on_before_browse_trampoline(struct _cef_request_handler_t* self,
                                                    struct _cef_browser_t* browser,
                                                    struct _cef_frame_t* frame,
                                                    struct _cef_request_t* request,
                                                    int user_gesture,
                                                    int is_redirect) {
    return goOnBeforeBrowse(self, browser, frame, request, is_redirect);
}

static struct _cef_resource_request_handler_t* CEF_CALLBACK get_resource_request_handler_trampoline(
             struct _cef_request_handler_t* self,
             struct _cef_browser_t* browser,
             struct _cef_frame_t* frame,
             struct _cef_request_t* request,
             int is_navigation,
             int is_download,
             const cef_string_t* request_initiator,
             int* disable_default_handling) {
    return goGetResourceRequestHandler(self, browser, frame, request);
}

static int CEF_CALLBACK on_certificate_error_trampoline(struct _cef_request_handler_t* self,
                                                        struct _cef_browser_t* browser,
                                                        cef_errorcode_t cert_error,
                                                        const cef_string_t* request_url,
                                                        struct _cef_sslinfo_t* ssl_info,
                                                        struct _cef_callback_t* callback) {
    return goOnCertificateError(self, browser, cert_error, (cef_string_t*)request_url, ssl_info, callback);
}

static int CEF_CALLBACK on_get_auth_credentials_trampoline(struct _cef_request_handler_t* self,
                                                           struct _cef_browser_t* browser,
                                                           const cef_string_t* origin_url,
                                                           int isProxy,
                                                           const cef_string_t* host,
                                                           int port,
                                                           const cef_string_t* realm,
                                                           const cef_string_t* scheme,
                                                           struct _cef_auth_callback_t* callback) {
    return goOnAuthCredentials(self, browser, (cef_string_t*)origin_url, isProxy,
                               (cef_string_t*)host, port, (cef_string_t*)realm,
                               (cef_string_t*)scheme, callback);
}

void cBridgeAuthCallbackContinue(struct _cef_auth_callback_t* callback, const cef_string_t* username, const cef_string_t* password) {
    callback->cont(callback, username, password);
}

void cBridgeAuthCallbackCancel(struct _cef_auth_callback_t* callback) {
    callback->cancel(callback);
}

struct _cef_request_handler_t* cRequestHandlerCreate() { DEBUG_CALL;
    cef_request_handler_t* handler = (cef_request_handler_t*)calloc(1, sizeof(cef_request_handler_t));
    handler -> base.size = sizeof(cef_request_handler_t);
    initialize_cef_base_ref_counted((cef_base_ref_counted_t*)handler);

    handler -> on_before_browse = on_before_browse_trampoline;
    handler -> get_resource_request_handler = get_resource_request_handler_trampoline;
    handler -> on_certificate_error = on_certificate_error_trampoline;
    handler -> get_auth_credentials = on_get_auth_credentials_trampoline;

    return handler;
}
void cRequestHandlerFree(struct _cef_request_handler_t* handler) { DEBUG_CALL;
    ((cef_base_ref_counted_t*)handler)->release((cef_base_ref_counted_t*)handler);
}

static struct _cef_resource_handler_t* CEF_CALLBACK get_resource_handler_trampoline(
             struct _cef_resource_request_handler_t* self,
             struct _cef_browser_t* browser,
             struct _cef_frame_t* frame,
             struct _cef_request_t* request) {
    return goResourceRequestGetResourceHandler(self, browser, frame, request);
}

static cef_return_value_t CEF_CALLBACK on_before_resource_load_trampoline(
             struct _cef_resource_request_handler_t* self,
             struct _cef_browser_t* browser,
             struct _cef_frame_t* frame,
             struct _cef_request_t* request,
             struct _cef_callback_t* callback) {
    return goOnBeforeResourceLoad(self, browser, frame, request, callback);
}

static void CEF_CALLBACK on_protocol_execution_trampoline(struct _cef_resource_request_handler_t* self,
                                                          struct _cef_browser_t* browser,
                                                          struct _cef_frame_t* frame,
                                                          struct _cef_request_t* request,
                                                          int* allow_os_execution) {
    goOnProtocolExecution(self, browser, frame, request, allow_os_execution);
}

struct _cef_resource_request_handler_t* cResourceRequestHandlerCreate() { DEBUG_CALL;
    cef_resource_request_handler_t* handler = (cef_resource_request_handler_t*)calloc(1, sizeof(cef_resource_request_handler_t));
    handler -> base.size = sizeof(cef_resource_request_handler_t);
    initialize_cef_base_ref_counted((cef_base_ref_counted_t*)handler);

    handler -> on_before_resource_load = on_before_resource_load_trampoline;
    handler -> on_protocol_execution = on_protocol_execution_trampoline;
    handler -> get_resource_handler = get_resource_handler_trampoline;

    return handler;
}
void cResourceRequestHandlerFree(struct _cef_resource_request_handler_t* handler) { DEBUG_CALL;
    ((cef_base_ref_counted_t*)handler)->release((cef_base_ref_counted_t*)handler);
}

void cBridgeContinueResourceLoad(struct _cef_callback_t* callback) { DEBUG_CALL;
    callback -> cont(callback);
}
void cBridgeCancelResourceLoad(struct _cef_callback_t* callback) { DEBUG_CALL;
    callback -> cancel(callback);
}
