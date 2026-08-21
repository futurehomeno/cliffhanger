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
	assert.Equal(t, 1, predicateCalls, "the sweep must remain once per process")
}
