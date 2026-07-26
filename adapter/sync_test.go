package adapter_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/futurehomeno/cliffhanger/adapter"
	mockedadapter "github.com/futurehomeno/cliffhanger/test/mocks/adapter"
)

type device struct{ ID, Name string }

var (
	errFetch   = errors.New("fetch failed")
	errFactory = errors.New("factory failed")
)

func seedDevice(d device) *adapter.ThingSeed {
	return &adapter.ThingSeed{ID: d.ID, Info: d.Name}
}

// fetchOK returns a fetch callback succeeding with the provided devices.
func fetchOK[T any](items []T) func() ([]T, error) {
	return func() ([]T, error) { return items, nil }
}

// fetchErr returns a fetch callback failing with the provided error.
func fetchErr[T any](err error) func() ([]T, error) {
	return func() ([]T, error) { return nil, err }
}

func TestSyncThings_ReconcilesSelectedDevices(t *testing.T) {
	t.Parallel()

	available := []device{{"1", "a"}, {"2", "b"}, {"3", "c"}}

	a := mockedadapter.NewAdapter(t)
	a.On("EnsureThings", mock.MatchedBy(func(seeds adapter.ThingSeeds) bool {
		return len(seeds) == 2 && seeds.Contains("1") && seeds.Contains("3")
	})).Return(nil)

	seeds, err := adapter.SyncThings(a, fetchOK(available), []string{"1", "3"}, seedDevice)

	assert.NoError(t, err)
	assert.Len(t, seeds, 2)
	// Both selected devices are in the fetched list, so nothing looks vanished.
	a.AssertNotCalled(t, "DestroyThingByAddress", mock.Anything)
}

func TestSyncThings_FetchFailureMutatesNothing(t *testing.T) {
	t.Parallel()

	// No expectations are registered: the mock fails the test on any adapter call, which is
	// what proves a failed fetch mutates nothing.
	a := mockedadapter.NewAdapter(t)

	seeds, err := adapter.SyncThings(a, fetchErr[device](errFetch), []string{"1"}, seedDevice)

	assert.ErrorIs(t, err, errFetch)
	assert.Nil(t, seeds)
	a.AssertNotCalled(t, "EnsureThings", mock.Anything)
	a.AssertNotCalled(t, "DestroyThingByAddress", mock.Anything)
}

func TestSyncThings_ExcludesVanishedDevice(t *testing.T) {
	t.Parallel()

	// Device "2" is selected but the successful fetch no longer lists it, and the adapter owns
	// no thing for it: a node left behind by a legacy adapter, which still needs an exclusion.
	a := mockedadapter.NewAdapter(t)
	a.On("ExchangeID", "2").Return("", false)
	a.On("ExchangeAddress", "2").Return("", false)
	a.On("DestroyThingByAddress", "2").Return(nil)
	a.On("EnsureThings", mock.Anything).Return(nil)

	_, err := adapter.SyncThings(a, fetchOK([]device{{"1", "a"}}), []string{"1", "2"}, seedDevice)

	assert.NoError(t, err)
}

func TestSyncThings_DoesNotExcludeOwnedOrForeignAddress(t *testing.T) {
	t.Parallel()

	// "2" is owned by the adapter, so EnsureThings destroys it and announces its real address.
	// "3" is not an owned ID but happens to be another thing's address, so excluding it would
	// kill an unrelated thing.
	a := mockedadapter.NewAdapter(t)
	a.On("ExchangeID", "2").Return("20", true)
	a.On("ExchangeID", "3").Return("", false)
	a.On("ExchangeAddress", "3").Return("other", true)
	a.On("EnsureThings", mock.Anything).Return(nil)

	_, err := adapter.SyncThings(a, fetchOK([]device{{"1", "a"}}), []string{"1", "2", "3"}, seedDevice)

	assert.NoError(t, err)
	a.AssertNotCalled(t, "DestroyThingByAddress", mock.Anything)
}

func TestSyncThings_NilSelectionIncludesEverything(t *testing.T) {
	t.Parallel()

	available := []device{{"1", "a"}, {"2", "b"}}

	a := mockedadapter.NewAdapter(t)
	a.On("EnsureThings", mock.MatchedBy(func(seeds adapter.ThingSeeds) bool {
		return len(seeds) == 2 && seeds.Contains("1") && seeds.Contains("2")
	})).Return(nil)

	seeds, err := adapter.SyncThings(a, fetchOK(available), nil, seedDevice)

	assert.NoError(t, err)
	assert.Len(t, seeds, 2)
	// Nothing is selected by ID, so the vanished-device pass has nothing to iterate.
	a.AssertNotCalled(t, "ExchangeID", mock.Anything)
	a.AssertNotCalled(t, "DestroyThingByAddress", mock.Anything)
}

func TestSyncThings_AggregatesFailures(t *testing.T) {
	t.Parallel()

	errExclude, errEnsure := errors.New("exclude"), errors.New("ensure")

	a := mockedadapter.NewAdapter(t)
	a.On("ExchangeID", "2").Return("", false)
	a.On("ExchangeAddress", "2").Return("", false)
	a.On("DestroyThingByAddress", "2").Return(errExclude)
	a.On("EnsureThings", mock.Anything).Return(errEnsure)

	seeds, err := adapter.SyncThings(a, fetchOK([]device{{"1", "a"}}), []string{"1", "2"}, seedDevice)

	// A failure of one device neither hides the other nor withholds the seeds a follow-up
	// drift rebuild needs.
	assert.ErrorIs(t, err, errExclude)
	assert.ErrorIs(t, err, errEnsure)
	assert.Len(t, seeds, 1)
}
