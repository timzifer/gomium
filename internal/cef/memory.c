#include "memory.h"
#include "_cgo_export.h"

void base_add_ref(cef_base_ref_counted_t* self) {
     if (self && self->add_ref) {
         self->add_ref(self);
     }
}

void base_release_ref(cef_base_ref_counted_t* self) {
     if (self && self->release) {
         self->release(self);
     }
}

int base_has_one_ref(cef_base_ref_counted_t* self) {
     if (self && self->has_one_ref) {
         return self->has_one_ref(self);
     }
     return -1;
}

///
// Increment the reference count.
///
void CEF_CALLBACK add_ref(cef_base_ref_counted_t* self) {
    goMemoryAddRef((void *) self);
}

///
// Decrement the reference count.  Delete this object when no references
// remain.
///
int CEF_CALLBACK release(cef_base_ref_counted_t* self) {
    return goMemoryRelease((void *) self);
}

///
// Returns true (1) if the current reference count is 1.
///
int CEF_CALLBACK has_one_ref(cef_base_ref_counted_t* self) {
    return goMemoryHasOneRef((void *) self);
}

///
// Returns true (1) if the current reference count is at least 1.
///
int CEF_CALLBACK has_at_least_one_ref(cef_base_ref_counted_t* self) {
    return goMemoryHasAtLeastOneRef((void *) self);
}

void initialize_cef_base_ref_counted(cef_base_ref_counted_t* base) {
    size_t size = base->size;
    if (size == 0) {
        DEBUG_FATAL("FATAL: initialize_cef_base failed, size member not set");
        return;
    }
    base->add_ref = add_ref;
    base->release = release;
    base->has_one_ref = has_one_ref;
    base->has_at_least_one_ref = has_at_least_one_ref;
    goMemoryRegister((void *) base);
}
