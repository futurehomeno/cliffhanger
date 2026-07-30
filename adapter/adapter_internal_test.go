package adapter

import (
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
// state.remove faithfully: the in-memory record is dropped first and only the save fails, which
// is why every caller of destroyThing must treat the error as already-applied rather than retry.
type failingRemoveState struct {
	State
}

func (f failingRemoveState) remove(id string) error {
	_ = f.State.remove(id)

	return errors.New("state write failed")
}

// TestDestroyThing_UnregistersWhenStateRemoveFails pins that a failed state write does not skip
// the unregister. Returning early there left the thing connected while DestroyAllThings cleared
// the map, dropping the last reference to a live connector and leaking it for the process
// lifetime.
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
