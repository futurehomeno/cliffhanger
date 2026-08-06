package debug

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/cliffhanger/config"
)

func TestFlushInterval_DebugPastRevertDeadline_FallsBackToConfiguredInterval(t *testing.T) { //nolint:paralleltest
	saved := logrus.GetLevel()
	t.Cleanup(func() { logrus.SetLevel(saved) })
	logrus.SetLevel(logrus.DebugLevel)

	cfg := &config.Default{LogRevertAt: time.Now().Add(-time.Minute)}
	store := config.NewDefaultStore(func() *config.Default { return cfg }, func() error { return nil })

	ptr := &logManagerT{store: store}

	assert.Equal(t, defaultLogFlushInterval, ptr.flushInterval(),
		"an elapsed revert deadline must fall back to the slow interval even though the process-wide level is still debug")
}

func TestFlushInterval_DebugBeforeRevertDeadline_StaysFast(t *testing.T) { //nolint:paralleltest
	saved := logrus.GetLevel()
	t.Cleanup(func() { logrus.SetLevel(saved) })
	logrus.SetLevel(logrus.DebugLevel)

	cfg := &config.Default{LogRevertAt: time.Now().Add(time.Hour)}
	store := config.NewDefaultStore(func() *config.Default { return cfg }, func() error { return nil })

	ptr := &logManagerT{store: store}

	assert.Equal(t, debugLogFlushInterval, ptr.flushInterval())
}

func TestFlushInterval_DebugNoRevertDeadlineArmed_StaysFast(t *testing.T) { //nolint:paralleltest
	saved := logrus.GetLevel()
	t.Cleanup(func() { logrus.SetLevel(saved) })
	logrus.SetLevel(logrus.DebugLevel)

	cfg := &config.Default{} // zero LogRevertAt: not armed, e.g. a persisted level applied without SetLevel having run this boot
	store := config.NewDefaultStore(func() *config.Default { return cfg }, func() error { return nil })

	ptr := &logManagerT{store: store}

	assert.Equal(t, debugLogFlushInterval, ptr.flushInterval())
}

func TestSetLevel_FlusherRestartsOnlyWhenCadenceChanges(t *testing.T) { //nolint:paralleltest
	cases := []struct {
		name          string
		newLevel      string
		wantRestarted bool
	}{
		{"same-side change (info -> warn) does not restart", "warn", false},
		{"crossing into debug restarts", "debug", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { //nolint:paralleltest
			saved := logrus.GetLevel()
			savedOut := logrus.StandardLogger().Out
			t.Cleanup(func() {
				logrus.SetLevel(saved)
				logrus.SetOutput(savedOut)
			})

			dir := t.TempDir()
			cfg := &config.Default{LogLevel: "info", LogFormat: "text", LogFile: filepath.Join(dir, "app.log")}
			store := config.NewDefaultStore(func() *config.Default { return cfg }, func() error { return nil })

			require.NoError(t, InitializeLogger(store))
			t.Cleanup(func() { logManager.stopFlusher() })

			before := logManager.flushStop

			require.NoError(t, logManager.SetLevel(tc.newLevel))

			restarted := before != logManager.flushStop
			assert.Equal(t, tc.wantRestarted, restarted,
				"the flusher must restart exactly when the flush cadence actually changes")
		})
	}
}
