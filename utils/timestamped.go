package utils

import (
	"sync"
	"time"
)

// Timestamped is a concurrency-safe last-value holder rejecting out-of-order updates,
// e.g. streamed observations arriving after a fresher poll. An update carrying a timestamp
// equal to the held one wins, matching last-write-wins semantics of the source adapters.
//
// The lock guards the timestamp ordering, not the value itself: for a T holding references
// (slice, map, pointer) the caller must not mutate what it passed to Set or got from Get.
type Timestamped[T any] struct {
	mu    sync.RWMutex
	value T
	ts    time.Time
	set   bool
}

// Set stores the value unless a fresher one is already held, reporting whether it was stored.
func (c *Timestamped[T]) Set(value T, ts time.Time) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.set && ts.Before(c.ts) {
		return false
	}

	c.value, c.ts, c.set = value, ts, true

	return true
}

// Get returns the held value and its timestamp, with false if nothing was stored yet.
func (c *Timestamped[T]) Get() (T, time.Time, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.value, c.ts, c.set
}
