package adapter

import (
	"errors"
	"sync"
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

	failSave   bool
	failOnSave int // 1-based save attempt that fails; 0 disables
	saves      int
}

func (f *failingSaveStorage) Save() error {
	if f.failSave {
		return errors.New("disk full")
	}

	f.saves++

	if f.failOnSave > 0 && f.saves == f.failOnSave {
		return errors.New("disk full")
	}

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

// TestState_BatchCollapsesWrites pins that a pass touching many things costs one write. Every
// record lives in the same file, so a fleet rebuild rewrote and fsynced all of them per thing.
func TestState_BatchCollapsesWrites(t *testing.T) {
	t.Parallel()

	failing := &failingSaveStorage{Storage: storage.NewState(&adapterStateModel{}, t.TempDir(), "adapter.json")}
	s := &state{Storage: failing}

	err := s.batch(func() error {
		for _, id := range []string{"1", "2", "3"} {
			ts, err := s.add(&thingStateModel{ID: id})
			require.NoError(t, err)
			require.NoError(t, ts.SetInclusionChecksum(42))
		}

		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, 1, failing.saves)

	assert.NoError(t, s.batch(func() error { return nil }))
	assert.Equal(t, 1, failing.saves, "a batch that changes nothing must not write")

	err = s.batch(func() error {
		require.NoError(t, s.remove("1"))

		return errors.New("partially applied")
	})
	require.Error(t, err)
	assert.Equal(t, 2, failing.saves, "a partially applied pass must still reach the disk")

	_, err = s.add(&thingStateModel{ID: "4"})
	require.NoError(t, err)
	assert.Equal(t, 3, failing.saves, "deferral must be lifted once the batch is over")
}

// TestState_BatchRetriesAFailedFlush pins that a failed flush is retried by the next batch. The
// pass that failed announced and registered its things, so a retry skips them and has nothing left
// to save - without carrying the dirty mark over, their checksums would never reach the disk and
// every device would be re-announced after a restart.
func TestState_BatchRetriesAFailedFlush(t *testing.T) {
	t.Parallel()

	failing := &failingSaveStorage{Storage: storage.NewState(&adapterStateModel{}, t.TempDir(), "adapter.json")}
	s := &state{Storage: failing}

	failing.failSave = true

	err := s.batch(func() error {
		_, addErr := s.add(&thingStateModel{ID: "1"})

		return addErr
	})
	require.Error(t, err)
	assert.Zero(t, failing.saves)

	failing.failSave = false

	require.NoError(t, s.batch(func() error { return nil }))
	assert.Equal(t, 1, failing.saves, "a batch that changed nothing must still retry the failed flush")

	require.NoError(t, s.batch(func() error { return nil }))
	assert.Equal(t, 1, failing.saves, "once the retry lands there is nothing left to write")
}

// TestState_SaveOutsideABatchSatisfiesTheRetry pins that a save made between a failed flush and the
// next batch clears the retry mark. It writes the whole model, so leaving the mark set would cost
// the next batch a redundant rewrite of a file that is already current.
func TestState_SaveOutsideABatchSatisfiesTheRetry(t *testing.T) {
	t.Parallel()

	failing := &failingSaveStorage{Storage: storage.NewState(&adapterStateModel{}, t.TempDir(), "adapter.json")}
	s := &state{Storage: failing}

	failing.failSave = true

	require.Error(t, s.batch(func() error {
		_, addErr := s.add(&thingStateModel{ID: "1"})

		return addErr
	}))

	failing.failSave = false

	_, err := s.add(&thingStateModel{ID: "2"})
	require.NoError(t, err)
	assert.Equal(t, 1, failing.saves)

	require.NoError(t, s.batch(func() error { return nil }))
	assert.Equal(t, 1, failing.saves, "a standalone save already put the whole model on disk")
}

// TestState_BatchFlushDoesNotRaceWithConcurrentSaves pins that the flush marshals the model under
// the same lock every mutation holds. The flush is the only save not made from inside a locked
// mutation, so without it a concurrent SetState writes to the model while it is being marshalled.
// Only meaningful under -race.
func TestState_BatchFlushDoesNotRaceWithConcurrentSaves(t *testing.T) {
	t.Parallel()

	s := &state{Storage: storage.NewState(&adapterStateModel{}, t.TempDir(), "adapter.json")}

	ts, err := s.add(&thingStateModel{ID: "1"})
	require.NoError(t, err)

	var wg sync.WaitGroup

	wg.Add(2)

	go func() {
		defer wg.Done()

		for i := range 100 {
			assert.NoError(t, ts.SetState(map[string]int{"n": i}))
		}
	}()

	go func() {
		defer wg.Done()

		for range 100 {
			assert.NoError(t, s.batch(func() error {
				_, addErr := s.add(&thingStateModel{ID: "2"})

				return addErr
			}))
		}
	}()

	wg.Wait()
}

// TestState_BatchSurvivesAPanic pins that a panicking pass still lifts the deferral and flushes.
// The task manager recovers panics and keeps the process alive, so a deferral left set would turn
// every later save into a silent no-op until restart.
func TestState_BatchSurvivesAPanic(t *testing.T) {
	t.Parallel()

	failing := &failingSaveStorage{Storage: storage.NewState(&adapterStateModel{}, t.TempDir(), "adapter.json")}
	s := &state{Storage: failing}

	require.Panics(t, func() {
		_ = s.batch(func() error {
			_, err := s.add(&thingStateModel{ID: "1"})
			require.NoError(t, err)

			panic("boom")
		})
	})

	assert.Equal(t, 1, failing.saves, "the pass applied before the panic must still reach the disk")

	_, err := s.add(&thingStateModel{ID: "2"})
	require.NoError(t, err)
	assert.Equal(t, 2, failing.saves, "a panic must not leave every later save deferred for good")
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
