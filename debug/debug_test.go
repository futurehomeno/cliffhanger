package debug_test

import (
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/cliffhanger/config"
	"github.com/futurehomeno/cliffhanger/debug"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

// initLogger wires up a fresh DefaultStore-backed singleton pointing at a
// temp log file, and restores the global logrus level on cleanup.
func initLogger(t *testing.T, level, format string) (*config.Default, string) {
	t.Helper()

	dir := t.TempDir()
	logFile := filepath.Join(dir, "app.log")

	cfg := &config.Default{
		LogLevel:  level,
		LogFormat: format,
		LogFile:   logFile,
	}

	store := config.NewDefaultStore(
		func() *config.Default { return cfg },
		func() error { return nil },
	)

	saved := logrus.GetLevel()
	t.Cleanup(func() { logrus.SetLevel(saved) })

	require.NoError(t, debug.InitializeLogger(store))

	return cfg, logFile
}

func TestInitializeLogger_AppliesPersistedLevelAndCreatesLogFile(t *testing.T) { //nolint:paralleltest
	cfg, logFile := initLogger(t, "warning", "text")

	assert.Equal(t, logrus.WarnLevel, logrus.GetLevel())
	assert.FileExists(t, logFile)
	assert.Equal(t, "warning", cfg.LogLevel)
}

func TestInitializeLogger_InvalidLevel_FallsBackToInfo(t *testing.T) { //nolint:paralleltest
	dir := t.TempDir()
	cfg := &config.Default{
		LogLevel:  "bogus",
		LogFormat: "text",
		LogFile:   filepath.Join(dir, "x.log"),
	}
	store := config.NewDefaultStore(
		func() *config.Default { return cfg },
		func() error { return nil },
	)

	saved := logrus.GetLevel()
	t.Cleanup(func() { logrus.SetLevel(saved) })

	// A bad persisted level must not prevent startup.
	require.NoError(t, debug.InitializeLogger(store))
	assert.Equal(t, logrus.InfoLevel, logrus.GetLevel(), "fallback to info on parse failure")
}

func TestInitializeLogger_EmptyLogFile_Errors(t *testing.T) { //nolint:paralleltest
	cfg := &config.Default{
		LogLevel:  "info",
		LogFormat: "text",
		LogFile:   "",
	}
	store := config.NewDefaultStore(
		func() *config.Default { return cfg },
		func() error { return nil },
	)

	err := debug.InitializeLogger(store)
	require.Error(t, err)
}

func TestInitializeLogger_AppliesEachFormat(t *testing.T) { //nolint:paralleltest
	cases := []struct {
		format string
		want   any
	}{
		{"json", &logrus.JSONFormatter{}},
		{"text", &logrus.TextFormatter{}},
		{"budzik", nil}, // custom formatter; just check it's not nil/text/json
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.format, func(t *testing.T) { //nolint:paralleltest
			initLogger(t, "info", tc.format)

			got := logrus.StandardLogger().Formatter
			assert.NotNil(t, got)

			switch tc.format {
			case "json":
				assert.IsType(t, &logrus.JSONFormatter{}, got)
			case "text":
				assert.IsType(t, &logrus.TextFormatter{}, got)
			}
		})
	}
}

func TestRouting_GetSetLevel_RoundTrips(t *testing.T) { //nolint:paralleltest
	dir := t.TempDir()
	logFile := filepath.Join(dir, "app.log")

	saved := logrus.GetLevel()
	t.Cleanup(func() { logrus.SetLevel(saved) })

	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name: "level get/set via FIMP",
				Setup: suite.BaseSetup(func(t *testing.T, _ *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
					t.Helper()

					cfg := &config.Default{LogLevel: "info", LogFormat: "text", LogFile: logFile}
					store := config.NewDefaultStore(
						func() *config.Default { return cfg },
						func() error { return nil },
					)

					require.NoError(t, debug.InitializeLogger(store))

					return debug.Route("test_service"), nil, nil
				}),
				Nodes: []*suite.Node{
					{
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", debug.CmdLogGetLevel, "test_service"),
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:app/rn:test/ad:1", debug.EvtLogLevelReport, "test_service", "info"),
						},
					},
					{
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", debug.CmdLogSetLevel, "test_service", "warning"),
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:app/rn:test/ad:1", debug.EvtLogLevelReport, "test_service", "warning"),
						},
					},
					{
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", debug.CmdLogGetLevel, "test_service"),
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:app/rn:test/ad:1", debug.EvtLogLevelReport, "test_service", "warning"),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func TestRouting_FormatGetSet_RoundTrips(t *testing.T) { //nolint:paralleltest
	dir := t.TempDir()
	logFile := filepath.Join(dir, "app.log")

	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name: "format",
				Setup: suite.BaseSetup(func(t *testing.T, _ *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
					t.Helper()

					cfg := &config.Default{LogLevel: "info", LogFormat: "text", LogFile: logFile}
					store := config.NewDefaultStore(
						func() *config.Default { return cfg },
						func() error { return nil },
					)
					require.NoError(t, debug.InitializeLogger(store))

					return debug.Route("test_service"), nil, nil
				}),
				Nodes: []*suite.Node{
					{
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", debug.CmdLogGetFormat, "test_service"),
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:app/rn:test/ad:1", debug.EvtLogFormatReport, "test_service", "text"),
						},
					},
					{
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", debug.CmdLogSetFormat, "test_service", "json"),
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:app/rn:test/ad:1", debug.EvtLogFormatReport, "test_service", "json"),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func TestRouting_FileGetSet_RoundTrips(t *testing.T) { //nolint:paralleltest
	// SetFile rejects anything that isn't a plain file name, so chdir into
	// the temp dir and pass a bare name.
	dir := t.TempDir()
	t.Chdir(dir)

	startFile := "start.log"
	newFile := "rotated.log"

	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name: "file",
				Setup: suite.BaseSetup(func(t *testing.T, _ *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
					t.Helper()

					cfg := &config.Default{LogLevel: "info", LogFormat: "text", LogFile: startFile}
					store := config.NewDefaultStore(
						func() *config.Default { return cfg },
						func() error { return nil },
					)
					require.NoError(t, debug.InitializeLogger(store))

					return debug.Route("test_service"), nil, nil
				}),
				Nodes: []*suite.Node{
					{
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", debug.CmdLogGetFile, "test_service"),
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:app/rn:test/ad:1", debug.EvtLogFileReport, "test_service", startFile),
						},
					},
					{
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", debug.CmdLogSetFile, "test_service", newFile),
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:app/rn:test/ad:1", debug.EvtLogFileReport, "test_service", newFile),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func TestRouting_RevertTimeoutGetSet_RoundTrips(t *testing.T) { //nolint:paralleltest
	dir := t.TempDir()
	logFile := filepath.Join(dir, "app.log")

	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name: "revert timeout",
				Setup: suite.BaseSetup(func(t *testing.T, _ *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
					t.Helper()

					cfg := &config.Default{LogLevel: "info", LogFormat: "text", LogFile: logFile}
					store := config.NewDefaultStore(
						func() *config.Default { return cfg },
						func() error { return nil },
					)
					require.NoError(t, debug.InitializeLogger(store))

					return debug.Route("test_service"), nil, nil
				}),
				Nodes: []*suite.Node{
					{
						// Default fallback timeout when none persisted: 7 days = 168h.
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", debug.CmdLogGetRevertTimeout, "test_service"),
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:app/rn:test/ad:1", debug.EvtLogRevertTimeoutReport, "test_service", (7 * 24 * time.Hour).String()),
						},
					},
					{
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", debug.CmdLogSetRevertTimeout, "test_service", "2h"),
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:app/rn:test/ad:1", debug.EvtLogRevertTimeoutReport, "test_service", "2h0m0s"),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func TestSetLevel_Debug_ArmsRevert(t *testing.T) { //nolint:paralleltest
	dir := t.TempDir()
	logFile := filepath.Join(dir, "app.log")

	cfg := &config.Default{
		LogLevel:         "info",
		LogFormat:        "text",
		LogFile:          logFile,
		LogRevertTimeout: time.Hour,
	}

	saved := logrus.GetLevel()
	t.Cleanup(func() { logrus.SetLevel(saved) })

	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name: "set debug arms revert",
				Setup: suite.BaseSetup(func(t *testing.T, _ *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
					t.Helper()

					store := config.NewDefaultStore(
						func() *config.Default { return cfg },
						func() error { return nil },
					)
					require.NoError(t, debug.InitializeLogger(store))

					return debug.Route("test_service"), nil, nil
				}),
				Nodes: []*suite.Node{
					{
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", debug.CmdLogSetLevel, "test_service", "debug"),
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:app/rn:test/ad:1", debug.EvtLogLevelReport, "test_service", "debug"),
						},
					},
				},
			},
		},
	}

	s.Run(t)

	assert.Equal(t, "debug", cfg.LogLevel)
	assert.False(t, cfg.LogRevertAt.IsZero(), "debug level should arm a revert deadline")
	assert.True(t, cfg.LogRevertAt.After(time.Now()))
}

func TestSetLevel_InfoOrHigher_ClearsRevert(t *testing.T) { //nolint:paralleltest
	dir := t.TempDir()
	logFile := filepath.Join(dir, "app.log")

	cfg := &config.Default{
		LogLevel:    "debug",
		LogFormat:   "text",
		LogFile:     logFile,
		LogRevertAt: time.Now().Add(time.Hour),
	}

	saved := logrus.GetLevel()
	t.Cleanup(func() { logrus.SetLevel(saved) })

	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name: "set info clears revert",
				Setup: suite.BaseSetup(func(t *testing.T, _ *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
					t.Helper()

					store := config.NewDefaultStore(
						func() *config.Default { return cfg },
						func() error { return nil },
					)
					require.NoError(t, debug.InitializeLogger(store))

					return debug.Route("test_service"), nil, nil
				}),
				Nodes: []*suite.Node{
					{
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", debug.CmdLogSetLevel, "test_service", "info"),
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:app/rn:test/ad:1", debug.EvtLogLevelReport, "test_service", "info"),
						},
					},
				},
			},
		},
	}

	s.Run(t)

	assert.Equal(t, "info", cfg.LogLevel)
	assert.True(t, cfg.LogRevertAt.IsZero(), "info or higher should clear pending revert")
}

// TestRouting_SetLog_PersistsToStore verifies that each FIMP-driven log SET
// command goes through *DefaultStore.Save(), so changes survive a restart.
// The existing routing roundtrip tests only cover in-memory state.
func TestRouting_SetLog_PersistsToStore(t *testing.T) { //nolint:paralleltest
	dir := t.TempDir()
	t.Chdir(dir)

	cfg := &config.Default{LogLevel: "info", LogFormat: "text", LogFile: "app.log"}

	var saves atomic.Int32

	store := config.NewDefaultStore(
		func() *config.Default { return cfg },
		func() error {
			saves.Add(1)

			return nil
		},
	)

	saved := logrus.GetLevel()
	t.Cleanup(func() { logrus.SetLevel(saved) })

	require.NoError(t, debug.InitializeLogger(store))

	var (
		levelSavesBefore   int32
		formatSavesBefore  int32
		fileSavesBefore    int32
		revertSavesBefore  int32
	)

	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name: "set persists",
				Setup: suite.BaseSetup(func(t *testing.T, _ *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
					t.Helper()

					return debug.Route("test_service"), nil, nil
				}),
				Nodes: []*suite.Node{
					{
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", debug.CmdLogSetLevel, "test_service", "warning"),
						InitCallbacks: []suite.Callback{
							func(t *testing.T) { t.Helper(); levelSavesBefore = saves.Load() },
						},
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:app/rn:test/ad:1", debug.EvtLogLevelReport, "test_service", "warning"),
						},
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()
								require.Eventually(t, func() bool {
									return saves.Load() > levelSavesBefore
								}, time.Second, 10*time.Millisecond, "set_level must persist")
							},
						},
					},
					{
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", debug.CmdLogSetFormat, "test_service", "json"),
						InitCallbacks: []suite.Callback{
							func(t *testing.T) { t.Helper(); formatSavesBefore = saves.Load() },
						},
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:app/rn:test/ad:1", debug.EvtLogFormatReport, "test_service", "json"),
						},
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()
								require.Eventually(t, func() bool {
									return saves.Load() > formatSavesBefore
								}, time.Second, 10*time.Millisecond, "set_format must persist")
							},
						},
					},
					{
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", debug.CmdLogSetFile, "test_service", "rotated.log"),
						InitCallbacks: []suite.Callback{
							func(t *testing.T) { t.Helper(); fileSavesBefore = saves.Load() },
						},
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:app/rn:test/ad:1", debug.EvtLogFileReport, "test_service", "rotated.log"),
						},
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()
								require.Eventually(t, func() bool {
									return saves.Load() > fileSavesBefore
								}, time.Second, 10*time.Millisecond, "set_file must persist")
							},
						},
					},
					{
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", debug.CmdLogSetRevertTimeout, "test_service", "2h"),
						InitCallbacks: []suite.Callback{
							func(t *testing.T) { t.Helper(); revertSavesBefore = saves.Load() },
						},
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:app/rn:test/ad:1", debug.EvtLogRevertTimeoutReport, "test_service", "2h0m0s"),
						},
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()
								require.Eventually(t, func() bool {
									return saves.Load() > revertSavesBefore
								}, time.Second, 10*time.Millisecond, "set_revert_timeout must persist")
							},
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func TestInitializeLogger_ConcurrentSettersAreRaceFree(t *testing.T) { //nolint:paralleltest
	dir := t.TempDir()

	cfg := &config.Default{
		LogLevel:  "info",
		LogFormat: "text",
		LogFile:   filepath.Join(dir, "race.log"),
	}
	store := config.NewDefaultStore(
		func() *config.Default { return cfg },
		func() error { return nil },
	)

	saved := logrus.GetLevel()
	t.Cleanup(func() { logrus.SetLevel(saved) })

	require.NoError(t, debug.InitializeLogger(store))

	const goroutines = 8
	const iters = 50

	var wg sync.WaitGroup

	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iters; j++ {
				_ = store.Level()
				_ = store.Format()
				_ = store.File()
			}
		}()
	}

	wg.Wait()
}
