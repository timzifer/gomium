#ifndef MEMORY_H
#define MEMORY_H

#include "include/capi/cef_base_capi.h"
#include "include/internal/cef_types.h"
#include "debug.h"

// What a bridge function does with the reference a reference-counted
// argument carries, for tools/refcheck. REF_CONSUMED: the function passes it
// on to a CEF method, which takes it over - the caller gives away a reference
// it owns. REF_BORROWED: the function adds the reference CEF will take, the
// caller keeps its own. The first parameter of a bridge function is the
// object whose method runs and is borrowed without saying so.
#define REF_CONSUMED
#define REF_BORROWED

void base_add_ref(cef_base_ref_counted_t* self);
void base_release_ref(cef_base_ref_counted_t* self);
int base_has_one_ref(cef_base_ref_counted_t* self);
void initialize_cef_base_ref_counted(cef_base_ref_counted_t* base);

#endif