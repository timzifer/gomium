package cef

/*
#include "cef_task_runner.h"


*/
import "C"
import (
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

type (
	cefTaskRunner struct {
		cStruct *C.cef_task_runner_t
	}
	CefTaskRunner interface {
		Equals(CefTaskRunner) bool
		BelongsToCurrentThread() bool
		BelongsToThread(CefThreadId) bool

		PostTask(CefTask) bool
		PostDelayedTask(CefTask, time.Duration) bool

		// Release drops the reference held by this wrapper. The wrapper must
		// not be used afterwards.
		Release()

		ptr() *C.cef_task_runner_t
	}

	CefTask func()
)

var (
	taskRegistry     map[*C.cef_task_t]CefTask = make(map[*C.cef_task_t]CefTask)
	taskRegistryLock sync.Mutex
)

// createTask returns a task with the one reference the caller owns; posting
// it hands that reference to CEF.
//
// refcheck:fresh
func createTask(task CefTask) *C.cef_task_t {
	theTask := C.task_create()

	taskRegistryLock.Lock()
	taskRegistry[theTask] = task
	taskRegistryLock.Unlock()

	return theTask
}

// removeTask drops the registry entry for a task that was never executed
// (e.g. because post_task failed).
func removeTask(task *C.cef_task_t) {
	taskRegistryLock.Lock()
	delete(taskRegistry, task)
	taskRegistryLock.Unlock()
}

//export goTaskExecute
func goTaskExecute(task *C.cef_task_t) {
	taskRegistryLock.Lock()
	var (
		theTask func()
		exists  bool
	)
	if theTask, exists = taskRegistry[task]; exists {
		delete(taskRegistry, task)
	}
	taskRegistryLock.Unlock()
	if nil != theTask {
		theTask()
	}
}

// runOnUiThread executes fn on the CEF UI thread and blocks until it ran.
// Returns false when posting failed or the message loop never executed the
// task within 10 seconds - and then fn is guaranteed never to run.
//
// That guarantee is the point of the state below. Callers free what fn uses
// (the browser reference, C strings, a dictionary) as soon as this returns;
// a timed-out task that still ran later would hand all of it to CEF after
// the fact. So the task and the timeout race for one CAS: whoever moves the
// state away from pending first decides. A task that already started is
// waited for, however long it takes, because abandoning it would be the
// use-after-free again.
func runOnUiThread(fn func()) bool {
	if CefCurrentlyOn(TidUI) {
		fn()
		return true
	}

	const (
		taskPending int32 = iota
		taskRunning
		taskAbandoned
	)
	var state atomic.Int32
	done := make(chan struct{})
	theTask := createTask(func() {
		if !state.CompareAndSwap(taskPending, taskRunning) {
			return // the caller gave up and has freed what fn would use
		}
		defer close(done)
		fn()
	})
	if C.cef_post_task(C.TID_UI, theTask) == 0 {
		removeTask(theTask)
		return false
	}

	select {
	case <-done:
		return true
	case <-time.After(10 * time.Second):
		if state.CompareAndSwap(taskPending, taskAbandoned) {
			return false
		}
		<-done
		return true
	}
}

func (c cefTaskRunner) PostTask(task CefTask) bool {
	theTask := createTask(task)
	// post_task consumes the task reference, no matter whether posting
	// succeeded - only the registry entry needs cleanup on failure.
	if C.task_runner_post_task(c.cStruct, theTask) == 0 {
		removeTask(theTask)
		return false
	}
	return true
}

func (c cefTaskRunner) PostDelayedTask(task CefTask, duration time.Duration) bool {
	theTask := createTask(task)
	if C.task_runner_post_delayed_task(c.cStruct, theTask, C.int64_t(duration.Milliseconds())) == 0 {
		removeTask(theTask)
		return false
	}
	return true
}

func (c cefTaskRunner) ptr() *C.cef_task_runner_t {
	return c.cStruct
}

func (c cefTaskRunner) Release() {
	releaseBase(unsafe.Pointer(c.cStruct))
}

func (c cefTaskRunner) Equals(that CefTaskRunner) bool {
	if that == nil {
		return false
	}
	return C.task_runner_is_same(c.cStruct, that.ptr()) != 0
}

func (c cefTaskRunner) BelongsToCurrentThread() bool {
	return C.task_runner_belongs_to_current_thread(c.cStruct) != 0
}

func (c cefTaskRunner) BelongsToThread(id CefThreadId) bool {
	return C.task_runner_belongs_to_thread(c.cStruct, C.cef_thread_id_t(id)) != 0
}

var _ CefTaskRunner = (*cefTaskRunner)(nil)
