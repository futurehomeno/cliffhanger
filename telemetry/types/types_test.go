package types_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/cliffhanger/telemetry/types"
)

func TestTelemetryConfig_CloneIsDeep(t *testing.T) {
	t.Parallel()

	orig := types.TelemetryConfig{Enabled: true, Suppressed: &types.SuppressedEntry{Domains: []string{"auth"}, Events: []string{"x"}}}

	clone := orig.Clone()
	require.NotNil(t, clone.Suppressed)
	assert.NotSame(t, orig.Suppressed, clone.Suppressed)

	clone.Suppressed.Domains[0] = "changed"
	clone.Suppressed.Events = append(clone.Suppressed.Events, "y")

	assert.Equal(t, []string{"auth"}, orig.Suppressed.Domains)
	assert.Equal(t, []string{"x"}, orig.Suppressed.Events)
	assert.Nil(t, types.TelemetryConfig{}.Clone().Suppressed)
}
