package adapter

import (
	"encoding/json"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// countingPublisher is needed because NewAdapter derives its publisher from a live MQTT transport.
type countingPublisher struct {
	stubPublisher

	adapterMessages atomic.Int32
}

func (p *countingPublisher) PublishAdapterMessage(*fimpgo.FimpMessage) error {
	p.adapterMessages.Add(1)

	return nil
}

type factoryFunc func(Adapter, Publisher, ThingState) (Thing, error)

func (f factoryFunc) Create(a Adapter, p Publisher, ts ThingState) (Thing, error) { return f(a, p, ts) }

func connectorThing(ts ThingState, c Connector, groups ...string) Thing {
	return NewThing(stubPublisher{}, ts, &ThingConfig{
		InclusionReport: &fimptype.ThingInclusionReport{Address: ts.Address(), Groups: groups},
		Connector:       c,
	})
}

func TestCreateThing_DisconnectsTheDisplacedInstance(t *testing.T) {
	t.Parallel()

	s, err := NewState(t.TempDir())
	require.NoError(t, err)

	ts, err := s.add(&thingStateModel{ID: "A", Address: "1"})
	require.NoError(t, err)

	oldConnector := &recordingConnector{}
	newConnector := &recordingConnector{}

	a := &adapter{
		publisher: stubPublisher{},
		state:     s,
		factory: factoryFunc(func(_ Adapter, _ Publisher, ts ThingState) (Thing, error) {
			return connectorThing(ts, newConnector), nil
		}),
		things: map[string]Thing{"1": connectorThing(ts, oldConnector)},
		lock:   &sync.RWMutex{},
	}

	require.NoError(t, a.CreateThing(&ThingSeed{ID: "A", CustomAddress: "1"}))

	assert.True(t, oldConnector.wasDisconnected(), "the displaced instance must release its connector")
	assert.False(t, newConnector.wasDisconnected())
	assert.Len(t, a.things, 1)
}

func TestDestroyThingByID_UnknownIDPublishesNothing(t *testing.T) {
	t.Parallel()

	s, err := NewState(t.TempDir())
	require.NoError(t, err)

	p := &countingPublisher{}
	a := &adapter{publisher: p, state: s, things: map[string]Thing{}, lock: &sync.RWMutex{}}

	require.NoError(t, a.DestroyThingByID("missing"))
	assert.Zero(t, p.adapterMessages.Load(), "an unknown ID has no address to exclude")

	require.NoError(t, a.DestroyThingByAddress("missing"))
	assert.Equal(t, int32(1), p.adapterMessages.Load(), "an unknown address is still excluded")
}

func TestRebuildChangedThings_ProspectiveBuildIsInMemory(t *testing.T) {
	t.Parallel()

	s, err := NewState(t.TempDir())
	require.NoError(t, err)

	ts, err := s.add(&thingStateModel{
		ID:      "A",
		Address: "7",
		Info:    json.RawMessage(`{"groups":["stored"]}`),
		State:   json.RawMessage(`{"persisted":true}`),
	})
	require.NoError(t, err)

	var (
		prospective ThingState
		seenGroups  []string
		creates     int
	)

	a := &adapter{
		publisher: stubPublisher{},
		state:     s,
		factory: factoryFunc(func(_ Adapter, _ Publisher, ts ThingState) (Thing, error) {
			creates++
			prospective = ts

			var info groupsInfo
			if err := ts.Info(&info); err != nil {
				return nil, err
			}

			seenGroups = info.Groups

			return connectorThing(ts, &recordingConnector{}, info.Groups...), nil
		}),
		things:      map[string]Thing{"7": connectorThing(ts, &recordingConnector{}, "g1")},
		lock:        &sync.RWMutex{},
		initialized: true,
	}

	require.NoError(t, a.RebuildChangedThings(ThingSeeds{{ID: "A", Info: groupsInfo{Groups: []string{"g1"}}}}))
	require.Equal(t, 1, creates, "an unchanged topology must only be built prospectively")
	assert.Equal(t, []string{"g1"}, seenGroups, "the prospective thing is built from the seed's info, not the stored one")

	assert.Equal(t, "A", prospective.ID())
	assert.Equal(t, "7", prospective.Address(), "the prospective thing is built at the live address")
	assert.Zero(t, prospective.InclusionChecksum())

	var saved map[string]bool

	require.NoError(t, prospective.State(&saved))
	assert.Nil(t, saved, "the prospective state carries no persisted state")
	require.NoError(t, prospective.SetState(map[string]bool{"overwritten": true}))
	require.NoError(t, prospective.SetInclusionChecksum(42))

	require.NoError(t, s.byID("A").State(&saved))
	assert.Equal(t, map[string]bool{"persisted": true}, saved, "prospective writes must not reach the record")
	assert.Zero(t, s.byID("A").InclusionChecksum())

	var noInfo groupsInfo

	require.NoError(t, newSeedState(&ThingSeed{ID: "B"}, "8").Info(&noInfo), "a seed without info is a no-op")
	assert.Empty(t, noInfo.Groups)
}
