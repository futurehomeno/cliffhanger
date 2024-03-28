package event

import (
	"sync"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

type Manager interface {
	Subscribe(subID string, buffer int, filters ...Filter) chan Event
	Unsubscribe(subID string)
	Publish(event Event)
	WaitFor(timeout time.Duration, filters ...Filter) <-chan Event
}

func NewManager() Manager {
	return &manager{
		lock:          &sync.RWMutex{},
		subscriptions: make(map[string]*subscription),
		waitBuffer:    10,
	}
}

type manager struct {
	lock          *sync.RWMutex
	subscriptions map[string]*subscription
	waitBuffer    int
}

func (m *manager) Publish(event Event) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	for _, s := range m.subscriptions {
		// Filter event out if it doesn't match the filter.
		if !s.filter(event) {
			continue
		}

		select {
		case s.channel <- event:
			continue
		default:
			log.Warnf("event manager: event subscriber ID %s is busy, an event for domain %s and class %s was dropped", s.id, event.Domain(), event.Class())
		}
	}
}

func (m *manager) Subscribe(subID string, buffer int, filters ...Filter) chan Event {
	m.lock.Lock()
	defer m.lock.Unlock()

	// Returning already existing subscription channel if it exists.
	if _, ok := m.subscriptions[subID]; ok {
		return m.subscriptions[subID].channel
	}

	subCh := make(chan Event, buffer)

	m.subscriptions[subID] = &subscription{
		id:      subID,
		channel: subCh,
		filters: filters,
	}

	return subCh
}

func (m *manager) Unsubscribe(subID string) {
	m.lock.Lock()
	defer m.lock.Unlock()

	if _, ok := m.subscriptions[subID]; !ok {
		return
	}

	delete(m.subscriptions, subID)
}

// WaitFor returns a channel that returns the waited for event or nil on timeout.
func (m *manager) WaitFor(timeout time.Duration, filters ...Filter) <-chan Event {
	subID := uuid.New().String()
	subChannel := m.Subscribe(subID, m.waitBuffer, filters...)
	resultChannel := make(chan Event, 1)

	go func() {
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
