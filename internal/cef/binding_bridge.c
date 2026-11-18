#include <stdlib.h>
#include "binding_bridge.h"
#include "memory.h"
#include "_cgo_export.h"

// gomium_binding_handler_t is the cef_v8_handler_t the page calls through. The
// base must stay first: CEF hands the handler pointer back to the execute
// callback unchanged.
typedef struct {
    cef_v8_handler_t base;
    int64_t id;
} gomium_binding_handler_t;

static int CEF_CALLBACK binding_execute(struct _cef_v8_handler_t* self, const cef_string_t* name, struct _cef_v8_value_t* object, size_t arguments_count, struct _cef_v8_value_t* const* arguments, struct _cef_v8_value_t** retval, cef_string_t* exception) {
    gomium_binding_handler_t* handler = (gomium_binding_handler_t*)self;
    int handled = goBindingExecute(handler->id, (GoInt64)arguments_count, (void*)arguments, (void*)retval);

    // Struct parameters belong to the callee: CEF wrapped |object| and every
    // argument with a reference of their own, and nothing else drops them.
    // The Go side only borrows the arguments (exportArguments takes and
    // returns its own reference), so they are released here, after it is
    // done - otherwise every call leaks this, the arguments and their V8
    // persistent handles.
    if (object) {
        object->base.release(&object->base);
    }
    for (size_t i = 0; i < arguments_count; i++) {
        if (arguments[i]) {
            arguments[i]->base.release(&arguments[i]->base);
        }
    }
    return handled;
}

struct _cef_v8_value_t* cBindingFunctionCreate(const cef_string_t* name, int64_t id) {
    gomium_binding_handler_t* handler = (gomium_binding_handler_t*)calloc(1, sizeof(gomium_binding_handler_t));
    if (handler == NULL) {
        return NULL;
    }
    handler->base.base.size = sizeof(cef_v8_handler_t);
    initialize_cef_base_ref_counted((cef_base_ref_counted_t*)&handler->base);
    handler->base.execute = binding_execute;
    handler->id = id;

    // create_function takes over the handler reference
    return cef_v8_value_create_function(name, &handler->base);
}

int cBindingSetProperty(struct _cef_v8_value_t* object, const cef_string_t* key, struct _cef_v8_value_t* value) {
    return object->set_value_bykey(object, key, value, V8_PROPERTY_ATTRIBUTE_NONE);
}
