#include "cef_task_runner.h"
#include "_cgo_export.h"

static void CEF_CALLBACK task_execute(struct _cef_task_t* self) {
	goTaskExecute(self);
}

int task_runner_is_same(struct _cef_task_runner_t* self, struct _cef_task_runner_t* that) {
	// CEF releases the reference an argument carries (CppToC::Unwrap),
	// so the caller-owned |that| needs one of its own for the call.
	if (that) ((cef_base_ref_counted_t*)that)->add_ref((cef_base_ref_counted_t*)that);
	return self -> is_same(self, that);
}

int task_runner_belongs_to_current_thread(struct _cef_task_runner_t* self) {
	return self -> belongs_to_current_thread(self);
}

int task_runner_belongs_to_thread(struct _cef_task_runner_t* self, cef_thread_id_t threadId) {
	return self -> belongs_to_thread(self, threadId);
}

int task_runner_post_task(struct _cef_task_runner_t* self, struct _cef_task_t* task) {
	return self -> post_task(self, task);
}

int task_runner_post_delayed_task(struct _cef_task_runner_t* self, struct _cef_task_t* task, int64_t delay_ms) {
	return self -> post_delayed_task(self, task, delay_ms);
}

struct _cef_task_t* task_create() {
	cef_task_t* task = (cef_task_t*)calloc(1, sizeof(cef_task_t));
	task -> base.size = sizeof(cef_task_t);
	initialize_cef_base_ref_counted((cef_base_ref_counted_t*) task);

	task -> execute = task_execute;

	return task;
}
