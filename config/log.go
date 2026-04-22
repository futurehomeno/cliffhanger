package config

import (
	"fmt"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// DefaultLogRevertTimeout is the timeout after which a verbose log level
// (debug/trace) is automatically reverted to the previous level. The revert
// is evaluated on application start; a long-running process therefore keeps
// verbose logging until its next restart at or after the deadline.
const DefaultLogRevertTimeout = 7 * 24 * time.Hour

// LogStore persists log configuration so an armed auto-revert survives
// application restarts.
type LogStore interface {
	Level() string
	SetLevel(level string) error
	Format() string
	SetFormat(format string) error
	File() string
	SetFile(file string) error
	RevertTimeout() time.Duration
	SetRevertTimeout(d time.Duration) error
	RevertAt() time.Time
	SetRevertAt(t time.Time) error
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
// lowered to debug or trace, it persists an absolute revert deadline. The
// deadline is evaluated on Start and the level reverts if it has elapsed.
type LogManager struct {
	store         LogStore
	formatApplier func(string) error
	outputApplier func(string) error

	lock sync.Mutex
}

// NewLogManager creates a log manager backed by the given store.
func NewLogManager(store LogStore, opts ...LogManagerOption) *LogManager {
	m := &LogManager{store: store}
	for _, opt := range opts {
		opt(m)
	}

	return m
}

// Start evaluates a persisted revert deadline and reverts the level when it
// has elapsed. Safe to call when no revert is pending.
func (m *LogManager) Start() {
	m.lock.Lock()
	defer m.lock.Unlock()

	revertAt := m.store.RevertAt()
	if revertAt.IsZero() {
		return
	}

	currentLevel, err := log.ParseLevel(m.store.Level())
	if err != nil || currentLevel < log.DebugLevel {
		_ = m.clearRevertStateLocked()
		return
	}

	if time.Now().Before(revertAt) {
		return
	}

	m.revertLog("startup: revert deadline elapsed")
}

// Level returns the currently persisted log level.
func (m *LogManager) Level() string {
	return m.store.Level()
}

// SetLevel applies and persists the given log level. When the level is
// lowered to debug or trace, a revert deadline is persisted using the
// configured revert timeout (defaulting to DefaultLogRevertTimeout); any
// level of info or higher clears it.
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

		log.SetLevel(lvl)
		log.Infof("[cliff] Log level updated to %s", lvl)

		return nil
	}

	timeout := m.revertTimeoutLocked()

	if err := m.store.SetRevertAt(time.Now().Add(timeout)); err != nil {
		return err
	}

	if err := m.store.SetLevel(lvl.String()); err != nil {
		return err
	}

	log.SetLevel(lvl)
	log.Infof("[cliff] Log level updated to %s; will revert to info on next startup after %s", lvl, timeout)

	return nil
}

// RevertTimeout returns the configured revert timeout, falling back to
// DefaultLogRevertTimeout when none has been set.
func (m *LogManager) RevertTimeout() time.Duration {
	m.lock.Lock()
	defer m.lock.Unlock()

	return m.revertTimeoutLocked()
}

// SetRevertTimeout persists the revert timeout. If a revert is currently
// armed, the deadline is recalculated from now.
func (m *LogManager) SetRevertTimeout(d time.Duration) error {
	if d <= 0 {
		return fmt.Errorf("log: revert timeout must be positive")
	}

	m.lock.Lock()
	defer m.lock.Unlock()

	if err := m.store.SetRevertTimeout(d); err != nil {
		return err
	}

	if !m.store.RevertAt().IsZero() {
		return m.store.SetRevertAt(time.Now().Add(d))
	}

	return nil
}

func (m *LogManager) revertTimeoutLocked() time.Duration {
	if t := m.store.RevertTimeout(); t > 0 {
		return t
	}

	return DefaultLogRevertTimeout
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

func (m *LogManager) revertLog(reason string) {
	if err := m.store.SetLevel(log.InfoLevel.String()); err != nil {
		log.WithError(err).Errorf("[cliff] failed to persist reverted log level, keeping revert state for next startup retry")

		return
	}

	log.SetLevel(log.InfoLevel)
	log.Infof("[cliff] Log level reverted to %s (%s)", log.InfoLevel, reason)

	if err := m.clearRevertStateLocked(); err != nil {
		log.WithError(err).Errorf("[cliff] failed to clear log revert state")
	}
}

func (m *LogManager) clearRevertStateLocked() error {
	return m.store.SetRevertAt(time.Time{})
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

func (s *defaultLogStore) RevertTimeout() time.Duration { return s.accessor().LogRevertTimeout }

func (s *defaultLogStore) SetRevertTimeout(d time.Duration) error {
	s.accessor().LogRevertTimeout = d

	return s.save()
}

func (s *defaultLogStore) RevertAt() time.Time { return s.accessor().LogRevertAt }

func (s *defaultLogStore) SetRevertAt(t time.Time) error {
	s.accessor().LogRevertAt = t

	return s.save()
}
