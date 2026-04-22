package config_test

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/futurehomeno/fimpgo"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/config"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

// memLogStore is an in-memory LogStore for tests.
type memLogStore struct {
	lock        sync.Mutex
	level       string
	format      string
	file        string
	revertAt    time.Time
	setLevelErr error
}

func (s *memLogStore) Level() string {
	s.lock.Lock()
	defer s.lock.Unlock()

	return s.level
}

func (s *memLogStore) SetLevel(level string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.setLevelErr != nil {
		return s.setLevelErr
	}

	s.level = level

	return nil
}

func (s *memLogStore) Format() string {
	s.lock.Lock()
	defer s.lock.Unlock()

	return s.format
}

func (s *memLogStore) SetFormat(format string) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.format = format

	return nil
}

func (s *memLogStore) File() string {
	s.lock.Lock()
	defer s.lock.Unlock()

	return s.file
}

func (s *memLogStore) SetFile(file string) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.file = file

	return nil
}

func (s *memLogStore) RevertAt() time.Time {
	s.lock.Lock()
	defer s.lock.Unlock()

	return s.revertAt
}

func (s *memLogStore) SetRevertAt(t time.Time) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.revertAt = t

	return nil
}

// restoreGlobalLogLevel snapshots the global logrus level and restores it
// when the test ends, so tests that mutate the level don't leak into others.
func restoreGlobalLogLevel(t *testing.T) {
	t.Helper()

	saved := log.GetLevel()
	t.Cleanup(func() { log.SetLevel(saved) })
}

func TestLogManager_SetLevel_InfoOrHigher_HasNoRevertState(t *testing.T) { //nolint:paralleltest
	restoreGlobalLogLevel(t)

	store := &memLogStore{level: "info"}
	mgr := config.NewLogManager(store)

	assert.NoError(t, mgr.SetLevel("warning"))
	assert.Equal(t, "warning", store.Level())
	assert.True(t, store.RevertAt().IsZero())
	assert.Equal(t, log.WarnLevel, log.GetLevel())
}

func TestLogManager_SetLevel_Debug_ArmsRevert(t *testing.T) { //nolint:paralleltest
	restoreGlobalLogLevel(t)

	store := &memLogStore{level: "warning"}
	mgr := config.NewLogManager(store)

	before := time.Now()
	assert.NoError(t, mgr.SetLevel("debug"))

	assert.Equal(t, "debug", store.Level())
	assert.False(t, store.RevertAt().IsZero())
	assert.True(t, !store.RevertAt().Before(before.Add(config.DefaultLogRevertTimeout)))
	assert.Equal(t, log.DebugLevel, log.GetLevel())
}

func TestLogManager_SetLevel_TraceAfterDebug_RefreshesRevertAt(t *testing.T) { //nolint:paralleltest
	restoreGlobalLogLevel(t)

	store := &memLogStore{level: "error"}
	mgr := config.NewLogManager(store)

	assert.NoError(t, mgr.SetLevel("debug"))
	firstRevertAt := store.RevertAt()

	time.Sleep(5 * time.Millisecond)

	assert.NoError(t, mgr.SetLevel("trace"))
	assert.Equal(t, "trace", store.Level())
	assert.True(t, store.RevertAt().After(firstRevertAt), "revertAt should refresh on re-arm")
}

func TestLogManager_SetLevel_InfoAfterDebug_CancelsRevert(t *testing.T) { //nolint:paralleltest
	restoreGlobalLogLevel(t)

	store := &memLogStore{level: "warning"}
	mgr := config.NewLogManager(store)

	assert.NoError(t, mgr.SetLevel("debug"))

	assert.NoError(t, mgr.SetLevel("info"))
	assert.Equal(t, "info", store.Level())
	assert.True(t, store.RevertAt().IsZero())
	assert.Equal(t, log.InfoLevel, log.GetLevel())
}

func TestLogManager_SetLevel_InvalidLevel_Errors(t *testing.T) { //nolint:paralleltest
	restoreGlobalLogLevel(t)

	store := &memLogStore{level: "info"}
	mgr := config.NewLogManager(store)

	err := mgr.SetLevel("bogus")
	assert.Error(t, err)
	assert.Equal(t, "info", store.Level())
}

func TestLogManager_SetLevel_StoreErrorPropagates(t *testing.T) { //nolint:paralleltest
	restoreGlobalLogLevel(t)

	store := &memLogStore{level: "info", setLevelErr: errors.New("disk full")}
	mgr := config.NewLogManager(store)

	err := mgr.SetLevel("warning")
	assert.Error(t, err)
}

func TestLogManager_Start_NoPendingRevert_IsNoop(t *testing.T) { //nolint:paralleltest
	restoreGlobalLogLevel(t)
	log.SetLevel(log.InfoLevel)

	store := &memLogStore{level: "info"}
	mgr := config.NewLogManager(store)

	mgr.Start()

	assert.Equal(t, "info", store.Level())
}

func TestLogManager_Start_PendingButCurrentLevelIsInfo_ClearsStaleState(t *testing.T) { //nolint:paralleltest
	restoreGlobalLogLevel(t)

	// Persisted state says a revert is pending but the current level is info.
	// Start should treat this as stale and clear the revert state.
	store := &memLogStore{
		level:    "info",
		revertAt: time.Now().Add(time.Hour),
	}
	mgr := config.NewLogManager(store)
	mgr.Start()

	assert.True(t, store.RevertAt().IsZero())
}

func TestLogManager_Start_RevertAtElapsed_RevertsImmediately(t *testing.T) { //nolint:paralleltest
	restoreGlobalLogLevel(t)
	log.SetLevel(log.DebugLevel)

	store := &memLogStore{
		level:    "debug",
		revertAt: time.Now().Add(-time.Minute),
	}
	mgr := config.NewLogManager(store)
	mgr.Start()

	assert.Equal(t, "info", store.Level())
	assert.True(t, store.RevertAt().IsZero())
	assert.Equal(t, log.InfoLevel, log.GetLevel())
}

func TestLogManager_Start_RevertAtInFuture_LeavesLevelAlone(t *testing.T) { //nolint:paralleltest
	restoreGlobalLogLevel(t)
	log.SetLevel(log.DebugLevel)

	store := &memLogStore{
		level:    "debug",
		revertAt: time.Now().Add(time.Hour),
	}
	mgr := config.NewLogManager(store)
	mgr.Start()

	assert.Equal(t, "debug", store.Level())
	assert.False(t, store.RevertAt().IsZero())
}

func TestLogManager_SetFormat_CallsApplierAndPersists(t *testing.T) { //nolint:paralleltest
	store := &memLogStore{format: "text"}

	var gotFormat string

	mgr := config.NewLogManager(store, config.WithFormatApplier(func(f string) error {
		gotFormat = f

		return nil
	}))

	assert.NoError(t, mgr.SetFormat("json"))
	assert.Equal(t, "json", store.Format())
	assert.Equal(t, "json", gotFormat)
}

func TestLogManager_SetFormat_ApplierErrorPropagates(t *testing.T) { //nolint:paralleltest
	store := &memLogStore{}
	mgr := config.NewLogManager(store, config.WithFormatApplier(func(string) error {
		return errors.New("bad format")
	}))

	assert.Error(t, mgr.SetFormat("weird"))
	// Persistence is skipped when the applier fails so a bad format is not
	// retained across restarts.
	assert.Equal(t, "", store.Format())
}

func TestLogManager_SetFile_CallsApplierAndPersists(t *testing.T) { //nolint:paralleltest
	store := &memLogStore{}

	var called atomic.Int32

	mgr := config.NewLogManager(store, config.WithOutputApplier(func(f string) error {
		assert.Equal(t, "/tmp/new.log", f)
		called.Add(1)

		return nil
	}))

	assert.NoError(t, mgr.SetFile("/tmp/new.log"))
	assert.Equal(t, "/tmp/new.log", store.File())
	assert.Equal(t, int32(1), called.Load())
}

func TestLogManager_NoApplier_PersistsSilently(t *testing.T) { //nolint:paralleltest
	store := &memLogStore{}
	mgr := config.NewLogManager(store)

	assert.NoError(t, mgr.SetFormat("json"))
	assert.NoError(t, mgr.SetFile("/tmp/x.log"))
	assert.Equal(t, "json", store.Format())
	assert.Equal(t, "/tmp/x.log", store.File())
}

func TestNewDefaultLogStore_RoundTripsAllFields(t *testing.T) { //nolint:paralleltest
	cfg := &config.Default{}

	var saveCalls int32

	store := config.NewDefaultLogStore(
		func() *config.Default { return cfg },
		func() error { atomic.AddInt32(&saveCalls, 1); return nil },
	)

	assert.NoError(t, store.SetLevel("debug"))
	assert.NoError(t, store.SetFormat("json"))
	assert.NoError(t, store.SetFile("/var/log/a.log"))

	revertAt := time.Date(2026, 4, 17, 10, 0, 0, 0, time.UTC)
	assert.NoError(t, store.SetRevertAt(revertAt))

	assert.Equal(t, "debug", store.Level())
	assert.Equal(t, "json", store.Format())
	assert.Equal(t, "/var/log/a.log", store.File())
	assert.True(t, store.RevertAt().Equal(revertAt))

	assert.NoError(t, store.SetRevertAt(time.Time{}))
	assert.True(t, store.RevertAt().IsZero())

	assert.Equal(t, int32(5), atomic.LoadInt32(&saveCalls))
}

// TestRouteCmdLog_Managed exercises the log-related FIMP routing end-to-end
// against an in-memory LogManager.
func TestRouteCmdLog_Managed(t *testing.T) { //nolint:paralleltest
	restoreGlobalLogLevel(t)

	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name: "getters and setters via LogManager",
				Setup: suite.BaseSetup(func(t *testing.T, mqtt *fimpgo.MqttTransport) (routing []*router.Routing, tasks []*task.Task, mocks []suite.Mock) {
					t.Helper()

					store := &memLogStore{
						level:  "info",
						format: "text",
						file:   "/tmp/app.log",
					}
					mgr := config.NewLogManager(store)

					routing = config.RoutingForLogManager("test_service", mgr)

					return routing, nil, nil
				}),
				Nodes: []*suite.Node{
					{
						Name:    "get level",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.log.get_level", "test_service"),
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.log.level_report", "test_service", "info"),
						},
					},
					{
						Name:    "get format",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.log.get_format", "test_service"),
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.log.format_report", "test_service", "text"),
						},
					},
					{
						Name:    "set format",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.log.set_format", "test_service", "json"),
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.log.format_report", "test_service", "json"),
						},
					},
					{
						Name:    "get file",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.log.get_file", "test_service"),
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.log.file_report", "test_service", "/tmp/app.log"),
						},
					},
					{
						Name:    "set file",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.log.set_file", "test_service", "other.log"),
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.log.file_report", "test_service", "other.log"),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}
