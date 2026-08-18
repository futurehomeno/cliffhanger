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

// TestSetFile_AlwaysPersistsAnAbsolutePath pins that a relative log path inherited from an older
// install does not keep the new file relative to the working directory: the persisted value would
// otherwise move with the daemon's cwd on every restart, which is the bug SetFile exists to close.
func TestSetFile_AlwaysPersistsAnAbsolutePath(t *testing.T) { //nolint:paralleltest
	savedOut := logrus.StandardLogger().Out
	t.Cleanup(func() { logrus.SetOutput(savedOut) })

	for _, current := range []string{"app.log", "logs/app.log", ""} {
		dir := t.TempDir()
		t.Chdir(dir)

		cfg := &config.Default{LogLevel: "info", LogFormat: "text", LogFile: current}
		store := config.NewDefaultStore(
			func() *config.Default { return cfg },
			func() error { return nil },
		)

		m := &logManagerT{store: store}

		require.NoError(t, m.SetFile("rotated.log"), "current=%q", current)
		assert.True(t, filepath.IsAbs(cfg.LogFile),
			"a current path of %q must still persist an absolute path, got %q", current, cfg.LogFile)
		assert.Equal(t, "rotated.log", filepath.Base(cfg.LogFile))

		m.stopFlusher()
	}
}

// TestInitializeLogger_FailedFirstInitKeepsManager pins that a first initialization that cannot
// open its log file still leaves a manager behind: Route panics on a nil one, so an application
// that logs the error and carries on would crash loop instead of running without file logging.
func TestInitializeLogger_FailedFirstInitKeepsManager(t *testing.T) { //nolint:paralleltest
	saved := logManager
	savedLevel := logrus.GetLevel()
	t.Cleanup(func() {
		logManager = saved
		logrus.SetLevel(savedLevel)
	})

	logManager = nil

	cfg := &config.Default{LogLevel: "info", LogFormat: "text"} // no LogFile: setLogOutput fails
	store := config.NewDefaultStore(func() *config.Default { return cfg }, func() error { return nil })

	require.Error(t, InitializeLogger(store))
	require.NotNil(t, logManager, "a failed first initialization must not leave the manager nil")

	assert.NotPanics(t, func() { Route("test") })
}

// TestSetFile_AfterFailedFirstInitStartsFlusher pins that recovering through set_file also starts
// the periodic flusher the failed initialization never reached: only error-level writes flush on
// their own, so everything below it would otherwise sit in the buffer until it fills.
func TestSetFile_AfterFailedFirstInitStartsFlusher(t *testing.T) { //nolint:paralleltest
	saved := logManager
	savedLevel := logrus.GetLevel()
	savedOut := logrus.StandardLogger().Out
	t.Cleanup(func() {
		logManager = saved
		logrus.SetLevel(savedLevel)
		logrus.SetOutput(savedOut)
	})

	logManager = nil

	cfg := &config.Default{LogLevel: "info", LogFormat: "text"} // no LogFile: setLogOutput fails
	store := config.NewDefaultStore(func() *config.Default { return cfg }, func() error { return nil })

	require.Error(t, InitializeLogger(store))
	require.Nil(t, logManager.flushStop, "a failed initialization must not have started a flusher")

	require.NoError(t, logManager.SetFile(filepath.Join(t.TempDir(), "app.log")))
	t.Cleanup(func() { logManager.stopFlusher() })

	assert.NotNil(t, logManager.flushStop, "recovering through SetFile must start the periodic flusher")
}
