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
