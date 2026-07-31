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
}

func (f *failingSaveStorage) Save() error {
	if f.failSave {
		return errors.New("disk full")
	}

	return f.Storage.Save()
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
