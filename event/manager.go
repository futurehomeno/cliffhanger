package event

import (
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/cliffhanger/utils"
)

type Manager interface {
	// Subscribe registers a new subscription under the given ID and returns its channel.
	// Subscribers sharing an ID each get their own channel, buffer and filters rather than
	// silently inheriting the first one's - but Unsubscribe closes every channel registered under
	// that ID, so an ID is only safe to share between subscribers with the same lifetime.
	Subscribe(subID string, buffer int, filters ...Filter) chan Event
	Unsubscribe(subID string)
	Publish(event Event)
	WaitFor(timeout time.Duration, filters ...Filter) <-chan Event
}

func NewManager() Manager {
	return &manager{
		bus:        NewBus[Event](),
		waitBuffer: 10,
	}
}

type manager struct {
	bus        *Bus[Event]
	waitBuffer int
}

func (m *manager) Publish(event Event) {
	m.bus.Publish(event, func(subID string) {
		log.Warnf("[event] Subscriber %s busy, dropped domain=%s class=%s", subID, event.Domain(), event.Class())
	})
}

func (m *manager) Subscribe(subID string, buffer int, filters ...Filter) chan Event {
	predicates := make([]func(Event) bool, len(filters))
	for i, f := range filters {
		predicates[i] = f.Filter
	}

	return m.bus.Subscribe(subID, buffer, predicates...)
}

func (m *manager) Unsubscribe(subID string) {
	m.bus.Unsubscribe(subID)
}

// WaitFor returns a channel that returns the waited for event or nil on timeout.
func (m *manager) WaitFor(timeout time.Duration, filters ...Filter) <-chan Event {
	subID := uuid.New().String()
	subChannel := m.Subscribe(subID, m.waitBuffer, filters...)
	resultChannel := make(chan Event, 1)

	go func() {
		defer utils.PrintStackOnRecover("event", true)

		timer := time.NewTimer(timeout)
		defer timer.Stop()
		defer m.Unsubscribe(subID)

		for {
			select {
			case e := <-subChannel:
				resultChannel <- e

				return
			case <-timer.C:
				resultChannel <- nil

				return
			}
		}
	}()

	return resultChannel
}
