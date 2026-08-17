package event

import (
	"runtime/debug"
	"sync"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
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
		lock:          &sync.RWMutex{},
		subscriptions: make(map[string][]*subscription),
		waitBuffer:    10,
	}
}

type manager struct {
	lock          *sync.RWMutex
	subscriptions map[string][]*subscription
	waitBuffer    int
}

func (m *manager) Publish(event Event) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	for _, subscriptions := range m.subscriptions {
		for _, s := range subscriptions {
			// Filter event out if it doesn't match the filter.
			if !s.filter(event) {
				continue
			}

			select {
			case s.channel <- event:
				continue
			default:
				log.Warnf("[cliff] Event subscriber ID=%s busy, event domain=%s class=%s dropped", s.id, event.Domain(), event.Class())
			}
		}
	}
}

func (m *manager) Subscribe(subID string, buffer int, filters ...Filter) chan Event {
	m.lock.Lock()
	defer m.lock.Unlock()

	subCh := make(chan Event, buffer)

	m.subscriptions[subID] = append(m.subscriptions[subID], &subscription{
		id:      subID,
		channel: subCh,
		filters: filters,
	})

	return subCh
}

func (m *manager) Unsubscribe(subID string) {
	m.lock.Lock()
	defer m.lock.Unlock()

	for _, s := range m.subscriptions[subID] {
		close(s.channel)
	}

	delete(m.subscriptions, subID)
}

// WaitFor returns a channel that returns the waited for event or nil on timeout.
func (m *manager) WaitFor(timeout time.Duration, filters ...Filter) <-chan Event {
	subID := uuid.New().String()
	subChannel := m.Subscribe(subID, m.waitBuffer, filters...)
	resultChannel := make(chan Event, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Error(string(debug.Stack()))
				log.Error(r)
				panic(r)
			}
		}()

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

type subscription struct {
	id      string
	channel chan Event
	filters []Filter
}

func (s *subscription) filter(event Event) bool {
	for _, f := range s.filters {
		if !f.Filter(event) {
			return false
		}
	}

	return true
}
