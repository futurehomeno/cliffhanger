package utils

import "sync"

// Throttle collapses runs of an identical message into a single emission: Do runs
// emit only when key differs from the previous call, so a persistent error (e.g. a
// revoked token repeated on every retry) is logged once instead of forever.
type Throttle struct {
	mu      sync.Mutex
	started bool
	last    string
}

// Do invokes emit unless key matches the last key passed to Do.
func (t *Throttle) Do(key string, emit func()) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// started distinguishes never-called from a genuine previous call with key=="", which
	// the zero value of last cannot: without it, a first call with an empty key would be
	// mistaken for a repeat of itself and silently skip emission.
	if t.started && key == t.last {
		return
	}

	t.started = true
	t.last = key
	emit()
}

// Reset forgets the last key so the next Do always emits.
func (t *Throttle) Reset() {
	t.mu.Lock()
	t.started = false
	t.last = ""
	t.mu.Unlock()
}
