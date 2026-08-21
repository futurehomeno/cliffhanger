package adapter

import (
	"encoding/json"
	"errors"
	"strconv"
	"sync"
	"testing"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/cliffhanger/storage"
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

// TestDestroyAllThings_FailedSaveKeepsStateRecords pins that a reset whose state write fails
// leaves the record in memory so the next sync retries the destroy. Wrapping the pass in
// state.batch shadowed the Save inside state.remove - the very call whose failure restores the
// record - so memory dropped every record while the disk kept them all, and the next boot
// resurrected the fleet the reset was meant to clear.
func TestDestroyAllThings_FailedSaveKeepsStateRecords(t *testing.T) {
	t.Parallel()

	failing := &failingSaveStorage{Storage: storage.NewState(&adapterStateModel{}, t.TempDir(), "adapter.json")}
	s := &state{Storage: failing}

	ts, err := s.add(&thingStateModel{ID: "A"})
	require.NoError(t, err)

	thing := NewThing(stubPublisher{}, ts, &ThingConfig{
		InclusionReport: &fimptype.ThingInclusionReport{Address: ts.Address()},
		Connector:       &recordingConnector{},
	})

	a := &adapter{
		publisher: stubPublisher{},
		state:     s,
		things:    map[string]Thing{ts.Address(): thing},
		lock:      &sync.RWMutex{},
	}

	failing.failSave = true

	err = a.DestroyAllThings()

	assert.Error(t, err, "the failed state write must surface")
	assert.Empty(t, a.things, "the adapter must still be left empty")
	assert.NotEmpty(t, s.all(), "the record must survive so the next sync retries the destroy")
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

// TestCreateThing_FailedStateWriteKeepsTheGhostRecord pins the window the rollback defer cannot
// cover: createThingState overwrites the record and then fails to persist it, all before the
// defer is registered, so the ghost would keep an address that only ever existed in memory.
func TestCreateThing_FailedStateWriteKeepsTheGhostRecord(t *testing.T) {
	t.Parallel()

	failing := &failingSaveStorage{Storage: storage.NewState(&adapterStateModel{}, t.TempDir(), "adapter.json")}
	s := &state{Storage: failing}

	_, err := s.add(&thingStateModel{ID: "B", Address: "42"})
	require.NoError(t, err)

	a := &adapter{
		publisher:   stubPublisher{},
		state:       s,
		factory:     failingFactory{},
		things:      map[string]Thing{},
		lock:        &sync.RWMutex{},
		initialized: true,
	}

	failing.failSave = true

	// CustomAddress keeps the address assignment, which would consume an index, out of the way so
	// that the record write is the step that fails.
	require.Error(t, a.createThing(&ThingSeed{ID: "B", CustomAddress: "99"}))
	assert.Equal(t, "42", s.modelByID("B").Address, "the ghost must keep its address when the state write fails")
}

// countingFactory builds things the way groupsFactory does and records how many it was asked for.
type countingFactory struct{ created int }

func (f *countingFactory) Create(a Adapter, publisher Publisher, ts ThingState) (Thing, error) {
	f.created++

	return groupsFactory{}.Create(a, publisher, ts)
}

// TestInitializeThings_KeepsAlreadyRegisteredThings pins that initialization does not rebuild a
// thing that is already live. The router starts before the initialization task, so an inbound
// command can create one in between; rebuilding it replaced a connected instance with a duplicate,
// announced it a second time and left the original's connector open with nothing referencing it.
func TestInitializeThings_KeepsAlreadyRegisteredThings(t *testing.T) {
	t.Parallel()

	s, err := NewState(t.TempDir())
	require.NoError(t, err)

	ts, err := s.add(&thingStateModel{ID: "B", Address: "2"})
	require.NoError(t, err)

	connector := &recordingConnector{}
	live := NewThing(stubPublisher{}, ts, &ThingConfig{
		InclusionReport: &fimptype.ThingInclusionReport{Address: "2"},
		Connector:       connector,
	})

	factory := &countingFactory{}
	a := &adapter{
		publisher: stubPublisher{},
		state:     s,
		factory:   factory,
		things:    map[string]Thing{"2": live},
		lock:      &sync.RWMutex{},
	}

	require.NoError(t, a.InitializeThings())

	assert.Equal(t, 0, factory.created, "a live thing must not be rebuilt")
	assert.Same(t, live, a.things["2"], "the live instance must be kept")
	assert.False(t, connector.wasDisconnected())
}

// failOnNthFactory builds things the way groupsFactory does but fails the nth Create, so a
// mid-loop failure can be exercised without depending on map iteration order.
type failOnNthFactory struct {
	n       int
	created int
}

func (f *failOnNthFactory) Create(a Adapter, publisher Publisher, ts ThingState) (Thing, error) {
	f.created++

	if f.created == f.n {
		return nil, errors.New("factory refused")
	}

	return groupsFactory{}.Create(a, publisher, ts)
}

// TestInitializeThings_RegistersIncrementally pins that a failure partway through leaves the
// things built before it registered. Collecting them all first meant one bad record dropped every
// thing, while their inclusion reports had already gone out. The factory fails on its second call
// rather than on a particular record, because state.all() ranges over a map and has no order.
func TestInitializeThings_RegistersIncrementally(t *testing.T) {
	t.Parallel()

	s, err := NewState(t.TempDir())
	require.NoError(t, err)

	for i, id := range []string{"A", "B", "C"} {
		_, err = s.add(&thingStateModel{ID: id, Address: strconv.Itoa(i + 1)})
		require.NoError(t, err)
	}

	a := &adapter{
		publisher: stubPublisher{},
		state:     s,
		factory:   &failOnNthFactory{n: 2},
		things:    map[string]Thing{},
		lock:      &sync.RWMutex{},
	}

	require.Error(t, a.InitializeThings(), "the refused build must surface")
	assert.Len(t, a.things, 1, "the thing built before the failure must stay registered")
	assert.False(t, a.initialized, "a failed initialization must not mark the adapter initialized")
}

// failingReportThing fails its inclusion report a given number of times before succeeding.
type failingReportThing struct {
	Thing

	failures int
	sent     int
}

func (f *failingReportThing) SendInclusionReport(bool) (bool, error) {
	f.sent++

	if f.sent <= f.failures {
		return false, errors.New("publish refused")
	}

	return true, nil
}

// reportFactory hands out a thing whose inclusion report fails the first time.
type reportFactory struct{ thing *failingReportThing }

func (f *reportFactory) Create(_ Adapter, _ Publisher, _ ThingState) (Thing, error) {
	return f.thing, nil
}

// TestInitializeThings_AnnounceFailureAllowsRetry pins that a thing whose inclusion report failed
// is not registered. Registering first made the live check skip it on every later attempt, so the
// hub never heard about it while the adapter counted it as done.
func TestInitializeThings_AnnounceFailureAllowsRetry(t *testing.T) {
	t.Parallel()

	s, err := NewState(t.TempDir())
	require.NoError(t, err)

	ts, err := s.add(&thingStateModel{ID: "A", Address: "1"})
	require.NoError(t, err)

	thing := &failingReportThing{
		Thing: NewThing(stubPublisher{}, ts, &ThingConfig{
			InclusionReport: &fimptype.ThingInclusionReport{Address: "1"},
			Connector:       &recordingConnector{},
		}),
		failures: 1,
	}

	a := &adapter{
		publisher: stubPublisher{},
		state:     s,
		factory:   &reportFactory{thing: thing},
		things:    map[string]Thing{},
		lock:      &sync.RWMutex{},
	}

	require.Error(t, a.InitializeThings(), "the failed report must surface")
	assert.Empty(t, a.things, "a thing that was never announced must not be registered")

	require.NoError(t, a.InitializeThings(), "the retry must announce the thing again")
	assert.NotNil(t, a.things["1"], "the retry must register it once announced")
	assert.Equal(t, 2, thing.sent, "the report must actually be retried")
}

// TestThingServiceByTopic_ResolvesInboundTopics pins that the service index lookup matches both a
// bare service address and a full inbound message topic, and misses anything else.
func TestThingServiceByTopic_ResolvesInboundTopics(t *testing.T) {
	t.Parallel()

	address := "/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2"

	svc := NewService(stubPublisher{}, &fimptype.Service{Name: "out_lvl_switch", Address: address})

	thing := NewThing(stubPublisher{}, nil, &ThingConfig{
		InclusionReport: &fimptype.ThingInclusionReport{Address: "1"},
	}, svc)

	assert.Same(t, svc, thing.ServiceByTopic(address))
	assert.Same(t, svc, thing.ServiceByTopic("pt:j1/mt:cmd"+address))
	assert.Nil(t, thing.ServiceByTopic("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:3"))
	assert.Nil(t, thing.ServiceByTopic("rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2"))
}
