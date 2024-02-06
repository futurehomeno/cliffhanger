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
		filters []event.Filter
		buffer  int
		want    []event.Event
	}{
		{
			name: "Published events overflowing subscription buffer",
			publish: []event.Event{
				event.New("test1", "test1"),
				event.New("test2", "test2"),
				event.New("test3", "test3"),
			},
			subID:   "test",
			buffer:  2,
			filters: nil,
			want: []event.Event{
				event.New("test1", "test1"),
				event.New("test2", "test2"),
			},
		},
		{
			name: "Published events filtered out",
			publish: []event.Event{
				event.New("test1", "test1"),
				event.New("test2", "test2"),
				event.New("test3", "test3"),
			},
			subID:   "test",
			buffer:  3,
			filters: []event.Filter{event.Or(event.WaitForDomain("test1"), event.WaitForDomain("test2"))},
			want: []event.Event{
				event.New("test1", "test1"),
				event.New("test2", "test2"),
			},
		},
		{
			name: "Published events filtered out",
			publish: []event.Event{
				event.New("test1", "test1"),
				event.New("test2", "test2"),
				event.New("test3", "test3"),
			},
			subID:   "test",
			buffer:  3,
			filters: []event.Filter{event.And(event.WaitForDomain("test1"), event.WaitForClass("test1"))},
			want: []event.Event{
				event.New("test1", "test1"),
			},
		},
	}

	for _, tc := range tcs {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			manager := event.NewManager()

			subscription := manager.Subscribe(tc.subID, tc.buffer, tc.filters...)

			for _, e := range tc.publish {
				manager.Publish(e)
			}

			manager.Unsubscribe(tc.subID)

			close(subscription)

			var got []event.Event

			for e := range subscription {
				got = append(got, e)
			}

			assert.Equal(t, len(tc.want), len(got))

			for i, e := range got {
				assert.Equal(t, tc.want[i].Domain(), e.Domain())
				assert.Equal(t, tc.want[i].Class(), e.Class())
			}
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
		filters []event.Filter
		timeout time.Duration
		publish event.Event
		want    event.Event
	}{
		{
			name:    "Published waited for event",
			filters: []event.Filter{event.WaitForDomain("test1"), event.WaitForClass("test1")},
			timeout: 100 * time.Millisecond,
			publish: event.New("test1", "test1"),
			want:    event.New("test1", "test1"),
		},
		{
			name:    "Published event with different values",
			filters: []event.Filter{event.WaitForDomain("test1"), event.WaitForClass("test1")},
			timeout: 100 * time.Millisecond,
			publish: event.New("test2", "test2"),
			want:    nil,
		},
	}

	for _, tc := range tcs {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			manager := event.NewManager()
			waitFor := manager.WaitFor(tc.timeout, tc.filters...)

			if tc.publish != nil {
				manager.Publish(tc.publish)
			}

			assert.Equal(t, tc.want, <-waitFor)
		})
	}
}
