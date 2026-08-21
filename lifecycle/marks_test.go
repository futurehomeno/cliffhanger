package lifecycle_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/lifecycle"
)

func TestLifecycle_Marks(t *testing.T) {
	t.Parallel()

	lc := lifecycle.New(nil)

	lc.MarkRunning()
	assert.Equal(t, lifecycle.AppHealthRunning, lc.AppHealth())
	assert.Equal(t, lifecycle.ConfigStateConfigured, lc.ConfigState())
	assert.Equal(t, lifecycle.ConnStateConnected, lc.ConnectionState())
	assert.Equal(t, lifecycle.AuthStateAuthenticated, lc.AuthState())

	lc.MarkNotConfigured()
	assert.Equal(t, lifecycle.AppHealthNotConfigured, lc.AppHealth())
	assert.Equal(t, lifecycle.ConfigStateNotConfigured, lc.ConfigState())
	assert.Equal(t, lifecycle.ConnStateDisconnected, lc.ConnectionState())
	assert.Equal(t, lifecycle.AuthStateNotAuthenticated, lc.AuthState())
}

// TestLifecycle_Marks_EmitSingleEvent pins that MarkRunning/MarkNotConfigured bundle their four
// state changes into one auth-state emit, like SetConnAndAuthState. Four separate emits would
// reopen the exact drop risk (emitStateChangeEvent's non-blocking send onto a possibly-full
// subscriber buffer) that SetConnAndAuthState exists to avoid for a conn+auth transition.
func TestLifecycle_Marks_EmitSingleEvent(t *testing.T) {
	t.Parallel()

	lc := lifecycle.New(nil)
	lc.MarkNotConfigured()

	ch := lc.Subscribe("test", 5)

	lc.MarkRunning()

	event := <-ch
	assert.Equal(t, lifecycle.StateTypeAuthState, event.Type, "a single auth event carries the transition")
	assert.Equal(t, lifecycle.AuthStateAuthenticated, event.State)
	assert.Empty(t, ch, "no separate app-health/config/connection event competes for the buffer")

	lc.MarkNotConfigured()

	event = <-ch
	assert.Equal(t, lifecycle.StateTypeAuthState, event.Type)
	assert.Equal(t, lifecycle.AuthStateNotAuthenticated, event.State)
	assert.Empty(t, ch)
}
