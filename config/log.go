package config

import (
	"errors"
	"fmt"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// DefaultLogRevertTimeout is the default timeout after which a verbose log
// level (debug/trace) is automatically reverted to the previous level.
const DefaultLogRevertTimeout = 72 * time.Hour

// LogStore persists log configuration so that an armed auto-revert survives
// application restarts. Consumer applications implement this against their
// own configuration storage.
type LogStore interface {
	Level() string
	SetLevel(level string) error
	Format() string
	SetFormat(format string) error
	File() string
	SetFile(file string) error
	RevertTimeout() time.Duration
	SetRevertTimeout(timeout time.Duration) error
	PreviousLevel() log.Level
	SetPreviousLevel(level log.Level) error
	ClearPreviousLevel() error
	LevelSetAt() time.Time
	SetLevelSetAt(t time.Time) error
}

// LogManagerOption configures a LogManager.
type LogManagerOption func(*LogManager)

// WithFormatApplier registers a hook that applies a new log format at runtime.
// When nil or not provided, format changes are persisted only and take effect
// on next restart.
func WithFormatApplier(applier func(format string) error) LogManagerOption {
	return func(m *LogManager) { m.formatApplier = applier }
}

// WithOutputApplier registers a hook that applies a new log output file at
// runtime. When nil or not provided, file changes are persisted only and take
// effect on next restart.
func WithOutputApplier(applier func(file string) error) LogManagerOption {
	return func(m *LogManager) { m.outputApplier = applier }
}

// LogManager coordinates dynamic log configuration. When the log level is
// lowered to debug or trace, it stores the timestamp and arms a timer that
// reverts to the previously active level after the configured timeout. Any
// change to a level of Info or higher cancels the pending revert.
type LogManager struct {
	store         LogStore
	formatApplier func(string) error
	outputApplier func(string) error

	lock  sync.Mutex
	timer *time.Timer
}

// NewLogManager creates a log manager backed by the given store.
func NewLogManager(store LogStore, opts ...LogManagerOption) *LogManager {
	m := &LogManager{store: store}
	for _, opt := range opts {
		opt(m)
	}

	return m
}

// Start resumes any pending auto-revert from persisted state. It must be
// called after the logger has been initialised and the store has been
// loaded. Safe to call when no revert is pending.
func (m *LogManager) Start() {
	m.lock.Lock()
	defer m.lock.Unlock()

	setAt := m.store.LevelSetAt()
	if setAt.IsZero() {
		return
	}

	currentLevel, err := log.ParseLevel(m.store.Level())
	if err != nil || currentLevel < log.DebugLevel {
		_ = m.clearRevertStateLocked()
		return
	}

	timeout := m.revertTimeout()
	elapsed := time.Since(setAt)
	if elapsed >= timeout {
		m.revertLog("startup: timeout already elapsed")

		return
	}

	m.startTimer(timeout - elapsed)
}

// Level returns the currently persisted log level.
func (m *LogManager) Level() string {
	return m.store.Level()
}

// SetLevel applies and persists the given log level and manages the
// auto-revert timer. Calling with a level of Info or higher cancels any
// pending revert; calling with debug or trace arms (or re-arms) it.
func (m *LogManager) SetLevel(level string) error {
	lvl, err := log.ParseLevel(level)
	if err != nil {
		return fmt.Errorf("log: invalid level %q: %w", level, err)
	}

	m.lock.Lock()
	defer m.lock.Unlock()

	if lvl < log.DebugLevel {
		if err := m.store.SetLevel(lvl.String()); err != nil {
			return err
		}

		if err := m.clearRevertStateLocked(); err != nil {
			log.WithError(err).Warnf("[cliff] failed to clear log revert state; startup recovery will retry")
		}

		m.stopTimer()

		log.SetLevel(lvl)
		log.Infof("[cliff] Log level updated to %s", lvl)

		return nil
	}

	previous := m.store.PreviousLevel()

	if currentLvl, err := log.ParseLevel(m.store.Level()); err == nil && currentLvl < log.DebugLevel {
		previous = currentLvl
	}

	if err := m.store.SetPreviousLevel(previous); err != nil {
		return err
	}

	if err := m.store.SetLevelSetAt(time.Now()); err != nil {
		return err
	}

	if err := m.store.SetLevel(lvl.String()); err != nil {
		return err
	}

	log.SetLevel(lvl)
	log.Infof("[cliff] Log level updated to %s; auto-revert to %s after %s", lvl, previous, m.revertTimeout())

	m.stopTimer()
	m.startTimer(m.revertTimeout())

	return nil
}

// Format returns the currently persisted log format.
func (m *LogManager) Format() string {
	return m.store.Format()
}

// SetFormat applies the given log format via the format applier hook (if
// configured) and persists it on success. Persistence is skipped when the
// applier fails so a bad format is not retained across restarts.
func (m *LogManager) SetFormat(format string) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	if m.formatApplier != nil {
		if err := m.formatApplier(format); err != nil {
			return err
		}
	}

	return m.store.SetFormat(format)
}

// File returns the currently persisted log file path.
func (m *LogManager) File() string {
	return m.store.File()
}

// SetFile applies the given log file path via the output applier hook (if
// configured) and persists it on success. Persistence is skipped when the
// applier fails so a bad path is not retained across restarts.
func (m *LogManager) SetFile(file string) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	if m.outputApplier != nil {
		if err := m.outputApplier(file); err != nil {
			return err
		}
	}

	return m.store.SetFile(file)
}

// RevertTimeout returns the currently persisted revert timeout, or the
// default when no value has been persisted.
func (m *LogManager) RevertTimeout() time.Duration {
	m.lock.Lock()
	defer m.lock.Unlock()

	return m.revertTimeout()
}

// SetRevertTimeout persists a new revert timeout. When a revert is currently
// armed, the timer is rescheduled against the new timeout (firing
// immediately if the new timeout has already elapsed).
func (m *LogManager) SetRevertTimeout(timeout time.Duration) error {
	if timeout <= 0 {
		return errors.New("log: revert timeout must be positive")
	}

	m.lock.Lock()
	defer m.lock.Unlock()

	if err := m.store.SetRevertTimeout(timeout); err != nil {
		return err
	}

	if m.timer == nil {
		return nil
	}

	setAt := m.store.LevelSetAt()
	if setAt.IsZero() {
		return nil
	}

	elapsed := time.Since(setAt)
	m.stopTimer()

	if elapsed >= timeout {
		m.revertLog("revert timeout reduced below elapsed time")

		return nil
	}

	m.startTimer(timeout - elapsed)

	return nil
}

func (m *LogManager) revertTimeout() time.Duration {
	timeout := m.store.RevertTimeout()
	if timeout <= 0 {
		return DefaultLogRevertTimeout
	}

	return timeout
}

func (m *LogManager) startTimer(d time.Duration) {
	var t *time.Timer
	t = time.AfterFunc(d, func() {
		m.lock.Lock()
		defer m.lock.Unlock()
		if m.timer != t {
			return
		}
		m.revertLog("auto-revert timer fired")
	})
	m.timer = t
}

func (m *LogManager) stopTimer() {
	if m.timer != nil {
		m.timer.Stop()
		m.timer = nil
	}
}

func (m *LogManager) revertLog(reason string) {
	lvl := m.store.PreviousLevel()

	if err := m.store.SetLevel(lvl.String()); err != nil {
		log.WithError(err).Errorf("[cliff] failed to persist reverted log level, keeping revert state for restart retry")
		m.timer = nil

		return
	}

	log.SetLevel(lvl)
	log.Infof("[cliff] Log level reverted to %s (%s)", lvl, reason)

	if err := m.clearRevertStateLocked(); err != nil {
		log.WithError(err).Errorf("[cliff] failed to clear log revert state")
	}

	m.timer = nil
}

func (m *LogManager) clearRevertStateLocked() error {
	if err := m.store.ClearPreviousLevel(); err != nil {
		return err
	}

	return m.store.SetLevelSetAt(time.Time{})
}

// NewDefaultLogStore adapts a config.Default-backed persistence layer to the
// LogStore interface. The accessor must return a pointer to the embedded
// Default block; save persists any field mutation to disk. Revert state
// (previous level, level-set timestamp) is held in memory only and does not
// survive a restart.
func NewDefaultLogStore(accessor func() *Default, save func() error) LogStore {
	return &defaultLogStore{accessor: accessor, save: save, previousLevel: log.InfoLevel}
}

type defaultLogStore struct {
	accessor      func() *Default
	save          func() error
	previousLevel log.Level
	levelSetAt    time.Time
}

func (s *defaultLogStore) Level() string { return s.accessor().LogLevel }

func (s *defaultLogStore) SetLevel(level string) error {
	s.accessor().LogLevel = level

	return s.save()
}

func (s *defaultLogStore) Format() string { return s.accessor().LogFormat }

func (s *defaultLogStore) SetFormat(format string) error {
	s.accessor().LogFormat = format

	return s.save()
}

func (s *defaultLogStore) File() string { return s.accessor().LogFile }

func (s *defaultLogStore) SetFile(file string) error {
	s.accessor().LogFile = file

	return s.save()
}

func (s *defaultLogStore) RevertTimeout() time.Duration {
	raw := s.accessor().LogRevertTimeout
	if raw == "" {
		return 0
	}

	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0
	}

	return d
}

func (s *defaultLogStore) SetRevertTimeout(timeout time.Duration) error {
	s.accessor().LogRevertTimeout = timeout.String()

	return s.save()
}

func (s *defaultLogStore) PreviousLevel() log.Level { return s.previousLevel }

func (s *defaultLogStore) SetPreviousLevel(level log.Level) error {
	s.previousLevel = level

	return nil
}

func (s *defaultLogStore) ClearPreviousLevel() error {
	s.previousLevel = log.InfoLevel

	return nil
}

func (s *defaultLogStore) LevelSetAt() time.Time { return s.levelSetAt }

func (s *defaultLogStore) SetLevelSetAt(t time.Time) error {
	s.levelSetAt = t

	return nil
}
