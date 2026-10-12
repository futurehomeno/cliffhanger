package adapter

import (
	"errors"
	"sync"
	"testing"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/cliffhanger/prime"
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
	a.excludeStaleNodes(hubClient{err: errors.New("timeout")})

	assert.False(t, a.staleNodesSwept.Load(), "a failed hub fetch must leave the sweep retryable")
}

type hubClient struct {
	prime.Client

	devices prime.Devices
	err     error
}

func (c hubClient) GetDevices() (prime.Devices, error) { return c.devices, c.err }

type failingPublisher struct{ stubPublisher }

func (failingPublisher) PublishAdapterMessage(*fimpgo.FimpMessage) error {
	return errors.New("publish failed")
}

// TestExcludeStaleNodes_FailedExclusionReleasesClaim pins that a node whose exclusion failed is
// retried by a later sync instead of staying on the hub until the process restarts.
func TestExcludeStaleNodes_FailedExclusionReleasesClaim(t *testing.T) {
	t.Parallel()

	s, err := NewState(t.TempDir())
	require.NoError(t, err)

	a := &adapter{
		publisher:   failingPublisher{},
		state:       s,
		name:        "mill",
		address:     "1",
		lock:        &sync.RWMutex{},
		things:      map[string]Thing{},
		initialized: true,
	}

	devices := prime.Devices{{
		FIMP:     prime.DeviceFIMP{Address: "2"},
		Services: map[fimptype.ServiceNameT]*prime.Service{"thermostat": {Addr: "pt:j1/mt:evt/rt:dev/rn:mill/ad:1/sv:thermostat/ad:2"}},
	}}

	a.staleNodesSwept.Store(true)
	a.excludeStaleNodes(hubClient{devices: devices})

	assert.False(t, a.staleNodesSwept.Load(), "a failed exclusion must leave the sweep retryable")
}
