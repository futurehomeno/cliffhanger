package telemetry

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/cliffhanger/config"
	"github.com/futurehomeno/cliffhanger/telemetry/types"
)

// TestResumeValidityWindow_StopsThePreviousTimer pins that resuming the window twice - which is now
// the normal path, since Start() resumes it as well as the constructor - does not leave the first
// timer armed. The stale one is harmless when it fires, but it stays in the runtime timer heap for
// the whole validity period.
func TestResumeValidityWindow_StopsThePreviousTimer(t *testing.T) { //nolint:paralleltest
	model := &config.Default{}
	store := config.NewDefaultStore(func() *config.Default { return model }, func() error { return nil })

	require.NoError(t, store.SetTelemetry(&types.TelemetryConfig{
		Enabled:   true,
		EnabledAt: time.Now(),
		Validity:  time.Hour,
	}))

	tel := &telemetryT{store: store}

	require.NoError(t, tel.resumeValidityWindow())
	first := tel.timer
	require.NotNil(t, first, "an enabled window arms a timer")

	require.NoError(t, tel.resumeValidityWindow())
	require.NotNil(t, tel.timer)
	assert.NotSame(t, first, tel.timer, "the second resume arms its own timer")

	// Stop reports false once the timer is already stopped or fired. It cannot have fired here:
	// the validity is an hour out.
	assert.False(t, first.Stop(), "the timer from the first resume must have been stopped")

	tel.stopValidityTimer()
}
