package cef

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"testing"
	"time"
)

// newTestDevToolsSession runs the protocol logic without a browser. Every
// message the session sends shows up on the returned channel.
func newTestDevToolsSession(t *testing.T) (*devToolsSession, chan []byte) {
	t.Helper()
	s := newDevToolsSession(nil)
	sent := make(chan []byte, 16)
	s.send = func(msg []byte) error {
		sent <- msg
		return nil
	}
	t.Cleanup(s.close)
	return s, sent
}

// sentID reads the next outgoing request and returns its id.
func sentID(t *testing.T, sent chan []byte) int {
	t.Helper()
	select {
	case raw := <-sent:
		var req struct {
			ID     int    `json:"id"`
			Method string `json:"method"`
		}
		if err := json.Unmarshal(raw, &req); err != nil {
			t.Fatalf("sent message is not JSON: %v", err)
		}
		return req.ID
	case <-time.After(2 * time.Second):
		t.Fatal("no message sent")
	}
	return 0
}

func TestDevToolsCallResolvesResult(t *testing.T) {
	s, sent := newTestDevToolsSession(t)

	type out struct {
		raw json.RawMessage
		err error
	}
	done := make(chan out, 1)
	go func() {
		raw, err := s.Call(context.Background(), "Runtime.evaluate", map[string]any{"expression": "1+1"})
		done <- out{raw, err}
	}()

	id := sentID(t, sent)
	s.handleMessage([]byte(`{"id":` + strconv.Itoa(id) + `,"result":{"result":{"value":2}}}`))

	got := <-done
	if got.err != nil {
		t.Fatalf("unexpected error: %v", got.err)
	}
	if string(got.raw) != `{"result":{"value":2}}` {
		t.Fatalf("result = %s", got.raw)
	}
}

func TestDevToolsCallReturnsProtocolError(t *testing.T) {
	s, sent := newTestDevToolsSession(t)

	errCh := make(chan error, 1)
	go func() {
		_, err := s.Call(context.Background(), "Nope.nope", nil)
		errCh <- err
	}()

	id := sentID(t, sent)
	s.handleMessage([]byte(`{"id":` + strconv.Itoa(id) + `,"error":{"code":-32601,"message":"not found"}}`))

	var devErr *DevToolsError
	if err := <-errCh; !errors.As(err, &devErr) {
		t.Fatalf("error = %v, want *DevToolsError", err)
	}
	if devErr.Code != -32601 || devErr.Message != "not found" {
		t.Fatalf("devErr = %+v", devErr)
	}
}

func TestDevToolsCallContextTimeoutRemovesPending(t *testing.T) {
	s, sent := newTestDevToolsSession(t)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err := s.Call(ctx, "Slow.method", nil)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want deadline exceeded", err)
	}

	// a late result for the abandoned id must be ignored, not block
	id := sentID(t, sent)
	s.handleMessage([]byte(`{"id":` + strconv.Itoa(id) + `,"result":{}}`))

	s.mu.Lock()
	left := len(s.pending)
	s.mu.Unlock()
	if left != 0 {
		t.Fatalf("pending = %d, want 0", left)
	}
}

func TestDevToolsDetachFailsPendingCalls(t *testing.T) {
	s, sent := newTestDevToolsSession(t)

	errCh := make(chan error, 1)
	go func() {
		_, err := s.Call(context.Background(), "Page.navigate", nil)
		errCh <- err
	}()
	sentID(t, sent)

	// the reattach after detach runs in the background and only sends
	s.onDetached()

	if err := <-errCh; !errors.Is(err, ErrDevToolsDetached) {
		t.Fatalf("error = %v, want ErrDevToolsDetached", err)
	}
}

func TestDevToolsCloseFailsPendingAndRejectsNewCalls(t *testing.T) {
	s, sent := newTestDevToolsSession(t)

	errCh := make(chan error, 1)
	go func() {
		_, err := s.Call(context.Background(), "Page.navigate", nil)
		errCh <- err
	}()
	sentID(t, sent)

	s.close()
	if err := <-errCh; !errors.Is(err, ErrDevToolsClosed) {
		t.Fatalf("pending error = %v, want ErrDevToolsClosed", err)
	}

	if _, err := s.Call(context.Background(), "Page.navigate", nil); !errors.Is(err, ErrDevToolsClosed) {
		t.Fatalf("call after close = %v, want ErrDevToolsClosed", err)
	}
}

func TestDevToolsSendFailureReturnsError(t *testing.T) {
	s, _ := newTestDevToolsSession(t)
	s.send = func([]byte) error { return errors.New("refused") }

	if _, err := s.Call(context.Background(), "Page.enable", nil); err == nil || err.Error() != "refused" {
		t.Fatalf("error = %v, want refused", err)
	}
	s.mu.Lock()
	left := len(s.pending)
	s.mu.Unlock()
	if left != 0 {
		t.Fatalf("pending = %d after failed send, want 0", left)
	}
}

func TestDevToolsEventsInOrderAndCancel(t *testing.T) {
	s, _ := newTestDevToolsSession(t)

	got := make(chan string, 16)
	s.On("Test.event", func(p json.RawMessage) { got <- "a:" + string(p) })
	cancelB := s.On("Test.event", func(p json.RawMessage) { got <- "b:" + string(p) })

	next := func() string {
		t.Helper()
		select {
		case entry := <-got:
			return entry
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for event")
		}
		return ""
	}
	seen := map[string][]string{}
	record := func() {
		entry := next()
		seen[entry[:1]] = append(seen[entry[:1]], entry)
	}

	s.handleMessage([]byte(`{"method":"Test.event","params":{"n":1}}`))
	s.handleMessage([]byte(`{"method":"Test.event","params":{"n":2}}`))
	// delivery is asynchronous: wait for both handlers before unsubscribing b,
	// otherwise b's events would be cancelled while still queued
	for i := 0; i < 4; i++ {
		record()
	}
	cancelB()

	s.handleMessage([]byte(`{"method":"Test.event","params":{"n":3}}`))
	record()

	want := map[string][]string{
		"a": {`a:{"n":1}`, `a:{"n":2}`, `a:{"n":3}`},
		"b": {`b:{"n":1}`, `b:{"n":2}`},
	}
	for tag, entries := range want {
		if len(seen[tag]) != len(entries) {
			t.Fatalf("handler %s got %v, want %v", tag, seen[tag], entries)
		}
		for i := range entries {
			if seen[tag][i] != entries[i] {
				t.Fatalf("handler %s got %v, want %v", tag, seen[tag], entries)
			}
		}
	}
}

func TestDevToolsDetachClearsHandlers(t *testing.T) {
	s, _ := newTestDevToolsSession(t)

	calls := make(chan string, 8)
	s.On("Test.evt", func(json.RawMessage) { calls <- "old" })
	s.onDetached()

	// the receiver of the new attachment registers again: exactly one call
	s.On("Test.evt", func(json.RawMessage) { calls <- "new" })
	s.handleMessage([]byte(`{"method":"Test.evt","params":{}}`))

	select {
	case got := <-calls:
		if got != "new" {
			t.Fatalf("handler %q ran, want only the new one", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("new handler not called")
	}
	select {
	case got := <-calls:
		t.Fatalf("extra handler call %q after reattach", got)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestDevToolsOnCancelStopsDelivery(t *testing.T) {
	s, _ := newTestDevToolsSession(t)

	calls := make(chan struct{}, 4)
	cancel := s.On("Test.gone", func(json.RawMessage) { calls <- struct{}{} })
	cancel()

	s.handleMessage([]byte(`{"method":"Test.gone","params":{}}`))

	select {
	case <-calls:
		t.Fatal("cancelled handler was called")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestDevToolsQueuesOnlySubscribedEvents(t *testing.T) {
	s, _ := newTestDevToolsSession(t)

	s.handleMessage([]byte(`{"method":"Network.dataReceived","params":{}}`))
	s.qmu.Lock()
	n := len(s.queue)
	s.qmu.Unlock()
	if n != 0 {
		t.Fatalf("queued %d events nobody subscribed to", n)
	}
}

func TestDevToolsQueueIsBounded(t *testing.T) {
	s, _ := newTestDevToolsSession(t)

	release := make(chan struct{})
	delivered := make(chan struct{}, devToolsMaxQueued+64)
	s.On("Test.flood", func(json.RawMessage) {
		<-release
		delivered <- struct{}{}
	})
	const extra = 50
	for i := 0; i < devToolsMaxQueued+extra; i++ {
		s.handleMessage([]byte(`{"method":"Test.flood","params":{}}`))
	}

	s.qmu.Lock()
	queued, dropped := len(s.queue), s.dropped
	s.qmu.Unlock()
	if queued > devToolsMaxQueued {
		t.Fatalf("queue grew to %d, limit %d", queued, devToolsMaxQueued)
	}
	// the pump may already hold one event in the blocked handler
	if dropped < extra-1 {
		t.Fatalf("dropped %d events, want at least %d", dropped, extra-1)
	}

	close(release)
	for i := 0; i < queued; i++ {
		select {
		case <-delivered:
		case <-time.After(5 * time.Second):
			t.Fatalf("only %d of %d queued events delivered", i, queued)
		}
	}
	deadline := time.Now().Add(2 * time.Second)
	for {
		s.qmu.Lock()
		empty := s.queue == nil
		s.qmu.Unlock()
		if empty {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("drained queue still holds its backing array")
		}
		time.Sleep(10 * time.Millisecond)
	}
}
