package event

import (
	"sync"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

type Event interface {
	Equal(Event) bool
}

func New(domain, eventType, subjectID string) Event {
	return &event{
		domain:    domain,
		eventType: eventType,
		subjectID: subjectID,
	}
}

type event struct {
	domain    string
	eventType string
	subjectID string
}

func (e *event) Equal(compared Event) bool {
	c, ok := compared.(*event)
	if !ok {
		return false
	}

	return e.domain == c.domain && e.eventType == c.eventType && e.subjectID == c.subjectID
}

type Manager interface {
	Subscribe(subID string, buffer int) chan Event
	Unsubscribe(subID string)
	Publish(event Event)
	WaitFor(waitFor Event, timeout time.Duration) <-chan Event
}

func NewManager() Manager {
	return &manager{
		lock:        &sync.RWMutex{},
		subChannels: make(map[string]chan Event),
		waitBuffer:  10,
	}
}

type manager struct {
	lock        *sync.RWMutex
	subChannels map[string]chan Event
	waitBuffer  int
}

func (m *manager) Publish(event Event) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	for id, subChannel := range m.subChannels {
		select {
		case subChannel <- event:
			continue
		default:
			log.Warnf("event manager: event listener ID %s is busy, an event was dropped", id)
		}
	}
}

func (m *manager) Subscribe(subID string, buffer int) chan Event {
	m.lock.Lock()
	defer m.lock.Unlock()

	// Returning already existing subscription channel if it exists.
	if _, ok := m.subChannels[subID]; ok {
		return m.subChannels[subID]
	}

	subChannel := make(chan Event, buffer)

	m.subChannels[subID] = subChannel

	return subChannel
}

func (m *manager) Unsubscribe(subID string) {
	m.lock.Lock()
	defer m.lock.Unlock()

	if _, ok := m.subChannels[subID]; !ok {
		return
	}

	delete(m.subChannels, subID)
}

// WaitFor returns a channel that returns the waited for event or nil on timeout.
func (m *manager) WaitFor(waitFor Event, timeout time.Duration) <-chan Event {
	subID := uuid.New().String()
	subChannel := m.Subscribe(subID, m.waitBuffer)
	resultChannel := make(chan Event, 1)

	go func() {
		timer := time.NewTimer(timeout)
		defer timer.Stop()
		defer m.Unsubscribe(subID)

		for {
			select {
			case e := <-subChannel:
				if e.Equal(waitFor) {
					resultChannel <- e

					return
				}

				continue

			case <-timer.C:
				resultChannel <- nil

				return
			}
		}
	}()

	return resultChannel
}
