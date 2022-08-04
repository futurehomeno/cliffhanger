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

	processor := event.ProcessorFn(func(e *event.Event) {
		assert.Equal(t, e.Domain, "test3")
		assert.Equal(t, e.Payload, "test3")

		close(finishCh)
	})

	manager := event.NewManager()

	listener := event.NewListener(
		processor,
		manager,
		"test_sub_id",
		10,
		event.WaitForDomain("test3"),
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
