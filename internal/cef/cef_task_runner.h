#ifndef CEF_TASK_RUNNER_H
#define CEF_TASK_RUNNER_H

#include <stdlib.h>
#include "include/capi/cef_task_capi.h"
#include "memory.h"

int task_runner_is_same(struct _cef_task_runner_t* self, struct _cef_task_runner_t* that REF_BORROWED);
int task_runner_belongs_to_current_thread(struct _cef_task_runner_t* self);
int task_runner_belongs_to_thread(struct _cef_task_runner_t* self, cef_thread_id_t threadId);
int task_runner_post_task(struct _cef_task_runner_t* self, struct _cef_task_t* task REF_CONSUMED);
int task_runner_post_delayed_task(struct _cef_task_runner_t* self, struct _cef_task_t* task REF_CONSUMED, int64_t delay_ms);
struct _cef_task_t* task_create();

#endif