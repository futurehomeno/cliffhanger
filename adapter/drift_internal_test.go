package adapter

import (
	"encoding/json"
	"sync"
	"testing"

	"github.com/futurehomeno/fimpgo/fimptype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTopologyChecksum_NilReportDoesNotPanic(t *testing.T) {
	t.Parallel()

	sum, err := topologyChecksum(nil)
	if err != nil || sum != 0 {
		t.Fatalf("nil report: got (%d, %v), want (0, nil)", sum, err)
	}
}

// groupsInfo drives the topology checksum: changing the groups makes a seed drift from the live
// thing, which is what triggers a rebuild.
type groupsInfo struct {
	Groups []string `json:"groups"`
}

// groupsFactory builds a thing whose inclusion report carries the groups held in its state.
type groupsFactory struct{}

func (groupsFactory) Create(_ Adapter, publisher Publisher, ts ThingState) (Thing, error) {
	var info groupsInfo

	if err := ts.Info(&info); err != nil {
		return nil, err
	}

	return NewThing(publisher, ts, &ThingConfig{
		InclusionReport: &fimptype.ThingInclusionReport{Address: ts.Address(), Groups: info.Groups},
		Connector:       &recordingConnector{},
	}), nil
}

// TestRebuildChangedThing_RecreatesWhenDestroyReportsError pins that a destroy reporting an error
// does not abort the rebuild. Whether the destroy failed past the removal or kept the record
// after a failed state write, the recreate overwrites it; aborting left the device excluded or
// ghosted, taking the persisted state the rebuild had just captured down with it.
func TestRebuildChangedThing_RecreatesWhenDestroyReportsError(t *testing.T) {
	t.Parallel()

	s, err := NewState(t.TempDir())
	require.NoError(t, err)

	ts, err := s.add(&thingStateModel{ID: "B", Address: "2", Info: json.RawMessage(`{"groups":["g1"]}`)})
	require.NoError(t, err)

	require.NoError(t, ts.SetState(map[string]string{"keep": "me"}))

	live, err := groupsFactory{}.Create(nil, stubPublisher{}, ts)
	require.NoError(t, err)

	a := &adapter{
		publisher: stubPublisher{},
		state:     failingRemoveState{State: s},
		factory:   groupsFactory{},
		things:    map[string]Thing{"2": live},
		lock:      &sync.RWMutex{},
	}

	// The seed gains a group, so the topology drifts and the thing is rebuilt.
	err = a.RebuildChangedThings(ThingSeeds{{ID: "B", Info: groupsInfo{Groups: []string{"g1", "g2"}}}})

	assert.NoError(t, err, "a destroy that only reports an error must not fail the rebuild")

	rebuilt := a.things["2"]
	require.NotNil(t, rebuilt, "the thing must be recreated, not left excluded")
	assert.Equal(t, []string{"g1", "g2"}, rebuilt.InclusionReport().Groups, "the rebuilt thing must carry the fresh topology")

	newTS := a.state.byID("B")
	require.NotNil(t, newTS, "the state record must be back")

	var kept map[string]string

	require.NoError(t, newTS.State(&kept))
	assert.Equal(t, map[string]string{"keep": "me"}, kept, "persisted state must survive the rebuild")
}
