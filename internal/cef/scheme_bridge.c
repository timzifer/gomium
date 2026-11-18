#include "scheme_bridge.h"
#include "_cgo_export.h"
#include <string.h>

// cSchemeRegistrarAdd announces one custom scheme. The registrar is a
// cef_base_scoped_t owned by CEF for the duration of the callback - it must
// neither be released nor kept.
int cSchemeRegistrarAdd(struct _cef_scheme_registrar_t* registrar, const char* name, int options) { DEBUG_CALL;
    cef_string_t scheme_name = {0};
    cef_string_utf8_to_utf16(name, strlen(name), &scheme_name);

    int registered = registrar->add_custom_scheme(registrar, &scheme_name, options);

    cef_string_clear(&scheme_name);
    return registered;
}

static struct _cef_resource_handler_t* CEF_CALLBACK scheme_factory_create_trampoline(
    struct _cef_scheme_handler_factory_t* self,
    struct _cef_browser_t* browser,
    struct _cef_frame_t* frame,
    const cef_string_t* scheme_name,
    struct _cef_request_t* request) {
    // scheme_name is a const string owned by CEF; the URL in the request
    // carries the same information and is what the Go side works with.
    return goSchemeHandlerFactoryCreate(self, browser, frame, request);
}

struct _cef_scheme_handler_factory_t* cSchemeHandlerFactoryCreate() { DEBUG_CALL;
    cef_scheme_handler_factory_t* factory =
        (cef_scheme_handler_factory_t*)calloc(1, sizeof(cef_scheme_handler_factory_t));
    factory->base.size = sizeof(cef_scheme_handler_factory_t);
    initialize_cef_base_ref_counted((cef_base_ref_counted_t*)factory);

    factory->create = scheme_factory_create_trampoline;

    return factory;
}

void cSchemeHandlerFactoryFree(struct _cef_scheme_handler_factory_t* factory) { DEBUG_CALL;
    ((cef_base_ref_counted_t*)factory)->release((cef_base_ref_counted_t*)factory);
}

// cRequestContextRegisterSchemeHandlerFactory hands the factory to one request
// context. A NULL domain matches every host of a standard scheme. The call
// consumes a reference on the factory - the caller adds one beforehand and
// keeps its own for cSchemeHandlerFactoryFree.
int cRequestContextRegisterSchemeHandlerFactory(struct _cef_request_context_t* context,
                                                const char* scheme,
                                                struct _cef_scheme_handler_factory_t* factory) { DEBUG_CALL;
    cef_string_t scheme_name = {0};
    cef_string_utf8_to_utf16(scheme, strlen(scheme), &scheme_name);

    int registered = context->register_scheme_handler_factory(context, &scheme_name, NULL, factory);

    cef_string_clear(&scheme_name);
    return registered;
}
