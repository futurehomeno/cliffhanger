package adapter

import (
	"sync"
	"testing"

	"github.com/futurehomeno/fimpgo"
	"github.com/stretchr/testify/assert"
)

// TestExcludeStaleNodesOnce_SkipsBeforeInitialization pins the guard that keeps a device sync
// racing InitializeThings at startup from wiping the fleet: while uninitialized no thing is
// registered, so every hub node would be classified stale. The sweep must be deferred rather than
// spent, so the first sync after initialization still performs it.
func TestExcludeStaleNodesOnce_SkipsBeforeInitialization(t *testing.T) {
	t.Parallel()

	var predicateCalls int

	a := &adapter{
		lock:   &sync.RWMutex{},
		things: map[string]Thing{},
		// Only needs to be non-nil: the predicate returns false, so it is never dereferenced.
		mqtt:       &fimpgo.MqttTransport{},
		staleNodes: func() bool { predicateCalls++; return false },
	}

	a.excludeStaleNodesOnce()
	assert.Zero(t, predicateCalls, "an uninitialized adapter must not begin a sweep")

	a.lock.Lock()
	a.initialized = true
	a.lock.Unlock()

	a.excludeStaleNodesOnce()
	assert.Equal(t, 1, predicateCalls, "the deferred sweep must still be available after initialization")

	a.excludeStaleNodesOnce()
	assert.Equal(t, 2, predicateCalls, "a false predicate must not consume the once; later syncs re-check")
}

// TestExcludeStaleNodesOnce_FalsePredicateDoesNotConsumeOnce pins the runtime-toggle fix: a sync
// that runs while the sweep is disabled must leave the sweep claim free so a later sync can still
// enable it.
func TestExcludeStaleNodesOnce_FalsePredicateDoesNotConsumeOnce(t *testing.T) {
	t.Parallel()

	a := &adapter{
		lock:        &sync.RWMutex{},
		things:      map[string]Thing{},
		initialized: true,
		mqtt:        &fimpgo.MqttTransport{},
		staleNodes:  func() bool { return false },
	}

	a.excludeStaleNodesOnce()
	a.excludeStaleNodesOnce()

	assert.False(t, a.staleNodesSwept.Load(), "disabling the sweep must not permanently spend the claim")
}

// TestExcludeStaleNodes_FailedFetchReleasesClaim pins that a hub fetch failure leaves the sweep
// retryable: the claim exists to stop overlapping sweeps, not to spend the single attempt on a
// Vinculum timeout and leave stale nodes behind until the process restarts.
func TestExcludeStaleNodes_FailedFetchReleasesClaim(t *testing.T) {
	t.Parallel()

	a := &adapter{
		lock:        &sync.RWMutex{},
		things:      map[string]Thing{},
		initialized: true,
	}

	a.staleNodesSwept.Store(true)
	a.excludeStaleNodes()

	assert.False(t, a.staleNodesSwept.Load(), "a failed hub fetch must leave the sweep retryable")
}
