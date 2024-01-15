package event_test

import (
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
