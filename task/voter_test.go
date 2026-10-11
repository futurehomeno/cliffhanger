package task_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/lifecycle"
	"github.com/futurehomeno/cliffhanger/task"
)

func TestAuthStateVoters(t *testing.T) {
	t.Parallel()

	tt := []struct {
		name                 string
		authState            lifecycle.State
		wantAuthenticated    bool
		wantNotAuthenticated bool
	}{
		{name: "authenticated", authState: lifecycle.AuthStateAuthenticated, wantAuthenticated: true},
		{name: "not authenticated", authState: lifecycle.AuthStateNotAuthenticated, wantNotAuthenticated: true},
		{name: "in progress", authState: lifecycle.AuthStateInProgress},
		{name: "error", authState: lifecycle.AuthStateError},
		{name: "lost", authState: lifecycle.AuthStateLost},
		{name: "not available", authState: lifecycle.AuthStateNA},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			l := lifecycle.New(nil)
			l.SetAuthState(tc.authState)

			assert.Equal(t, tc.wantAuthenticated, task.WhenAppIsAuthenticated(l).Vote())
			assert.Equal(t, tc.wantNotAuthenticated, task.WhenAppIsNotAuthenticated(l).Vote())
		})
	}
}

func TestAppHealthAndConnectionVoters(t *testing.T) {
	t.Parallel()

	healthVoters := map[lifecycle.State]func(*lifecycle.Lifecycle) task.Voter{
		lifecycle.AppHealthStarting:      task.WhenAppIsStarting,
		lifecycle.AppHealthNotConfigured: task.WhenAppIsNotConfigured,
		lifecycle.AppHealthRunning:       task.WhenAppIsRunning,
		lifecycle.AppHealthTerminate:     task.WhenAppIsTerminating,
		lifecycle.AppHealthStartupError:  task.WhenAppEncounteredStartupError,
		lifecycle.AppHealthError:         task.WhenAppEncounteredError,
	}

	connVoters := map[lifecycle.State]func(*lifecycle.Lifecycle) task.Voter{
		lifecycle.ConnStateConnected:    task.WhenAppIsConnected,
		lifecycle.ConnStateDisconnected: task.WhenAppIsDisconnected,
	}

	l := lifecycle.New(nil)
	l.SetConnState(lifecycle.ConnStateConnecting)
	l.SetAuthState(lifecycle.AuthStateAuthenticated)

	for state := range healthVoters {
		l.SetAppHealth(state, nil)

		for voterState, voter := range healthVoters {
			assert.Equal(t, voterState == state, voter(l).Vote(), "health=%s voter=%s", state, voterState)
		}

		for _, voter := range connVoters {
			assert.False(t, voter(l).Vote(), "connection voters ignore app health")
		}
	}

	for _, state := range []lifecycle.State{lifecycle.ConnStateConnected, lifecycle.ConnStateDisconnected, lifecycle.ConnStateNA} {
		l.SetConnState(state)

		for voterState, voter := range connVoters {
			assert.Equal(t, voterState == state, voter(l).Vote(), "conn=%s voter=%s", state, voterState)
		}
	}
}

func TestWhenNot_FollowsCurrentState(t *testing.T) {
	t.Parallel()

	l := lifecycle.New(nil)
	notConnected := task.WhenNot(task.WhenAppIsConnected(l))

	l.SetConnState(lifecycle.ConnStateDisconnected)
	assert.True(t, notConnected.Vote())

	l.SetConnState(lifecycle.ConnStateConnected)
	assert.False(t, notConnected.Vote())
}
