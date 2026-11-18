package cef

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

// pumpRecorder stands in for a host: it records when the scheduler handed it
// a worker, without running it (there is no CEF in a unit test).
type pumpRecorder struct {
	mtx   sync.Mutex
	times []time.Time
	fired chan struct{}
}

func (r *pumpRecorder) worker(delay int, _ Worker) {
	r.mtx.Lock()
	r.times = append(r.times, time.Now())
	r.mtx.Unlock()
	select {
	case r.fired <- struct{}{}:
	default:
	}
}

func (r *pumpRecorder) count() int {
	r.mtx.Lock()
	defer r.mtx.Unlock()
	return len(r.times)
}

func startScheduler(t *testing.T, interval time.Duration) (*cefApp, *pumpRecorder) {
	t.Helper()
	rec := &pumpRecorder{fired: make(chan struct{}, 64)}
	app := &cefApp{
		shutdownChannel: make(chan struct{}),
		pumpWakeup:      make(chan struct{}, 1),
		options: appOptions{
			logger:             zerolog.Nop(),
			pumpWorker:         rec.worker,
			pumpWorkerInterval: interval,
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	app.workerWG.Add(1)
	go app.runPumpScheduler(ctx)
	t.Cleanup(func() {
		cancel()
		app.workerWG.Wait()
	})
	return app, rec
}

func TestPumpSchedulerImmediateWork(t *testing.T) {
	app, rec := startScheduler(t, time.Hour)

	start := time.Now()
	app.scheduleWork(0)
	select {
	case <-rec.fired:
	case <-time.After(time.Second):
		t.Fatal("immediate work request was not served")
	}
	if d := time.Since(start); d > 200*time.Millisecond {
		t.Fatalf("immediate work took %v", d)
	}
}

// A positive delay must be honoured: serving it at once makes CEF ask again
// with the remaining delay, a wakeup storm until the delay is over.
func TestPumpSchedulerHonoursDelay(t *testing.T) {
	app, rec := startScheduler(t, time.Hour)

	start := time.Now()
	app.scheduleWork(150)
	select {
	case <-rec.fired:
	case <-time.After(2 * time.Second):
		t.Fatal("delayed work request was not served")
	}
	if d := time.Since(start); d < 140*time.Millisecond {
		t.Fatalf("delayed work served after %v, want ~150ms", d)
	}
	if n := rec.count(); n != 1 {
		t.Fatalf("got %d workers for one delayed request", n)
	}
}

// An earlier request replaces a later one, a later one never postpones an
// earlier one.
func TestPumpSchedulerKeepsEarliestDue(t *testing.T) {
	app, rec := startScheduler(t, time.Hour)

	start := time.Now()
	app.scheduleWork(500)
	app.scheduleWork(50)
	app.scheduleWork(800)
	select {
	case <-rec.fired:
	case <-time.After(2 * time.Second):
		t.Fatal("work request was not served")
	}
	if d := time.Since(start); d > 400*time.Millisecond {
		t.Fatalf("earliest due time lost: served after %v", d)
	}
}

// The heartbeat keeps the loop moving without any request - CEF does not
// announce every delayed task.
func TestPumpSchedulerHeartbeat(t *testing.T) {
	_, rec := startScheduler(t, 20*time.Millisecond)

	time.Sleep(250 * time.Millisecond)
	if n := rec.count(); n < 5 {
		t.Fatalf("heartbeat produced %d workers in 250ms at 20ms interval", n)
	}
}

// A worker the host runs before Run or after Close must not touch CEF.
func TestDoMessageWorkNotReady(t *testing.T) {
	app := &cefApp{pumpWakeup: make(chan struct{}, 1)}
	app.doMessageWork() // would crash in cef_do_message_loop_work if it went through
}

// Without an app, or before cef_initialize, a blocking wait must not pump:
// uiWaitPump stays nil before it ever reaches cef_currently_on (which is not
// bound in a unit test), and the wait never ticks.
func TestUIWaitWithoutPump(t *testing.T) {
	if pump := uiWaitPump(); pump != nil {
		t.Fatal("uiWaitPump without an app must be nil")
	}

	uw := beginUIWait()
	defer uw.stop()
	if uw.tick() != nil {
		t.Fatal("a wait without pump must not tick")
	}
	uw.run() // no-op, must not panic

	var none *uiWait
	if none.tick() != nil {
		t.Fatal("nil wait must not tick")
	}
	none.run()
	none.stop()
}
