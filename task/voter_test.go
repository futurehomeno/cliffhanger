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
