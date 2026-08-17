package adapter

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/cliffhanger/storage"
)

// TestThingState_ErrorBranchesDoNotDeadlock pins that the error paths of the thing state do not
// re-enter the state lock. They format the thing ID into their message, and reading it through
// the ID() accessor would take a read lock on the non-reentrant mutex the writer already holds,
// freezing the whole adapter permanently.
func TestThingState_ErrorBranchesDoNotDeadlock(t *testing.T) {
	t.Parallel()

	s, err := NewState(t.TempDir())
	require.NoError(t, err)

	ts, err := s.add(&thingStateModel{ID: "1", Address: "1"})
	require.NoError(t, err)

	done := make(chan error, 1)

	go func() {
		// A channel cannot be marshalled, so this takes the write lock and then fails.
		done <- ts.SetState(make(chan int))
	}()

	select {
	case err := <-done:
		assert.Error(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("SetState deadlocked on its error branch")
	}
}

type failingSaveStorage struct {
	storage.Storage[*adapterStateModel]

	failSave bool
	saves    int
}

func (f *failingSaveStorage) Save() error {
	if f.failSave {
		return errors.New("disk full")
	}

	f.saves++

	return f.Storage.Save()
}

// TestState_AddAssignsAddressInASingleWrite pins that creating a thing costs one state write.
// Acquiring the address separately doubled it, and every write is a full rewrite plus fsync of
// adapter.json on flash storage.
func TestState_AddAssignsAddressInASingleWrite(t *testing.T) {
	t.Parallel()

	failing := &failingSaveStorage{Storage: storage.NewState(&adapterStateModel{}, t.TempDir(), "adapter.json")}
	s := &state{Storage: failing}

	first, err := s.add(&thingStateModel{ID: "1"})
	require.NoError(t, err)
	assert.Equal(t, "1", first.Address())

	second, err := s.add(&thingStateModel{ID: "2"})
	require.NoError(t, err)
	assert.Equal(t, "2", second.Address())
	assert.Equal(t, 2, failing.saves)

	custom, err := s.add(&thingStateModel{ID: "3", Address: "99"})
	require.NoError(t, err)
	assert.Equal(t, "99", custom.Address(), "a model that carries an address must keep it")

	failing.failSave = true

	_, err = s.add(&thingStateModel{ID: "4"})
	require.Error(t, err)

	failing.failSave = false

	next, err := s.add(&thingStateModel{ID: "5"})
	require.NoError(t, err)
	assert.Equal(t, "3", next.Address(), "a failed write must not consume an address")
}

// TestThingState_SetInclusionChecksumSkipsUnchangedWrites pins that re-stamping the same checksum
// does not persist. SendInclusionReport(true) bypasses the checksum short-circuit, so every forced
// report rewrote the whole adapter state file with an identical value.
func TestThingState_SetInclusionChecksumSkipsUnchangedWrites(t *testing.T) {
	t.Parallel()

	failing := &failingSaveStorage{Storage: storage.NewState(&adapterStateModel{}, t.TempDir(), "adapter.json")}
	s := &state{Storage: failing}

	ts, err := s.add(&thingStateModel{ID: "1"})
	require.NoError(t, err)

	require.NoError(t, ts.SetInclusionChecksum(42))
	saves := failing.saves

	require.NoError(t, ts.SetInclusionChecksum(42))
	assert.Equal(t, saves, failing.saves, "an unchanged checksum must not be persisted again")

	require.NoError(t, ts.SetInclusionChecksum(43))
	assert.Equal(t, saves+1, failing.saves)
}

// TestThingState_InclusionChecksumRetriesAfterFailedSave pins that a failed write leaves the
// checksum unchanged in memory. The skip above is keyed on that value, so keeping the new one
// would turn every retry of the same checksum into a no-op and the disk would never catch up.
func TestThingState_InclusionChecksumRetriesAfterFailedSave(t *testing.T) {
	t.Parallel()

	failing := &failingSaveStorage{Storage: storage.NewState(&adapterStateModel{}, t.TempDir(), "adapter.json")}
	s := &state{Storage: failing}

	ts, err := s.add(&thingStateModel{ID: "1"})
	require.NoError(t, err)

	failing.failSave = true
	require.Error(t, ts.SetInclusionChecksum(42))
	assert.Zero(t, ts.InclusionChecksum(), "a failed write must not keep the new checksum in memory")

	failing.failSave = false
	saves := failing.saves

	require.NoError(t, ts.SetInclusionChecksum(42), "the retry must persist the same checksum")
	assert.Equal(t, saves+1, failing.saves)
	assert.Equal(t, uint32(42), ts.InclusionChecksum())
}

// TestState_RemoveRestoresEntryOnSaveFailure pins that a failed persist of a removal restores the
// in-memory record. The disk still holds it, so dropping it from memory only would let the next
// sync skip the retry and leave the thing to resurrect after a restart.
func TestState_RemoveRestoresEntryOnSaveFailure(t *testing.T) {
	t.Parallel()

	underlying := storage.NewState(&adapterStateModel{}, t.TempDir(), "adapter.json")
	failing := &failingSaveStorage{Storage: underlying}
	s := &state{Storage: failing}

	_, err := s.add(&thingStateModel{ID: "1", Address: "2"})
	require.NoError(t, err)

	failing.failSave = true

	err = s.remove("1")
	require.Error(t, err)
	require.NotNil(t, s.byID("1"), "record must be restored after a failed save")

	failing.failSave = false

	require.NoError(t, s.remove("1"))
	assert.Nil(t, s.byID("1"))
}

// TestState_AddRestoresPreviousEntryOnSaveFailure pins the mirror of the above for add: a failed
// persist of an overwrite puts the old record back. createThing rolls a failed heal back to the
// record it captured, but a write failing inside add happens before that rollback is armed, so
// only add itself can keep the ghost's address from being replaced by one no disk ever saw.
func TestState_AddRestoresPreviousEntryOnSaveFailure(t *testing.T) {
	t.Parallel()

	underlying := storage.NewState(&adapterStateModel{}, t.TempDir(), "adapter.json")
	failing := &failingSaveStorage{Storage: underlying}
	s := &state{Storage: failing}

	_, err := s.add(&thingStateModel{ID: "1", Address: "2"})
	require.NoError(t, err)

	failing.failSave = true

	_, err = s.add(&thingStateModel{ID: "1", Address: "3"})
	require.Error(t, err)
	assert.Equal(t, "2", s.modelByID("1").Address, "the overwritten record must be restored after a failed save")

	_, err = s.add(&thingStateModel{ID: "9", Address: "9"})
	require.Error(t, err)
	assert.Nil(t, s.modelByID("9"), "a record the disk never got must not linger in memory")
}
