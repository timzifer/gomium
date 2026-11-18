#ifndef MEMORY_H
#define MEMORY_H

#include "include/capi/cef_base_capi.h"
#include "include/internal/cef_types.h"
#include "debug.h"


void base_add_ref(cef_base_ref_counted_t* self);
void base_release_ref(cef_base_ref_counted_t* self);
int base_has_one_ref(cef_base_ref_counted_t* self);
void initialize_cef_base_ref_counted(cef_base_ref_counted_t* base);

#endif