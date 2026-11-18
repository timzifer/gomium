#include "v8.h"

int js_to_bool(cef_v8_value_t* arg) {
	return arg->get_bool_value(arg);
}

int js_to_int(cef_v8_value_t* arg) {
	return arg->get_int_value(arg);
}

uint32_t js_to_uint(cef_v8_value_t* arg) {
	return arg->get_uint_value(arg);
}

double js_to_double(cef_v8_value_t* arg) {
	return arg->get_double_value(arg);
}

int js_value_is_same(cef_v8_value_t* a, cef_v8_value_t* b) {
    // CEF releases the reference an argument carries (CppToC::Unwrap),
	// so the caller-owned |b| needs one of its own for the call.
	if (b) ((cef_base_ref_counted_t*)b)->add_ref((cef_base_ref_counted_t*)b);
	return a -> is_same(a, b);
}

cef_time_t js_to_date(cef_v8_value_t* arg) {
    cef_basetime_t baseTime = arg -> get_date_value(arg);
    cef_time_t result = {0};
    cef_time_from_basetime(baseTime, &result);
    return result;
}

int js_get_object_keys(cef_v8_value_t* arg, cef_string_list_t keys) {
    return arg -> get_keys(arg, keys);
}
cef_v8_value_t* js_get_object_value_by_key(cef_v8_value_t* parent, cef_string_t* key) {
    return parent -> get_value_bykey(parent, key);
}
int js_set_value_by_index(cef_v8_value_t* self, int index, cef_v8_value_t* value) {
    return self -> set_value_byindex(self, index, value);
}

int js_resolve_promise(cef_v8_value_t* promise, cef_v8_value_t* arg) {
    return promise -> resolve_promise(promise, arg);
}

int js_reject_promise(cef_v8_value_t* promise, const cef_string_t* errorMsg) {
    return promise -> reject_promise(promise, errorMsg);
}

struct _cef_v8_value_t* cef_v8_value_get_value_bykey(struct _cef_v8_value_t* self, const cef_string_t* key) {
	return self -> get_value_bykey(self, key);
}

struct _cef_task_runner_t* v8context_get_task_runner(struct _cef_v8_context_t* self) {
	return self -> get_task_runner(self);
}

int v8context_is_valid(struct _cef_v8_context_t* self) {
	return self -> is_valid(self);
}

struct _cef_browser_t* v8context_get_browser(struct _cef_v8_context_t* self) {
	return self -> get_browser(self);
}

struct _cef_frame_t* v8context_get_frame(struct _cef_v8_context_t* self) {
	return self -> get_frame(self);
}

struct _cef_v8_value_t* v8context_get_global(struct _cef_v8_context_t* self) {
	return self -> get_global(self);
}

int v8context_enter(struct _cef_v8_context_t* self) {
	return self -> enter(self);
}

int v8context_exit(struct _cef_v8_context_t* self) {
	return self -> exit(self);
}

int v8context_is_same(struct _cef_v8_context_t* self, struct _cef_v8_context_t* that) {
	// CEF releases the reference an argument carries (CppToC::Unwrap),
	// so the caller-owned |that| needs one of its own for the call.
	if (that) ((cef_base_ref_counted_t*)that)->add_ref((cef_base_ref_counted_t*)that);
	return self -> is_same(self, that);
}

int v8context_eval(struct _cef_v8_context_t* self, const cef_string_t* code, const cef_string_t* script_url, int start_line, struct _cef_v8_value_t** retval, struct _cef_v8_exception_t** exception) {
	return self -> eval(self, code, script_url, start_line, retval, exception);
}
