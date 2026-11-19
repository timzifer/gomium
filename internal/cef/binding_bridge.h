#ifndef BINDING_BRIDGE_H
#define BINDING_BRIDGE_H

#include <stdint.h>
#include "include/capi/cef_v8_capi.h"
#include "memory.h"

// cBindingFunctionCreate returns a new V8 function named name. Calling it in
// the page runs goBindingExecute with id. The returned reference belongs to
// the caller.
struct _cef_v8_value_t* cBindingFunctionCreate(const cef_string_t* name, int64_t id);

// cBindingSetProperty stores value under key on object. It consumes the
// caller's reference to value (CEF semantics, verified in the spike); keep an
// extra reference beforehand when the value is still needed.
int cBindingSetProperty(struct _cef_v8_value_t* object, const cef_string_t* key, struct _cef_v8_value_t* value REF_CONSUMED);

#endif // BINDING_BRIDGE_H
