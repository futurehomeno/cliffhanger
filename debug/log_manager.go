package debug

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/futurehomeno/cliffhanger/debug/formatters"
)

var (
	logManager *logManagerT
)

type logManagerT struct {
	logOutput *lumberjack.Logger
	store     storeIf
	lock      sync.Mutex
}

// defaultLogRevertTimeout is the timeout after which a verbose log level
// (debug/trace) is automatically reverted to the previous level (info/warn)
const defaultLogRevertTimeout = 7 * 24 * time.Hour

type storeIf interface {
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

func InitializeLogger(store storeIf) error {
	logManager = &logManagerT{
		store: store,
	}

	setLogFormat(store.Format())

	// SetLevel returns an error when the persisted level is unparseable,
	// but it has already fallen back to InfoLevel by then. Swallow it so a
	// bad log_level value in config.json does not prevent startup.
	_ = logManager.SetLevel()

	return logManager.setLogOutput(store.File())
}

func setLogFormat(logFormat string) {
	switch logFormat {
	case "json":
		logrus.SetFormatter(&logrus.JSONFormatter{TimestampFormat: "2006-01-02 15:04:05.999"})
	case "budzik":
		logrus.SetFormatter(formatters.NewBudzikFormatter())
	default:
		logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true, ForceColors: true, TimestampFormat: "2006-01-02T15:04:05.999"})
	}
}

func (ptr *logManagerT) setLogOutput(logFile string) error {
	if logFile == "" {
		return fmt.Errorf("log file not set")
	}

	ptr.lock.Lock()
	defer ptr.lock.Unlock()

	if err := os.MkdirAll(filepath.Dir(logFile), 0755); err != nil { //nolint:gosec
		return fmt.Errorf("create log dir=%s err: %w", filepath.Dir(logFile), err)
	}

	if f, err := os.OpenFile(logFile, os.O_RDONLY|os.O_CREATE, 0644); err != nil { //nolint:gosec
		return fmt.Errorf("open log file=%s err: %w", logFile, err)
	} else if cerr := f.Close(); cerr != nil {
		logrus.Errorf("close err: %v", cerr)
	}

	newOutput := &lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    5, // MiB
		MaxBackups: 4,
	}

	previous := ptr.logOutput
	ptr.logOutput = newOutput
	logrus.SetOutput(newOutput)

	if previous != nil && previous != newOutput {
		if err := previous.Close(); err != nil {
			logrus.Errorf("close previous log output err: %v", err)
		}
	}

	return nil
}

func (ptr *logManagerT) Level() string {
	return ptr.store.Level()
}

func (ptr *logManagerT) SetLevel() error {
	logLevel, err := logrus.ParseLevel(ptr.store.Level())
	if err != nil {
		logrus.SetLevel(logrus.InfoLevel)
		logrus.Warnf("[cliff] Invalid log level %q, falling back to %s", ptr.store.Level(), logrus.InfoLevel)

		return fmt.Errorf("log: invalid level %q: %w", ptr.store.Level(), err)
	}

	ptr.lock.Lock()
	defer ptr.lock.Unlock()

	if logLevel < logrus.DebugLevel {
		if err := ptr.store.SetLevel(logLevel.String()); err != nil {
			return err
		}

		if err := ptr.clearRevertStateLocked(); err != nil {
			logrus.WithError(err).Warnf("[cliff] failed to clear log revert state; startup recovery will retry")
		}

		logrus.SetLevel(logLevel)
		logrus.Infof("[cliff] Log level updated to %s", logLevel)

		return nil
	}

	timeout := ptr.store.RevertTimeout()
	if timeout <= 0 {
		timeout = defaultLogRevertTimeout
	}

	if err := ptr.store.SetRevertAt(time.Now().Add(timeout)); err != nil {
		return err
	}

	if err := ptr.store.SetLevel(logLevel.String()); err != nil {
		return err
	}

	logrus.SetLevel(logLevel)
	logrus.Infof("[cliff] Log level updated to %s; will revert to info on next startup after %s", logLevel, timeout)

	return nil
}

// SetRevertTimeout persists the revert timeout. If a revert is currently
// armed, the deadline is recalculated from now.
func (ptr *logManagerT) SetRevertTimeout(d time.Duration) error {
	if d <= 0 {
		return fmt.Errorf("log: revert timeout must be positive")
	}

	ptr.lock.Lock()
	defer ptr.lock.Unlock()

	if err := ptr.store.SetRevertTimeout(d); err != nil {
		return err
	}

	if !ptr.store.RevertAt().IsZero() {
		return ptr.store.SetRevertAt(time.Now().Add(d))
	}

	return nil
}

// Format returns the currently persisted log format.
func (ptr *logManagerT) Format() string {
	return ptr.store.Format()
}

// SetFormat applies the given log format via the format applier hook (if
// configured) and persists it on success. Persistence is skipped when the
// applier fails so a bad format is not retained across restarts.
func (ptr *logManagerT) SetFormat(format string) error {
	ptr.lock.Lock()
	defer ptr.lock.Unlock()

	setLogFormat(format)

	return ptr.store.SetFormat(format)
}

// File returns the currently persisted log file path.
func (ptr *logManagerT) File() string {
	return ptr.store.File()
}

// SetFile applies the given log file path via the output applier hook (if
// configured) and persists it on success. Persistence is skipped when the
// applier fails so a bad path is not retained across restarts.
func (ptr *logManagerT) SetFile(file string) error {
	if err := ptr.setLogOutput(file); err != nil {
		return err
	}

	ptr.lock.Lock()
	defer ptr.lock.Unlock()

	return ptr.store.SetFile(file)
}

func (ptr *logManagerT) clearRevertStateLocked() error {
	return ptr.store.SetRevertAt(time.Time{})
}
