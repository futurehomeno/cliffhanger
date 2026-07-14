package utils

import "sync"

// Throttle collapses runs of an identical message into a single emission: Do runs
// emit only when key differs from the previous call, so a persistent error (e.g. a
// revoked token repeated on every retry) is logged once instead of forever.
type Throttle struct {
	mu   sync.Mutex
	last string
}

// Do invokes emit unless key matches the last key passed to Do.
func (t *Throttle) Do(key string, emit func()) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if key == t.last {
		return
	}

	t.last = key
	emit()
}

// Reset forgets the last key so the next Do always emits.
func (t *Throttle) Reset() {
	t.mu.Lock()
	t.last = ""
	t.mu.Unlock()
}
