package event_test

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/event"
)

func TestListener(t *testing.T) {
	t.Parallel()

	finishCh := make(chan struct{})

	processor := event.ProcessorFn(func(e event.Event) {
		assert.Equal(t, e.Domain(), "test3")
		assert.Equal(t, e.Class(), "test3")

		close(finishCh)
	})

	manager := event.NewManager()

	listener := event.NewListener(
		manager,
		event.NewHandler(
			processor,
			"test_sub_id",
			10,
			event.WaitForDomain("test3"),
		),
	)

	err := listener.Start()
	assert.NoError(t, err)

	err = listener.Start()
	assert.Error(t, err)

	manager.Publish(event.New("test1", "test1"))
	manager.Publish(event.New("test2", "test2"))
	manager.Publish(event.New("test3", "test3"))

	select {
	case <-finishCh:
		break
	case <-time.After(time.Second):
		assert.Fail(t, "timeout")
	}

	err = listener.Stop()
	assert.NoError(t, err)

	err = listener.Stop()
	assert.Error(t, err)
}

func TestListener_Process(t *testing.T) {
	t.Parallel()

	finishCh := make(chan struct{})

	processor := event.ProcessorFn(func(e event.Event) {
		assert.Equal(t, e.Domain(), "test")

		if e.Class() == "test1" {
			panic("test panic")
		}

		close(finishCh)
	})

	manager := event.NewManager()

	listener := event.NewListener(
		manager,
		event.NewHandler(
			processor,
			"test_sub_id",
			10,
			event.WaitForDomain("test"),
		),
	)

	err := listener.Start()
	assert.NoError(t, err)

	manager.Publish(event.New("test", "test1"))
	manager.Publish(event.New("test", "test2"))

	select {
	case <-finishCh:
		break
	case <-time.After(1 * time.Second):
		assert.Fail(t, "timeout")
	}

	err = listener.Stop()
	assert.NoError(t, err)
}

func TestListener_StopUnsubscribesAndRestarts(t *testing.T) {
	t.Parallel()

	received := make(chan event.Event, 10)
	manager := event.NewManager()

	listener := event.NewListener(manager, event.NewHandler(
		event.ProcessorFn(func(e event.Event) { received <- e }),
		"listener_restart",
		10,
	))

	assert.NoError(t, listener.Start())

	sibling := manager.Subscribe("listener_restart", 1)

	manager.Publish(event.New("d", "first"))
	assert.Equal(t, "first", (<-received).Class())

	assert.NoError(t, listener.Stop())
	assert.Equal(t, "first", (<-sibling).Class())

	select {
	case _, open := <-sibling:
		assert.False(t, open, "Stop unsubscribes the handler ID, closing every channel under it")
	case <-time.After(time.Second):
		assert.Fail(t, "Stop must unsubscribe the handler ID, closing every channel under it")
	}

	assert.NoError(t, listener.Start())

	manager.Publish(event.New("d", "second"))

	select {
	case e := <-received:
		assert.Equal(t, "second", e.Class())
	case <-time.After(time.Second):
		assert.Fail(t, "a restarted listener must receive events again")
	}

	assert.NoError(t, listener.Stop())
}

func TestListener_HandlerExitsWhenSubscriptionIsClosed(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32

	manager := event.NewManager()
	listener := event.NewListener(manager, event.NewHandler(event.ProcessorFn(func(event.Event) { calls.Add(1) }), "listener_closed", 1))

	assert.NoError(t, listener.Start())

	manager.Unsubscribe("listener_closed")

	assert.Never(t, func() bool { return calls.Load() > 0 }, 50*time.Millisecond, time.Millisecond,
		"a closed subscription must end the handler, not feed it nil events")
	assert.NoError(t, listener.Stop())
}
