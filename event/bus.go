package event

import (
	"sync"
)

// Bus is the subscriber registry behind both the event manager and the application lifecycle.
// Subscribers sharing an ID each get their own channel, buffer and filters rather than silently
// inheriting the first one's - but Unsubscribe closes every channel registered under that ID, so
// an ID is only safe to share between subscribers with the same lifetime.
type Bus[T any] struct {
	lock          sync.RWMutex
	subscriptions map[string][]*subscription[T]
}

func NewBus[T any]() *Bus[T] {
	return &Bus[T]{subscriptions: make(map[string][]*subscription[T])}
}

// Subscribe registers a new subscription under the given ID and returns its channel.
func (b *Bus[T]) Subscribe(subID string, buffer int, filters ...func(T) bool) chan T {
	b.lock.Lock()
	defer b.lock.Unlock()

	channel := make(chan T, buffer)

	b.subscriptions[subID] = append(b.subscriptions[subID], &subscription[T]{channel: channel, filters: filters})

	return channel
}

// Unsubscribe closes and drops every subscription registered under the given ID.
func (b *Bus[T]) Unsubscribe(subID string) {
	b.lock.Lock()
	defer b.lock.Unlock()

	for _, s := range b.subscriptions[subID] {
		close(s.channel)
	}

	delete(b.subscriptions, subID)
}

// Publish delivers a value to every matching subscriber without blocking. A subscriber whose
// buffer is full has the value dropped and is reported to dropped, so that the caller can log it
// in its own terms.
func (b *Bus[T]) Publish(value T, dropped func(subID string)) {
	b.lock.RLock()
	defer b.lock.RUnlock()

	for subID, subscriptions := range b.subscriptions {
		for _, s := range subscriptions {
			if !s.matches(value) {
				continue
			}

			select {
			case s.channel <- value:
			default:
				dropped(subID)
			}
		}
	}
}

type subscription[T any] struct {
	channel chan T
	filters []func(T) bool
}

func (s *subscription[T]) matches(value T) bool {
	for _, f := range s.filters {
		if !f(value) {
			return false
		}
	}

	return true
}
