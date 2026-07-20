package adapter_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/futurehomeno/cliffhanger/adapter"
	mockedadapter "github.com/futurehomeno/cliffhanger/test/mocks/adapter"
)

type device struct{ ID, Name string }

func seedDevice(d device) *adapter.ThingSeed {
	return &adapter.ThingSeed{ID: d.ID, Info: d.Name}
}

func TestSyncThings_ReconcilesSelectedDevices(t *testing.T) {
	t.Parallel()

	available := []device{{"1", "a"}, {"2", "b"}, {"3", "c"}}

	a := mockedadapter.NewAdapter(t)
	a.On("ThingByID", mock.Anything).Return(nil)
	a.On("EnsureThings", mock.MatchedBy(func(seeds adapter.ThingSeeds) bool {
		return len(seeds) == 2 && seeds.Contains("1") && seeds.Contains("3")
	})).Return(nil)

	assert.NoError(t, adapter.SyncThings(a, available, []string{"1", "3"}, seedDevice))
}

func TestSyncThings_GuardsAgainstIncompleteFetch(t *testing.T) {
	t.Parallel()

	// Device "2" is selected and already registered, but absent from the fetched list: a
	// partial response. SyncThings must not destroy it, so EnsureThings is never called.
	available := []device{{"1", "a"}}

	a := mockedadapter.NewAdapter(t)
	a.On("ThingByID", "1").Return(nil)
	a.On("ThingByID", "2").Return(&mockedadapter.Thing{})

	err := adapter.SyncThings(a, available, []string{"1", "2"}, seedDevice)

	assert.ErrorIs(t, err, adapter.ErrIncompleteFetch)
	a.AssertNotCalled(t, "EnsureThings", mock.Anything)
}
