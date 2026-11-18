#include "resourcehandler_bridge.h"
#include "_cgo_export.h"

static int CEF_CALLBACK resource_open_trampoline(struct _cef_resource_handler_t* self,
                                                 struct _cef_request_t* request,
                                                 int* handle_request,
                                                 struct _cef_callback_t* callback) {
    return goResourceHandlerOpen(self, request, handle_request, callback);
}

static void CEF_CALLBACK resource_get_response_headers_trampoline(struct _cef_resource_handler_t* self,
                                                                  struct _cef_response_t* response,
                                                                  int64_t* response_length,
                                                                  cef_string_t* redirectUrl) {
    goResourceHandlerGetResponseHeaders(self, response, response_length);
}

static int CEF_CALLBACK resource_skip_trampoline(struct _cef_resource_handler_t* self,
                                                 int64_t bytes_to_skip,
                                                 int64_t* bytes_skipped,
                                                 struct _cef_resource_skip_callback_t* callback) {
    return goResourceHandlerSkip(self, bytes_to_skip, bytes_skipped, callback);
}

static int CEF_CALLBACK resource_read_trampoline(struct _cef_resource_handler_t* self,
                                                 void* data_out,
                                                 int bytes_to_read,
                                                 int* bytes_read,
                                                 struct _cef_resource_read_callback_t* callback) {
    return goResourceHandlerRead(self, data_out, bytes_to_read, bytes_read, callback);
}

static void CEF_CALLBACK resource_cancel_trampoline(struct _cef_resource_handler_t* self) {
    goResourceHandlerCancel(self);
}

struct _cef_resource_handler_t* cResourceHandlerCreate() { DEBUG_CALL;
    cef_resource_handler_t* handler = (cef_resource_handler_t*)calloc(1, sizeof(cef_resource_handler_t));
    handler->base.size = sizeof(cef_resource_handler_t);
    initialize_cef_base_ref_counted((cef_base_ref_counted_t*)handler);

    handler->open                 = resource_open_trampoline;
    handler->get_response_headers = resource_get_response_headers_trampoline;
    handler->skip                 = resource_skip_trampoline;
    handler->read                 = resource_read_trampoline;
    handler->cancel               = resource_cancel_trampoline;

    return handler;
}

void cBridgeResponseSet(struct _cef_response_t* response, const cef_string_t* mime, const cef_string_t* charset, int status) {
    response->set_mime_type(response, mime);
    if (charset) {
        response->set_charset(response, charset);
    }
    response->set_status(response, status);
}

void cBridgeCallbackCont(struct _cef_callback_t* cb) {
    cb->cont(cb);
}

void cBridgeCallbackCancel(struct _cef_callback_t* cb) {
    cb->cancel(cb);
}
