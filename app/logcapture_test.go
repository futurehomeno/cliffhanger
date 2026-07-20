package app_test

import (
	"strings"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/cliffhanger/app"
)

// TestNewLogCapture_CapturesGlobalWarnings verifies the returned provider is
// wired to the global logger — the one behaviour the compiler can't enforce.
// No t.Parallel: the logrus hook stays registered process-wide.
func TestNewLogCapture_CapturesGlobalWarnings(t *testing.T) {
	provider := app.NewLogCapture()
	require.NotNil(t, provider)

	const sentinel = "logcapture wiring sentinel 4f2a"
	log.Warn(sentinel)

	entries, err := provider.ErrorsReport()
	require.NoError(t, err)

	found := false
	for _, e := range entries {
		if strings.Contains(e, sentinel) {
			found = true

			break
		}
	}

	assert.True(t, found, "global warning should be captured by the returned provider")
}
