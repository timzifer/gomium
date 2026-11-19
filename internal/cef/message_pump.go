package cef

import (
	"context"
	"time"
)

// defaultPumpWorkerInterval is the heartbeat of the external message pump:
// the longest the pump ever waits between two cef_do_message_loop_work calls.
//
// It is not a fallback for lost wakeups but part of the contract. CEF's
// MessagePumpExternal learns the time of the next delayed task from every
// DoWork and then throws it away - only some of those times are announced
// through OnScheduleMessagePumpWork. cefclient therefore caps every wait at
// 1000/30 ms (kMaxTimerDelay) and so does gomium; a longer heartbeat makes
// timers, animations and offscreen frames run late by up to the interval. On
// Windows and macOS the heartbeat also bounds the latency of native input,
// since nothing announces a new Win32 or AppKit event to the pump.
const defaultPumpWorkerInterval = time.Second / 30

// scheduleWork records a OnScheduleMessagePumpWork request. Called from any
// CEF thread.
//
// delayMs <= 0 asks for work as soon as possible, a positive delay for work
// after that many milliseconds. Only the earliest outstanding due time is
// kept: doing work earlier than requested is harmless (CEF finds nothing due
// and asks again), doing it later is not.
func (c *cefApp) scheduleWork(delayMs int64) {
	due := time.Now()
	if delayMs > 0 {
		due = due.Add(time.Duration(delayMs) * time.Millisecond)
	}

	c.pumpMtx.Lock()
	earlier := c.pumpDue.IsZero() || due.Before(c.pumpDue)
	if earlier {
		c.pumpDue = due
	}
	c.pumpMtx.Unlock()

	if earlier {
		// wake the scheduler without blocking the CEF thread; a pending
		// wakeup already covers this request
		select {
		case c.pumpWakeup <- struct{}{}:
		default:
		}
	}
}

// runPumpScheduler decides when the host gets a cef_do_message_loop_work
// handed through the pump worker: at every due time CEF asked for, and at
// least once per heartbeat. It never calls into CEF itself - the work has to
// run on the thread that called cef_initialize, which only the host owns.
func (c *cefApp) runPumpScheduler(ctx context.Context) {
	defer c.workerWG.Done()

	interval := c.options.pumpWorkerInterval
	if interval <= 0 {
		interval = defaultPumpWorkerInterval
	}

	// a request that arrived during cef_initialize is already in pumpWakeup;
	// the first pass serves it, and the heartbeat starts from there
	timer := time.NewTimer(interval)
	defer timer.Stop()
	timerAt := time.Now().Add(interval)

	for {
		select {
		case <-c.shutdownChannel:
			c.options.logger.Debug("stopping worker loop (shutdown)")
			return
		case <-ctx.Done():
			c.options.logger.Debug("ctx done stopping worker loop")
			return
		case <-c.pumpWakeup:
		case <-timer.C:
			// the timer is drained; Reset below needs no Stop
			timerAt = time.Time{}
		}

		now := time.Now()
		c.pumpMtx.Lock()
		due := c.pumpDue
		dueReached := !due.IsZero() && !due.After(now)
		if dueReached {
			c.pumpDue = time.Time{}
		}
		c.pumpMtx.Unlock()

		heartbeat := timerAt.IsZero() || !timerAt.After(now)
		if dueReached || heartbeat {
			c.options.pumpWorker(0, c.doMessageWork)
		}

		// next wakeup: the heartbeat, or the outstanding due time when that
		// comes first. A due time that is still pending (the heartbeat fired
		// before it) stays recorded - CEF does not necessarily announce it
		// a second time.
		next := now.Add(interval)
		if !dueReached && !due.IsZero() && due.Before(next) {
			next = due
		}
		if dueReached || heartbeat || next.Before(timerAt) {
			if !timerAt.IsZero() && !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(next.Sub(now))
			timerAt = next
		}
	}
}

// uiWaitInterval is how often a caller blocked on the UI thread advances the
// message loop itself, see beginUIWait. Same cadence as Run uses while it
// waits for on_context_initialized.
const uiWaitInterval = 10 * time.Millisecond

// uiWaitPump returns the function a caller has to run while it blocks on a
// CEF callback, or nil when blocking is safe as it is.
//
// With an external message pump the CEF UI thread is the host's thread that
// called cef_initialize - in a Fyne host, the main loop. A call there that
// waits for a UI-thread callback (on_after_created, on_before_close, a cookie
// or print callback, a DevTools reply) blocks the only thread that could run
// it: the callback arrives after the timeout, never before. Such a caller has
// to pump while it waits.
//
// A wait that starts inside a CEF callback (doMessageWork is already active)
// cannot pump: CEF must not be pumped reentrantly. It still gets the pump
// function - doMessageWork drops the nested call - and times out as before.
func uiWaitPump() func() {
	c := getInstance()
	if c == nil || c.options.pumpWorker == nil || !c.pumpReady.Load() {
		return nil
	}
	if !CefCurrentlyOn(TidUI) {
		return nil
	}
	return c.doMessageWork
}

// uiWait keeps the message loop moving while the caller blocks on a CEF
// callback, see uiWaitPump. The zero value and a nil *uiWait never tick.
type uiWait struct {
	ticker *time.Ticker
	pump   func()
}

// beginUIWait starts a wait. Callers add `case <-w.tick(): w.run()` to the
// select they block in, loop until their own case fires, and call stop.
func beginUIWait() *uiWait {
	pump := uiWaitPump()
	if pump == nil {
		return &uiWait{}
	}
	return &uiWait{ticker: time.NewTicker(uiWaitInterval), pump: pump}
}

func (w *uiWait) tick() <-chan time.Time {
	if w == nil || w.ticker == nil {
		return nil
	}
	return w.ticker.C
}

func (w *uiWait) run() {
	if w != nil && w.pump != nil {
		w.pump()
	}
}

func (w *uiWait) stop() {
	if w != nil && w.ticker != nil {
		w.ticker.Stop()
	}
}
