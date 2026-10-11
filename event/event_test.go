package event_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/event"
)

func TestFilters_Payload(t *testing.T) {
	t.Parallel()

	type payload struct{ ID string }

	withPayload := event.NewWithPayload("thing", "updated", payload{ID: "1"})
	plain := event.New("thing", "updated")

	p, ok := withPayload.(event.EventWithPayload)
	assert.True(t, ok)
	assert.Equal(t, payload{ID: "1"}, p.Payload())
	assert.Equal(t, "thing", withPayload.Domain())
	assert.Equal(t, "updated", withPayload.Class())

	assert.True(t, event.WaitForPayload(payload{ID: "1"}).Filter(withPayload))
	assert.False(t, event.WaitForPayload(payload{ID: "2"}).Filter(withPayload))
	assert.False(t, event.WaitForPayload(nil).Filter(plain), "an event without payload never matches")

	assert.True(t, event.WaitFor[event.EventWithPayload]().Filter(withPayload))
	assert.False(t, event.WaitFor[event.EventWithPayload]().Filter(plain))
	assert.True(t, event.WaitFor[event.Event]().Filter(plain))
}

func TestManager_SubscribeIgnoresNilFilter(t *testing.T) {
	t.Skip("openspec/issues/messaging-event-manager-nil-filter-panics.md")

	m := event.NewManager()
	ch := m.Subscribe("nil_filter", 1, nil, event.WaitForDomain("d"))

	m.Publish(event.New("d", "c"))

	assert.Len(t, ch, 1)
}
