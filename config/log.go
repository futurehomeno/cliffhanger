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
	PreviousLevel() string
	SetPreviousLevel(level string) error
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

	mu    sync.Mutex
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
	m.mu.Lock()
	defer m.mu.Unlock()

	prev := m.store.PreviousLevel()
	setAt := m.store.LevelSetAt()
	if prev == "" || setAt.IsZero() {
		return
	}

	currentLevel, err := log.ParseLevel(m.store.Level())
	if err != nil || currentLevel < log.DebugLevel {
		_ = m.clearRevertStateLocked()
		return
	}

	timeout := m.revertTimeoutLocked()
	elapsed := time.Since(setAt)
	if elapsed >= timeout {
		m.revertLocked("startup: timeout already elapsed")

		return
	}

	m.armTimerLocked(timeout - elapsed)
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

	m.mu.Lock()
	defer m.mu.Unlock()

	if lvl < log.DebugLevel {
		m.cancelTimerLocked()

		if err := m.store.SetLevel(lvl.String()); err != nil {
			return err
		}

		if err := m.clearRevertStateLocked(); err != nil {
			return err
		}

		log.SetLevel(lvl)
		log.Infof("[cliff] Log level updated to %s", lvl)

		return nil
	}

	previous := m.store.PreviousLevel()

	currentStr := m.store.Level()
	currentLvl, currentErr := log.ParseLevel(currentStr)

	switch {
	case currentErr == nil && currentLvl < log.DebugLevel:
		previous = currentStr
	case previous == "":
		previous = log.InfoLevel.String()
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
	log.Infof("[cliff] Log level updated to %s; auto-revert to %s after %s", lvl, previous, m.revertTimeoutLocked())

	m.cancelTimerLocked()
	m.armTimerLocked(m.revertTimeoutLocked())

	return nil
}

// Format returns the currently persisted log format.
func (m *LogManager) Format() string {
	return m.store.Format()
}

// SetFormat persists the given log format and applies it via the format
// applier hook if one is configured.
func (m *LogManager) SetFormat(format string) error {
	if err := m.store.SetFormat(format); err != nil {
		return err
	}

	if m.formatApplier != nil {
		return m.formatApplier(format)
	}

	return nil
}

// File returns the currently persisted log file path.
func (m *LogManager) File() string {
	return m.store.File()
}

// SetFile persists the given log file path and applies it via the output
// applier hook if one is configured.
func (m *LogManager) SetFile(file string) error {
	if err := m.store.SetFile(file); err != nil {
		return err
	}

	if m.outputApplier != nil {
		return m.outputApplier(file)
	}

	return nil
}

// RevertTimeout returns the currently persisted revert timeout, or the
// default when no value has been persisted.
func (m *LogManager) RevertTimeout() time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.revertTimeoutLocked()
}

// SetRevertTimeout persists a new revert timeout. When a revert is currently
// armed, the timer is rescheduled against the new timeout (firing
// immediately if the new timeout has already elapsed).
func (m *LogManager) SetRevertTimeout(timeout time.Duration) error {
	if timeout <= 0 {
		return errors.New("log: revert timeout must be positive")
	}

	if err := m.store.SetRevertTimeout(timeout); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.timer == nil {
		return nil
	}

	setAt := m.store.LevelSetAt()
	if setAt.IsZero() {
		return nil
	}

	elapsed := time.Since(setAt)
	m.cancelTimerLocked()

	if elapsed >= timeout {
		m.revertLocked("revert timeout reduced below elapsed time")

		return nil
	}

	m.armTimerLocked(timeout - elapsed)

	return nil
}

func (m *LogManager) revertTimeoutLocked() time.Duration {
	timeout := m.store.RevertTimeout()
	if timeout <= 0 {
		return DefaultLogRevertTimeout
	}

	return timeout
}

func (m *LogManager) armTimerLocked(d time.Duration) {
	m.timer = time.AfterFunc(d, func() {
		m.mu.Lock()
		defer m.mu.Unlock()
		m.revertLocked("auto-revert timer fired")
	})
}

func (m *LogManager) cancelTimerLocked() {
	if m.timer != nil {
		m.timer.Stop()
		m.timer = nil
	}
}

func (m *LogManager) revertLocked(reason string) {
	previous := m.store.PreviousLevel()
	if previous == "" {
		previous = log.InfoLevel.String()
	}

	lvl, err := log.ParseLevel(previous)
	if err != nil {
		lvl = log.InfoLevel
	}

	if err := m.store.SetLevel(lvl.String()); err != nil {
		log.WithError(err).Errorf("[cliff] failed to persist reverted log level")
	}

	log.SetLevel(lvl)
	log.Infof("[cliff] Log level reverted to %s (%s)", lvl, reason)

	if err := m.clearRevertStateLocked(); err != nil {
		log.WithError(err).Errorf("[cliff] failed to clear log revert state")
	}

	m.timer = nil
}

func (m *LogManager) clearRevertStateLocked() error {
	if err := m.store.SetPreviousLevel(""); err != nil {
		return err
	}

	return m.store.SetLevelSetAt(time.Time{})
}

// NewDefaultLogStore adapts a config.Default-backed persistence layer to the
// LogStore interface. The accessor must return a pointer to the embedded
// Default block; save persists any field mutation to disk.
func NewDefaultLogStore(accessor func() *Default, save func() error) LogStore {
	return &defaultLogStore{accessor: accessor, save: save}
}

type defaultLogStore struct {
	accessor func() *Default
	save     func() error
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

func (s *defaultLogStore) PreviousLevel() string { return s.accessor().LogPreviousLevel }

func (s *defaultLogStore) SetPreviousLevel(level string) error {
	s.accessor().LogPreviousLevel = level

	return s.save()
}

func (s *defaultLogStore) LevelSetAt() time.Time {
	raw := s.accessor().LogLevelSetAt
	if raw == "" {
		return time.Time{}
	}

	t, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return time.Time{}
	}

	return t
}

func (s *defaultLogStore) SetLevelSetAt(t time.Time) error {
	if t.IsZero() {
		s.accessor().LogLevelSetAt = ""
	} else {
		s.accessor().LogLevelSetAt = t.Format(time.RFC3339Nano)
	}

	return s.save()
}
