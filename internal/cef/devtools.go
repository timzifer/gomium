package cef

/*
#include <stdlib.h>
#include "browser_client_bridge.h"
#include "devtools_bridge.h"
*/
import "C"
import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
	"unsafe"
)

// DevTools is the DevTools protocol of one browser, handed to the receiver
// registered with WithDevTools. Calls and events go through CEF in-process;
// no remote debugging port is involved.
//
// The agent detaches on certain navigations and on renderer changes. Event
// subscriptions are dropped by CEF at that point, so enable domains again in
// the receiver, which runs on every attach.
type DevTools interface {
	// Call runs one protocol method and waits for its result. params is
	// marshalled to JSON and may be nil. A failing method returns a
	// *DevToolsError. If the agent detaches while the call is pending, the
	// call fails with ErrDevToolsDetached - CEF does not deliver that result.
	Call(ctx context.Context, method string, params any) (json.RawMessage, error)
	// On registers fn for an event of the protocol, e.g. "Page.loadEventFired".
	// fn runs on a goroutine that delivers events in order. The returned
	// function removes the handler again.
	On(event string, fn func(params json.RawMessage)) (cancel func())
}

var (
	// ErrDevToolsDetached: the agent detached while the call was pending.
	ErrDevToolsDetached = errors.New("devtools agent detached before the result arrived")
	// ErrDevToolsClosed: the browser closed; the session is gone.
	ErrDevToolsClosed = errors.New("devtools session closed with its browser")
)

// DevToolsError is an error object returned by the protocol.
type DevToolsError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *DevToolsError) Error() string {
	return fmt.Sprintf("devtools error %d: %s", e.Code, e.Message)
}

// devToolsIDBase keeps the message ids of sessions away from the ids that
// ExecuteDevToolsMethod gets from CEF (they count up from 1). Both kinds of
// results reach the same observer; unknown ids are ignored.
const devToolsIDBase = 1 << 30

// devToolsBootstrapMethod is sent once an observer is registered. CEF attaches
// the agent only in response to a message sent while it is detached, so this
// call is what makes the attach callback (and with it the receiver) happen.
const devToolsBootstrapMethod = "Page.getFrameTree"

// devToolsReattachDelay spaces the re-bootstrap after a detach so a detaching
// agent cannot cause a busy loop.
const devToolsReattachDelay = 200 * time.Millisecond

type devToolsReply struct {
	result json.RawMessage
	err    error
}

type devToolsEvent struct {
	method string
	params json.RawMessage
}

type devToolsSession struct {
	browser *browser
	// send delivers one raw message to CEF. Set to postToBrowser; tests
	// replace it to run the protocol logic without a browser.
	send func(msg []byte) error

	mu      sync.Mutex
	closed  bool
	nextID  int
	pending map[int]chan devToolsReply
	subs    map[string]map[int]func(json.RawMessage)
	nextSub int

	qmu   sync.Mutex
	queue []devToolsEvent
	wake  chan struct{}
	done  chan struct{}
}

func newDevToolsSession(b *browser) *devToolsSession {
	s := &devToolsSession{
		browser: b,
		nextID:  devToolsIDBase,
		pending: make(map[int]chan devToolsReply),
		subs:    make(map[string]map[int]func(json.RawMessage)),
		wake:    make(chan struct{}, 1),
		done:    make(chan struct{}),
	}
	s.send = s.postToBrowser
	go s.pump()
	return s
}

// pump delivers events in arrival order. It keeps the UI thread free: the
// observer callback only appends to the queue.
func (s *devToolsSession) pump() {
	for {
		select {
		case <-s.done:
			return
		case <-s.wake:
		}
		for {
			s.qmu.Lock()
			if len(s.queue) == 0 {
				s.qmu.Unlock()
				break
			}
			ev := s.queue[0]
			s.queue[0] = devToolsEvent{}
			s.queue = s.queue[1:]
			s.qmu.Unlock()

			s.mu.Lock()
			var handlers []func(json.RawMessage)
			for _, fn := range s.subs[ev.method] {
				handlers = append(handlers, fn)
			}
			s.mu.Unlock()
			for _, fn := range handlers {
				fn(ev.params)
			}
		}
	}
}

func (s *devToolsSession) enqueue(ev devToolsEvent) {
	s.qmu.Lock()
	s.queue = append(s.queue, ev)
	s.qmu.Unlock()
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

type devToolsRequest struct {
	ID     int    `json:"id"`
	Method string `json:"method"`
	Params any    `json:"params,omitempty"`
}

type devToolsIncoming struct {
	ID     *int            `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
	Result json.RawMessage `json:"result"`
	Error  *DevToolsError  `json:"error"`
}

// handleMessage runs on the UI thread. It resolves pending calls directly and
// queues events for the pump.
func (s *devToolsSession) handleMessage(raw []byte) {
	var msg devToolsIncoming
	if err := json.Unmarshal(raw, &msg); err != nil {
		return
	}

	if msg.ID != nil {
		s.mu.Lock()
		ch, ok := s.pending[*msg.ID]
		if ok {
			delete(s.pending, *msg.ID)
		}
		s.mu.Unlock()
		if !ok {
			return
		}
		reply := devToolsReply{result: msg.Result}
		if msg.Error != nil {
			reply = devToolsReply{err: msg.Error}
		}
		ch <- reply // buffered, exactly one sender per id
		return
	}

	if msg.Method != "" {
		s.enqueue(devToolsEvent{method: msg.Method, params: msg.Params})
	}
}

// onDetached fails the pending calls and schedules a reattach.
func (s *devToolsSession) onDetached() {
	s.mu.Lock()
	for id, ch := range s.pending {
		ch <- devToolsReply{err: ErrDevToolsDetached}
		delete(s.pending, id)
	}
	// Event handlers belong to one attachment: CEF drops the protocol
	// subscriptions with the agent, so the handlers go too. The receiver
	// registers them again after the reattach. Keeping them would fire every
	// handler twice after the second attach.
	s.subs = make(map[string]map[int]func(json.RawMessage))
	closed := s.closed
	s.mu.Unlock()

	if !closed {
		// Reattach so the receiver runs again and can re-enable its domains.
		// Bounded by devToolsReattachDelay; the send fails once the browser is gone.
		go func() {
			time.Sleep(devToolsReattachDelay)
			s.bootstrap()
		}()
	}
}

// close fails pending calls and stops the pump. Runs on the UI thread from
// OnBeforeClose, right before the registration is released.
func (s *devToolsSession) close() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	for id, ch := range s.pending {
		ch <- devToolsReply{err: ErrDevToolsClosed}
		delete(s.pending, id)
	}
	s.mu.Unlock()
	close(s.done)
}

// postToBrowser sends one raw message. CEF requires the UI thread for this call.
func (s *devToolsSession) postToBrowser(msg []byte) error {
	ptr := s.browser.retainPtr()
	if ptr == nil {
		return ErrDevToolsClosed
	}
	defer releaseBase(unsafe.Pointer(ptr))

	cMsg := C.CBytes(msg)
	defer C.free(cMsg)

	var accepted C.int
	if !runOnUiThread(func() {
		accepted = C.cBridgeBrowserSendDevToolsMessage(ptr, cMsg, C.size_t(len(msg)))
	}) {
		return errors.New("devtools message could not run on the ui thread")
	}
	if accepted == 0 {
		return errors.New("devtools message was not accepted by CEF")
	}
	return nil
}

// bootstrap sends a call without waiting for its result. Its only purpose is
// to attach the agent.
func (s *devToolsSession) bootstrap() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.nextID++
	id := s.nextID
	s.mu.Unlock()

	msg, err := json.Marshal(devToolsRequest{ID: id, Method: devToolsBootstrapMethod})
	if err != nil {
		return
	}
	_ = s.send(msg)
}

func (s *devToolsSession) Call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	ch := make(chan devToolsReply, 1)

	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil, ErrDevToolsClosed
	}
	s.nextID++
	id := s.nextID
	s.pending[id] = ch
	s.mu.Unlock()

	fail := func(err error) (json.RawMessage, error) {
		s.mu.Lock()
		delete(s.pending, id)
		s.mu.Unlock()
		return nil, err
	}

	msg, err := json.Marshal(devToolsRequest{ID: id, Method: method, Params: params})
	if err != nil {
		return fail(fmt.Errorf("devtools params for %q: %w", method, err))
	}
	if err := s.send(msg); err != nil {
		return fail(err)
	}

	// DevTools replies arrive on the UI thread (see uiWaitPump)
	uw := beginUIWait()
	defer uw.stop()

	for {
		select {
		case reply := <-ch:
			return reply.result, reply.err
		case <-uw.tick():
			uw.run()
		case <-ctx.Done():
			return fail(ctx.Err())
		}
	}
}

func (s *devToolsSession) On(event string, fn func(params json.RawMessage)) func() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextSub++
	id := s.nextSub
	if s.subs[event] == nil {
		s.subs[event] = make(map[int]func(json.RawMessage))
	}
	s.subs[event][id] = fn

	return func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		if handlers := s.subs[event]; handlers != nil {
			delete(handlers, id)
			if len(handlers) == 0 {
				delete(s.subs, event)
			}
		}
	}
}

// WithDevTools registers receiver for every browser of the client, popups
// included. receiver runs on its own goroutine each time the DevTools agent
// attaches: once right after the browser is created, and again after every
// detach (navigations, renderer changes). The DevTools value stays valid across
// these cycles.
func WithDevTools(receiver func(instance Browser, dt DevTools)) ClientOption {
	return func(c *client) {
		c.devToolsReceiver = receiver
	}
}

// devToolsLookup resolves the session and receiver for a callback. The
// callback owns cefBrowser's reference; the caller releases it.
func devToolsLookup(observer *C.cef_dev_tools_message_observer_t, cefBrowser *C.cef_browser_t) (func(Browser, DevTools), *devToolsSession, Browser) {
	app := getInstance()
	if app == nil {
		return nil, nil, nil
	}

	app.clientMutex.Lock()
	var found *client
	for _, c := range app.clients {
		c.Lock()
		match := c.devToolsObserverPtr == observer
		c.Unlock()
		if match {
			found = c
			break
		}
	}
	app.clientMutex.Unlock()

	if found == nil {
		return nil, nil, nil
	}

	instance := found.browserByIdentifier(int(C.cBridgeGetBrowserIdentifier(cefBrowser)))
	b, ok := instance.(*browser)
	if !ok || b == nil {
		return nil, nil, nil
	}

	b.Lock()
	s := b.devTools
	b.Unlock()

	found.Lock()
	receiver := found.devToolsReceiver
	found.Unlock()
	return receiver, s, instance
}

// attachDevTools creates the session for a freshly created browser and
// registers the client's observer on it. Runs on the UI thread from
// OnAfterCreated.
func attachDevTools(b *browser, cefBrowser *C.cef_browser_t, observer *C.cef_dev_tools_message_observer_t) {
	s := newDevToolsSession(b)

	// The registration takes over one reference of the observer.
	addRefBase(unsafe.Pointer(observer))
	registration := C.cBridgeBrowserAddDevToolsObserver(cefBrowser, observer)

	b.Lock()
	b.devTools = s
	b.devToolsRegistration = registration
	b.Unlock()

	go s.bootstrap()
}

// detachDevTools ends the session of a browser that is closing. Runs on the UI
// thread from OnBeforeClose.
func (b *browser) detachDevTools() {
	b.Lock()
	s := b.devTools
	registration := b.devToolsRegistration
	b.devTools = nil
	b.devToolsRegistration = nil
	b.Unlock()

	if s != nil {
		s.close()
	}
	releaseBase(unsafe.Pointer(registration))
}
