package event_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/event"
)

func TestManager_Publish(t *testing.T) {
	t.Parallel()

	tcs := []struct {
		name    string
		publish []event.Event
		subID   string
		buffer  int
		want    []event.Event
	}{
		{
			name: "Published events overflowing subscription buffer",
			publish: []event.Event{
				event.New("test1", "test1", "test1"),
				event.New("test2", "test2", "test2"),
				event.New("test3", "test3", "test3"),
			},
			subID:  "test",
			buffer: 2,
			want: []event.Event{
				event.New("test1", "test1", "test1"),
				event.New("test2", "test2", "test2"),
			},
		},
	}

	for _, tc := range tcs {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			manager := event.NewManager()

			subscription := manager.Subscribe(tc.subID, tc.buffer)

			for _, e := range tc.publish {
				manager.Publish(e)
			}

			manager.Unsubscribe(tc.subID)

			close(subscription)

			var got []event.Event

			for e := range subscription {
				got = append(got, e)
			}

			assert.Equal(t, tc.want, got)
		})
	}
}

func TestManager_Subscribe_Unsubscribe(t *testing.T) {
	t.Parallel()

	manager := event.NewManager()

	sub1 := manager.Subscribe("test", 2)
	sub2 := manager.Subscribe("test", 3)

	assert.Equal(t, sub1, sub2)
	assert.Equal(t, 2, cap(sub2))

	assert.NotPanics(t, func() {
		manager.Unsubscribe("test")
		manager.Unsubscribe("test")
	})
}

func TestManager_WaitFor(t *testing.T) {
	t.Parallel()

	tcs := []struct {
		name    string
		waitFor event.Event
		timeout time.Duration
		publish event.Event
		want    event.Event
	}{
		{
			name:    "Published waited for event",
			waitFor: event.New("test", "test", "test"),
			timeout: 100 * time.Millisecond,
			publish: event.New("test", "test", "test"),
			want:    event.New("test", "test", "test"),
		},
		{
			name:    "Published event with different values",
			waitFor: event.New("test", "test", "test"),
			timeout: 100 * time.Millisecond,
			publish: event.New("test_other", "test_other", "test_other"),
			want:    nil,
		},
		{
			name:    "Published event with different type",
			waitFor: &testEvent{},
			timeout: 100 * time.Millisecond,
			publish: event.New("test", "test", "test"),
			want:    nil,
		},
	}

	for _, tc := range tcs {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			manager := event.NewManager()
			waitFor := manager.WaitFor(tc.waitFor, tc.timeout)

			if tc.publish != nil {
				manager.Publish(tc.publish)
			}

			assert.Equal(t, tc.want, <-waitFor)
		})
	}
}

type testEvent struct{}

func (e *testEvent) Equal(event.Event) bool {
	return false
}
