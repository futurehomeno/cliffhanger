package adapter

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
