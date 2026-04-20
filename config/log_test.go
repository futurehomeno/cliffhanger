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
	lock          sync.Mutex
	level         string
	format        string
	file          string
	revertTimeout time.Duration
	previousLevel string
	levelSetAt    time.Time
	setLevelErr   error
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

func (s *memLogStore) RevertTimeout() time.Duration {
	s.lock.Lock()
	defer s.lock.Unlock()

	return s.revertTimeout
}

func (s *memLogStore) SetRevertTimeout(d time.Duration) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.revertTimeout = d

	return nil
}

func (s *memLogStore) PreviousLevel() string {
	s.lock.Lock()
	defer s.lock.Unlock()

	return s.previousLevel
}

func (s *memLogStore) SetPreviousLevel(level string) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.previousLevel = level

	return nil
}

func (s *memLogStore) LevelSetAt() time.Time {
	s.lock.Lock()
	defer s.lock.Unlock()

	return s.levelSetAt
}

func (s *memLogStore) SetLevelSetAt(t time.Time) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.levelSetAt = t

	return nil
}

// restoreGlobalLogLevel snapshots the global logrus level and restores it
// when the test ends, so tests that mutate the level don't leak into others.
func restoreGlobalLogLevel(t *testing.T) {
	t.Helper()

	saved := log.GetLevel()
	t.Cleanup(func() { log.SetLevel(saved) })
}

// waitUntil polls until cond returns true or the deadline expires.
func waitUntil(t *testing.T, timeout time.Duration, cond func() bool) bool {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}

		time.Sleep(5 * time.Millisecond)
	}

	return cond()
}

func TestLogManager_SetLevel_InfoOrHigher_HasNoRevertState(t *testing.T) { //nolint:paralleltest
	restoreGlobalLogLevel(t)

	store := &memLogStore{level: "info"}
	mgr := config.NewLogManager(store)

	assert.NoError(t, mgr.SetLevel("warning"))
	assert.Equal(t, "warning", store.Level())
	assert.Equal(t, "", store.PreviousLevel())
	assert.True(t, store.LevelSetAt().IsZero())
	assert.Equal(t, log.WarnLevel, log.GetLevel())
}

func TestLogManager_SetLevel_Debug_ArmsRevertAndSnapshotsPrevious(t *testing.T) { //nolint:paralleltest
	restoreGlobalLogLevel(t)

	store := &memLogStore{level: "warning"}
	store.revertTimeout = 50 * time.Millisecond
	mgr := config.NewLogManager(store)

	before := time.Now()
	assert.NoError(t, mgr.SetLevel("debug"))

	assert.Equal(t, "debug", store.Level())
	assert.Equal(t, "warning", store.PreviousLevel())
	assert.False(t, store.LevelSetAt().IsZero())
	assert.True(t, !store.LevelSetAt().Before(before))
	assert.Equal(t, log.DebugLevel, log.GetLevel())

	// Timer should fire and revert to the previous level.
	reverted := waitUntil(t, 500*time.Millisecond, func() bool {
		return log.GetLevel() == log.WarnLevel && store.Level() == "warning" && store.PreviousLevel() == ""
	})
	assert.True(t, reverted, "expected auto-revert to restore warn level")
	assert.True(t, store.LevelSetAt().IsZero())
}

func TestLogManager_SetLevel_TraceAfterDebug_KeepsOriginalPrevious(t *testing.T) { //nolint:paralleltest
	restoreGlobalLogLevel(t)

	store := &memLogStore{level: "error", revertTimeout: time.Hour}
	mgr := config.NewLogManager(store)

	assert.NoError(t, mgr.SetLevel("debug"))
	assert.Equal(t, "error", store.PreviousLevel())
	firstSetAt := store.LevelSetAt()

	time.Sleep(5 * time.Millisecond)

	assert.NoError(t, mgr.SetLevel("trace"))
	assert.Equal(t, "trace", store.Level())
	assert.Equal(t, "error", store.PreviousLevel(), "previous must remain the pre-verbose level")
	assert.True(t, store.LevelSetAt().After(firstSetAt), "setAt should refresh on re-arm")
}

func TestLogManager_SetLevel_InfoAfterDebug_CancelsRevert(t *testing.T) { //nolint:paralleltest
	restoreGlobalLogLevel(t)

	store := &memLogStore{level: "warning", revertTimeout: time.Hour}
	mgr := config.NewLogManager(store)

	assert.NoError(t, mgr.SetLevel("debug"))
	assert.Equal(t, "warning", store.PreviousLevel())

	assert.NoError(t, mgr.SetLevel("info"))
	assert.Equal(t, "info", store.Level())
	assert.Equal(t, "", store.PreviousLevel())
	assert.True(t, store.LevelSetAt().IsZero())
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
	assert.Equal(t, "", store.PreviousLevel())
}

func TestLogManager_Start_PendingButCurrentLevelIsInfo_ClearsStaleState(t *testing.T) { //nolint:paralleltest
	restoreGlobalLogLevel(t)

	// Persisted state says "was debug, revert to warn" but the current level
	// is info. Start should treat this as stale and clear the revert state.
	store := &memLogStore{
		level:         "info",
		previousLevel: "warning",
		levelSetAt:    time.Now(),
	}
	mgr := config.NewLogManager(store)
	mgr.Start()

	assert.Equal(t, "", store.PreviousLevel())
	assert.True(t, store.LevelSetAt().IsZero())
}

func TestLogManager_Start_ElapsedGreaterThanTimeout_RevertsImmediately(t *testing.T) { //nolint:paralleltest
	restoreGlobalLogLevel(t)
	log.SetLevel(log.DebugLevel)

	store := &memLogStore{
		level:         "debug",
		previousLevel: "warning",
		levelSetAt:    time.Now().Add(-2 * time.Hour),
		revertTimeout: time.Hour,
	}
	mgr := config.NewLogManager(store)
	mgr.Start()

	assert.Equal(t, "warning", store.Level())
	assert.Equal(t, "", store.PreviousLevel())
	assert.True(t, store.LevelSetAt().IsZero())
	assert.Equal(t, log.WarnLevel, log.GetLevel())
}

func TestLogManager_Start_ElapsedLessThanTimeout_ArmsRemainingTimer(t *testing.T) { //nolint:paralleltest
	restoreGlobalLogLevel(t)
	log.SetLevel(log.DebugLevel)

	store := &memLogStore{
		level:         "debug",
		previousLevel: "error",
		levelSetAt:    time.Now().Add(-40 * time.Millisecond),
		revertTimeout: 100 * time.Millisecond,
	}
	mgr := config.NewLogManager(store)
	mgr.Start()

	// Immediately after Start, state must still be pending.
	assert.Equal(t, "debug", store.Level())
	assert.Equal(t, "error", store.PreviousLevel())

	// Timer should fire roughly 60ms later.
	reverted := waitUntil(t, 500*time.Millisecond, func() bool {
		return log.GetLevel() == log.ErrorLevel && store.Level() == "error"
	})
	assert.True(t, reverted, "expected auto-revert from Start to fire")
}

func TestLogManager_RevertTimeout_DefaultWhenUnset(t *testing.T) { //nolint:paralleltest
	store := &memLogStore{}
	mgr := config.NewLogManager(store)

	assert.Equal(t, config.DefaultLogRevertTimeout, mgr.RevertTimeout())
}

func TestLogManager_SetRevertTimeout_RejectsNonPositive(t *testing.T) { //nolint:paralleltest
	store := &memLogStore{}
	mgr := config.NewLogManager(store)

	assert.Error(t, mgr.SetRevertTimeout(0))
	assert.Error(t, mgr.SetRevertTimeout(-time.Second))
}

func TestLogManager_SetRevertTimeout_AlreadyElapsedTriggersImmediateRevert(t *testing.T) { //nolint:paralleltest
	restoreGlobalLogLevel(t)

	store := &memLogStore{level: "warning", revertTimeout: time.Hour}
	mgr := config.NewLogManager(store)

	assert.NoError(t, mgr.SetLevel("debug"))

	// Backdate the setAt so the new timeout appears to have already elapsed.
	assert.NoError(t, store.SetLevelSetAt(time.Now().Add(-time.Minute)))

	assert.NoError(t, mgr.SetRevertTimeout(time.Second))

	assert.Equal(t, "warning", store.Level())
	assert.Equal(t, "", store.PreviousLevel())
	assert.Equal(t, log.WarnLevel, log.GetLevel())
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
	assert.NoError(t, store.SetRevertTimeout(90*time.Minute))
	assert.NoError(t, store.SetPreviousLevel("info"))

	now := time.Date(2026, 4, 17, 10, 0, 0, 0, time.UTC)
	assert.NoError(t, store.SetLevelSetAt(now))

	assert.Equal(t, "debug", store.Level())
	assert.Equal(t, "json", store.Format())
	assert.Equal(t, "/var/log/a.log", store.File())
	assert.Equal(t, 90*time.Minute, store.RevertTimeout())
	assert.Equal(t, "info", store.PreviousLevel())
	assert.True(t, store.LevelSetAt().Equal(now))

	assert.NoError(t, store.SetLevelSetAt(time.Time{}))
	assert.True(t, store.LevelSetAt().IsZero())

	assert.Equal(t, int32(7), atomic.LoadInt32(&saveCalls))
}

// TestRouteCmdLog_Managed exercises the new FIMP routing functions end-to-end
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
						level:         "info",
						format:        "text",
						file:          "/tmp/app.log",
						revertTimeout: 72 * time.Hour,
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
					{
						Name:    "get revert timeout",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.log.get_revert_timeout", "test_service"),
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.log.revert_timeout_report", "test_service", "72h0m0s"),
						},
					},
					{
						Name:    "set revert timeout",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.log.set_revert_timeout", "test_service", "24h"),
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.log.revert_timeout_report", "test_service", "24h0m0s"),
						},
					},
					{
						Name:    "set revert timeout with invalid duration returns error",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.log.set_revert_timeout", "test_service", "not-a-duration"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:app/rn:test/ad:1", "test_service"),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}
