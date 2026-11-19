#ifndef V8_H
#define V8_H

#include <stdio.h>
#include <stdlib.h>
#include "include/capi/cef_v8_capi.h"
#include "debug.h"
#include "memory.h"



int get_js_type(cef_v8_value_t* arg);
int js_to_bool(cef_v8_value_t* arg);
int js_to_int(cef_v8_value_t* arg);
uint32_t js_to_uint(cef_v8_value_t* arg);
double js_to_double(cef_v8_value_t* arg);

int js_get_object_keys(cef_v8_value_t* arg, cef_string_list_t keys);
cef_time_t js_to_date(cef_v8_value_t* arg);
int js_value_is_same(cef_v8_value_t* a, cef_v8_value_t* b REF_BORROWED);
cef_v8_value_t* js_get_object_value_by_key(cef_v8_value_t* parent, cef_string_t* key);

int js_set_value_by_index(cef_v8_value_t* self, int index, cef_v8_value_t* value REF_CONSUMED);
// setValueByKey consumes the reference |value| carries (v8_value.go).
int setValueByKey(cef_v8_value_t* obj, const char* key, cef_v8_value_t* value REF_CONSUMED);

int js_resolve_promise(cef_v8_value_t* promise, cef_v8_value_t* arg REF_CONSUMED);
int js_reject_promise(cef_v8_value_t* promise, const cef_string_t* errorMsg);

struct _cef_v8_value_t* cef_v8_value_get_value_bykey(struct _cef_v8_value_t* self, const cef_string_t* key);
struct _cef_task_runner_t* v8context_get_task_runner(struct _cef_v8_context_t* self);
int v8context_is_valid(struct _cef_v8_context_t* self);
struct _cef_browser_t* v8context_get_browser(struct _cef_v8_context_t* self);
struct _cef_frame_t* v8context_get_frame(struct _cef_v8_context_t* self);
struct _cef_v8_value_t* v8context_get_global(struct _cef_v8_context_t* self);
int v8context_enter(struct _cef_v8_context_t* self);
int v8context_exit(struct _cef_v8_context_t* self);
int v8context_is_same(struct _cef_v8_context_t* self, struct _cef_v8_context_t* that REF_BORROWED);
int v8context_eval(struct _cef_v8_context_t* self, const cef_string_t* code, const cef_string_t* script_url, int start_line, struct _cef_v8_value_t** retval, struct _cef_v8_exception_t** exception);

#endif // V8_H
