package event_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/event"
)

func TestBus_PublishDropsWithoutACallback(t *testing.T) {
	t.Parallel()

	bus := event.NewBus[int]()
	bus.Subscribe("test", 1)

	assert.NotPanics(t, func() {
		// The second one has nowhere to go: a nil callback must not turn a subscriber falling
		// behind into a panic.
		bus.Publish(1, nil)
		bus.Publish(2, nil)
	})
}

func TestBus_PublishReportsDropsAndAppliesFilters(t *testing.T) {
	t.Parallel()

	bus := event.NewBus[int]()
	bus.Subscribe("odd", 1, func(v int) bool { return v%2 == 1 })

	var drops []string

	record := func(subID string) { drops = append(drops, subID) }

	bus.Publish(2, record)
	assert.Empty(t, drops, "a filtered out value must not count as a drop")

	bus.Publish(1, record)
	bus.Publish(3, record)
	assert.Equal(t, []string{"odd"}, drops, "a full subscriber must be reported once")
}

func TestBus_SubscribeDoesNotAliasTheCallersFilters(t *testing.T) {
	t.Parallel()

	bus := event.NewBus[int]()

	filters := []func(int) bool{func(v int) bool { return v == 1 }}
	sub := bus.Subscribe("test", 1, filters...)

	// A variadic call made with someSlice... hands over the caller's backing array.
	filters[0] = func(int) bool { return false }

	bus.Publish(1, nil)
	assert.Len(t, sub, 1, "a subscription must keep the predicates it was registered with")
}
