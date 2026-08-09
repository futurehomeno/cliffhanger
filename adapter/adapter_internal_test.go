package adapter

import (
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubPublisher swallows everything the adapter publishes; the destroy tests only care about
// what happened to the thing, not about the exclusion report reaching MQTT.
type stubPublisher struct{}

func (stubPublisher) PublishAdapterMessage(*fimpgo.FimpMessage) error          { return nil }
func (stubPublisher) PublishThingMessage(Thing, *fimpgo.FimpMessage) error     { return nil }
func (stubPublisher) PublishThingEvent(ThingEvent)                             {}
func (stubPublisher) PublishServiceMessage(Service, *fimpgo.FimpMessage) error { return nil }
func (stubPublisher) PublishServiceEvent(Service, ServiceEvent)                {}

// recordingConnector is a ControllableConnector that records whether the thing was disconnected.
type recordingConnector struct {
	lock         sync.Mutex
	disconnected bool
}

func (c *recordingConnector) Connectivity() *ConnectivityDetails { return &ConnectivityDetails{} }
func (c *recordingConnector) Ping() *PingDetails                 { return &PingDetails{} }
func (c *recordingConnector) Connect(Thing)                      {}

func (c *recordingConnector) Disconnect(Thing) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.disconnected = true
}

func (c *recordingConnector) wasDisconnected() bool {
	c.lock.Lock()
	defer c.lock.Unlock()

	return c.disconnected
}

// failingRemoveState stands in for a state file that can no longer be persisted. It mirrors
// state.remove after a failed save: the record is restored, so removal fails with it intact.
type failingRemoveState struct {
	State
}

func (f failingRemoveState) remove(string) error {
	return errors.New("state write failed")
}

// TestDestroyThing_UnregistersWhenStateRemoveFails pins that a failed state write does not skip
// the unregister. Returning early there left the thing connected while DestroyAllThings cleared
// the map, dropping the last reference to a live connector and leaking it for the process
// lifetime. The kept record and the unregistered thing form the ghost EnsureThings heals.
func TestDestroyThing_UnregistersWhenStateRemoveFails(t *testing.T) {
	t.Parallel()

	s, err := NewState(t.TempDir())
	require.NoError(t, err)

	ts, err := s.add(&thingStateModel{ID: "B", Address: "2"})
	require.NoError(t, err)

	connector := &recordingConnector{}

	thing := NewThing(stubPublisher{}, ts, &ThingConfig{
		InclusionReport: &fimptype.ThingInclusionReport{Address: "2"},
		Connector:       connector,
	})

	a := &adapter{
		publisher: stubPublisher{},
		state:     failingRemoveState{State: s},
		things:    map[string]Thing{"2": thing},
		lock:      &sync.RWMutex{},
	}

	err = a.DestroyAllThings()

	assert.Error(t, err, "the failed state write must still surface")
	assert.True(t, connector.wasDisconnected(), "the connector must be released even when the state write fails")
	assert.Empty(t, a.things, "the thing must not linger in the adapter")
}

// TestEnsureThings_RecreatesGhostThing pins that a state record without a live thing - left by a
// destroy whose state write failed after the thing was unregistered - is recreated once its
// device is selected again, instead of being skipped as present until a restart.
func TestEnsureThings_RecreatesGhostThing(t *testing.T) {
	t.Parallel()

	s, err := NewState(t.TempDir())
	require.NoError(t, err)

	_, err = s.add(&thingStateModel{ID: "B", Address: "2", Info: json.RawMessage(`{"groups":["g1"]}`)})
	require.NoError(t, err)

	a := &adapter{
		publisher: stubPublisher{},
		state:     s,
		factory:   groupsFactory{},
		things:    map[string]Thing{},
		lock:      &sync.RWMutex{},
	}

	seeds := ThingSeeds{{ID: "B", CustomAddress: "2", Info: groupsInfo{Groups: []string{"g1"}}}}

	require.NoError(t, a.EnsureThings(seeds), "healing must not report an error")
	assert.Empty(t, a.things, "before initialization nothing is registered, so healing must not fire")

	a.initialized = true

	require.NoError(t, a.EnsureThings(seeds))
	assert.NotNil(t, a.things["2"], "the ghost record must be recreated as a live thing")
	require.NotNil(t, a.state.byID("B"), "the record must survive the recreate")
}

// TestEnsureThings_HealsGhostWithoutLosingAddressOrState pins that healing a ghost preserves its
// original address and persisted state, matching rebuildChangedThing. Without that, a seed with
// no CustomAddress (the normal case: the caller doesn't track previously-assigned addresses) got
// a freshly acquired one from createThingState, and the state blob was silently dropped.
func TestEnsureThings_HealsGhostWithoutLosingAddressOrState(t *testing.T) {
	t.Parallel()

	s, err := NewState(t.TempDir())
	require.NoError(t, err)

	ts, err := s.add(&thingStateModel{ID: "B", Address: "42", Info: json.RawMessage(`{"groups":["g1"]}`)})
	require.NoError(t, err)

	require.NoError(t, ts.SetState(map[string]string{"keep": "me"}))

	a := &adapter{
		publisher:   stubPublisher{},
		state:       s,
		factory:     groupsFactory{},
		things:      map[string]Thing{},
		lock:        &sync.RWMutex{},
		initialized: true,
	}

	// No CustomAddress: a real discovery pass supplies one only when it already knows the
	// previously-assigned address, which is exactly what a ghost heal must recover on its own.
	seeds := ThingSeeds{{ID: "B", Info: groupsInfo{Groups: []string{"g1"}}}}

	require.NoError(t, a.EnsureThings(seeds), "healing must not report an error")
	assert.NotNil(t, a.things["42"], "the ghost must be recreated at its original address, not a freshly acquired one")

	healedTS := a.state.byID("B")
	require.NotNil(t, healedTS, "the record must survive the heal")
	assert.Equal(t, "42", healedTS.Address())

	var state map[string]string
	require.NoError(t, healedTS.State(&state))
	assert.Equal(t, "me", state["keep"], "the persisted state must survive the heal")
}

// failingFactory fails the build the way a vendor call refusing mid-heal would.
type failingFactory struct{}

func (failingFactory) Create(Adapter, Publisher, ThingState) (Thing, error) {
	return nil, errors.New("factory refused")
}

// TestEnsureThings_FailedHealKeepsTheGhostRecord pins that a heal failing past createThingState
// leaves the ghost record as it was. createThingState overwrites the record, so the rollback
// removing it dropped the address and state the next heal attempt needs - the device then came
// back as brand new at a different address, exactly what the heal exists to prevent.
func TestEnsureThings_FailedHealKeepsTheGhostRecord(t *testing.T) {
	t.Parallel()

	s, err := NewState(t.TempDir())
	require.NoError(t, err)

	ts, err := s.add(&thingStateModel{ID: "B", Address: "42", Info: json.RawMessage(`{"groups":["g1"]}`)})
	require.NoError(t, err)

	require.NoError(t, ts.SetState(map[string]string{"keep": "me"}))

	a := &adapter{
		publisher:   stubPublisher{},
		state:       s,
		factory:     failingFactory{},
		things:      map[string]Thing{},
		lock:        &sync.RWMutex{},
		initialized: true,
	}

	seeds := ThingSeeds{{ID: "B", Info: groupsInfo{Groups: []string{"g1"}}}}

	assert.Error(t, a.EnsureThings(seeds), "the failed heal must surface")

	ghost := a.state.byID("B")
	require.NotNil(t, ghost, "the ghost record must survive a failed heal")
	assert.Equal(t, "42", ghost.Address())

	var state map[string]string
	require.NoError(t, ghost.State(&state))
	assert.Equal(t, "me", state["keep"], "the persisted state must survive a failed heal")

	// The retry is what the preserved record buys: the same ghost heals at its own address.
	a.factory = groupsFactory{}

	require.NoError(t, a.EnsureThings(seeds))
	assert.NotNil(t, a.things["42"], "the retry must heal the ghost at its original address")
}

// TestCreateThing_RemovesTheRecordWhenNothingPreceded pins that the rollback still removes the
// record when there was no prior one, so a plain failed create does not leave a ghost behind.
func TestCreateThing_RemovesTheRecordWhenNothingPreceded(t *testing.T) {
	t.Parallel()

	s, err := NewState(t.TempDir())
	require.NoError(t, err)

	a := &adapter{
		publisher:   stubPublisher{},
		state:       s,
		factory:     failingFactory{},
		things:      map[string]Thing{},
		lock:        &sync.RWMutex{},
		initialized: true,
	}

	assert.Error(t, a.CreateThing(&ThingSeed{ID: "B", Info: groupsInfo{Groups: []string{"g1"}}}))
	assert.Nil(t, a.state.byID("B"), "a create that never had a record must not leave one behind")
}
