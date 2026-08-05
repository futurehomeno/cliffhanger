package debug

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/futurehomeno/cliffhanger/debug/formatters"
)

var (
	logManager *logManagerT

	// urgent is set by levelFormatter for entries at error level or above and
	// consumed by bufferedWriter.Write, which logrus calls immediately after
	// Format under the same logger mutex. A level hook cannot serve here:
	// hooks fire before the entry is written, so they cannot flush the very
	// line that triggered them.
	urgent atomic.Bool
)

type logManagerT struct {
	logOutput *bufferedWriter
	store     Store
	lock      sync.Mutex
	flushStop chan struct{}
}

// defaultLogRevertTimeout is the timeout after which a verbose log level
// (debug/trace) is automatically reverted to the previous level (info/warn)
const defaultLogRevertTimeout = 7 * 24 * time.Hour

const (
	// defaultLogFlushInterval bounds how long a line may sit in RAM before it
	// reaches the log file. The hub stores logs on eMMC, where an idle adapter
	// appending one short line at a time still dirties a page plus a journal
	// block on every filesystem commit; batching trades that write
	// amplification for losing the tail of the log on power loss.
	defaultLogFlushInterval = 240 * time.Second

	// debugLogFlushInterval applies while the level is debug/trace: someone
	// enabling debug is almost always about to tail the log file, and the
	// long interval would make a running adapter look hung. Debug is also a
	// bounded, operator-armed window (log_revert_timeout reverts it to info
	// on its own), so trading some of the write-amplification protection for
	// near-real-time tailing only lasts as long as the window does.
	debugLogFlushInterval = 2 * time.Second

	logBufferSize = 64 * 1024
)

type Store interface {
	Level() string
	SetLevel(level string) error
	Format() string
	SetFormat(format string) error
	LogFile() string
	SetLogFile(file string) error
	LogRevertTimeout() time.Duration
	SetLogRevertTimeout(d time.Duration) error
	LogRevertAt() time.Time
	SetLogRevertAt(t time.Time) error
	LogFlushInterval() time.Duration
}

func InitializeLogger(store Store) error {
	if logManager != nil {
		logManager.stopFlusher()

		// Close (which flushes) rather than drop: the old logOutput may hold up to
		// logBufferSize of not-yet-written lines, and nothing else keeps a reference to it
		// once logManager is reassigned below.
		if logManager.logOutput != nil {
			if err := logManager.logOutput.Close(); err != nil {
				logrus.Errorf("[cliff] close previous log output err: %v", err)
			}
		}
	}

	logManager = &logManagerT{
		store: store,
	}

	setLogFormat(store.Format())

	// applyPersistedLevel returns an error when the persisted level is
	// unparseable, but it has already fallen back to InfoLevel by then.
	// Swallow it so a bad log_level value in config.json does not prevent
	// startup.
	_ = logManager.applyPersistedLevel()

	if err := logManager.setLogOutput(store.LogFile()); err != nil {
		return err
	}

	logManager.startFlusher()

	return nil
}

// FlushLogs writes buffered log lines to the log file. Error-level entries and
// the periodic flusher do this on their own; call it before a deliberate exit
// so the closing lines are not lost with the process.
func FlushLogs() {
	if logManager != nil {
		logManager.flush()
	}
}

func (ptr *logManagerT) flush() {
	ptr.lock.Lock()
	out := ptr.logOutput
	ptr.lock.Unlock()

	if out == nil {
		return
	}

	if err := out.Flush(); err != nil {
		logrus.Errorf("[cliff] flush log output err: %v", err)
	}
}

// flushInterval returns the interval the periodic flusher should currently
// use: short while debug/trace is active, the configured (or default) one
// otherwise.
func (ptr *logManagerT) flushInterval() time.Duration {
	if logrus.GetLevel() >= logrus.DebugLevel {
		return debugLogFlushInterval
	}

	interval := ptr.store.LogFlushInterval()
	if interval <= 0 {
		interval = defaultLogFlushInterval
	}

	return interval
}

// startFlusher and stopFlusher guard flushStop with ptr.lock, like every other field on
// logManagerT, for callers (InitializeLogger) that do not already hold it. restartFlusher is
// called from within SetLevel, which already holds ptr.lock for its whole body, so it uses the
// Locked variants directly - sync.Mutex is not reentrant, and going through startFlusher /
// stopFlusher there would deadlock on every SetLevel call.
func (ptr *logManagerT) startFlusher() {
	ptr.lock.Lock()
	defer ptr.lock.Unlock()

	ptr.startFlusherLocked()
}

func (ptr *logManagerT) startFlusherLocked() {
	interval := ptr.flushInterval()

	stop := make(chan struct{})
	ptr.flushStop = stop

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				ptr.flush()
			case <-stop:
				return
			}
		}
	}()
}

func (ptr *logManagerT) stopFlusher() {
	ptr.lock.Lock()
	defer ptr.lock.Unlock()

	ptr.stopFlusherLocked()
}

func (ptr *logManagerT) stopFlusherLocked() {
	if ptr.flushStop != nil {
		close(ptr.flushStop)
		ptr.flushStop = nil
	}
}

// restartFlusher re-picks the flush interval and restarts the ticker. Called
// whenever the log level crosses the debug/trace boundary so the switch to
// (or from) near-real-time flushing takes effect immediately rather than on
// the next tick.
func (ptr *logManagerT) restartFlusher() {
	ptr.stopFlusherLocked()
	ptr.startFlusherLocked()
}

// applyPersistedLevel applies the persisted log level at startup. Unlike
// SetLevel, it does not re-arm the revert deadline on every boot: if the
// persisted level is debug/trace and the deadline has elapsed, the level
// is reset to info and the deadline cleared; otherwise the level is
// applied and the existing deadline left untouched.
func (ptr *logManagerT) applyPersistedLevel() error {
	logLevel, err := logrus.ParseLevel(ptr.store.Level())
	if err != nil {
		logrus.SetLevel(logrus.InfoLevel)
		logrus.Warnf("[cliff] Invalid log level %q, falling back to %s", ptr.store.Level(), logrus.InfoLevel)

		return fmt.Errorf("log: invalid level %q: %w", ptr.store.Level(), err)
	}

	ptr.lock.Lock()
	defer ptr.lock.Unlock()

	if logLevel >= logrus.DebugLevel {
		revertAt := ptr.store.LogRevertAt()
		if !revertAt.IsZero() && !time.Now().Before(revertAt) {
			if err := ptr.store.SetLevel(logrus.InfoLevel.String()); err != nil {
				return err
			}

			if err := ptr.clearRevertStateLocked(); err != nil {
				logrus.WithError(err).Warnf("[cliff] failed to clear log revert state on startup")
			}

			logrus.SetLevel(logrus.InfoLevel)
			logrus.Infof("[cliff] Log level reverted to info: revert deadline elapsed")

			return nil
		}
	}

	logrus.SetLevel(logLevel)

	return nil
}

func setLogFormat(logFormat string) {
	var formatter logrus.Formatter

	switch logFormat {
	case "json":
		formatter = &logrus.JSONFormatter{TimestampFormat: "2006-01-02 15:04:05.999"}
	case "budzik":
		formatter = formatters.NewBudzikFormatter()
	default:
		formatter = &logrus.TextFormatter{FullTimestamp: true, ForceColors: true, TimestampFormat: "2006-01-02T15:04:05.999"}
	}

	logrus.SetFormatter(&levelFormatter{Formatter: formatter})
}

type levelFormatter struct {
	logrus.Formatter
}

func (f *levelFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	urgent.Store(entry.Level <= logrus.ErrorLevel)

	return f.Formatter.Format(entry)
}

func (f *levelFormatter) Unwrap() logrus.Formatter {
	return f.Formatter
}

// bufferedWriter batches log lines in RAM ahead of the rotating log file,
// flushing on error-level entries, on a full buffer, and on the manager's
// periodic tick.
type bufferedWriter struct {
	lock sync.Mutex
	buf  *bufio.Writer
	file *lumberjack.Logger
}

func newBufferedWriter(file *lumberjack.Logger) *bufferedWriter {
	return &bufferedWriter{buf: bufio.NewWriterSize(file, logBufferSize), file: file}
}

func (w *bufferedWriter) Write(p []byte) (int, error) {
	w.lock.Lock()
	defer w.lock.Unlock()

	n, err := w.buf.Write(p)
	if err != nil || !urgent.Load() {
		return n, err
	}

	return n, w.buf.Flush()
}

func (w *bufferedWriter) Flush() error {
	w.lock.Lock()
	defer w.lock.Unlock()

	return w.buf.Flush()
}

func (w *bufferedWriter) Close() error {
	w.lock.Lock()
	defer w.lock.Unlock()

	if err := w.buf.Flush(); err != nil {
		return err
	}

	return w.file.Close()
}

func (ptr *logManagerT) setLogOutput(logFile string) error {
	if logFile == "" {
		return fmt.Errorf("log file not set")
	}

	if err := os.MkdirAll(filepath.Dir(logFile), 0755); err != nil { //nolint:gosec
		return fmt.Errorf("create log dir=%s err: %w", filepath.Dir(logFile), err)
	}

	if f, err := os.OpenFile(logFile, os.O_RDONLY|os.O_CREATE, 0644); err != nil { //nolint:gosec
		return fmt.Errorf("open log file=%s err: %w", logFile, err)
	} else if cerr := f.Close(); cerr != nil {
		logrus.Errorf("close err: %v", cerr)
	}

	newOutput := newBufferedWriter(&lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    5, // MiB
		MaxBackups: 4,
	})

	previous := ptr.logOutput
	ptr.logOutput = newOutput
	logrus.SetOutput(newOutput)

	if previous != nil {
		if err := previous.Close(); err != nil {
			logrus.Errorf("close previous log output err: %v", err)
		}
	}

	return nil
}

func (ptr *logManagerT) Level() string {
	return ptr.store.Level()
}

// SetLevel validates and applies the requested log level, persisting it
// alongside the revert deadline. For debug/trace, the deadline is
// written before the level so a partial failure leaves the system on
// the previous (less verbose) level rather than stranded on a verbose
// level with no deadline.
func (ptr *logManagerT) SetLevel(level string) error {
	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		return fmt.Errorf("log: invalid level %q: %w", level, err)
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
		ptr.restartFlusher()
		logrus.Infof("[cliff] Log level updated to %s", logLevel)

		return nil
	}

	timeout := ptr.store.LogRevertTimeout()
	if timeout <= 0 {
		timeout = defaultLogRevertTimeout
	}

	if err := ptr.store.SetLogRevertAt(time.Now().Add(timeout)); err != nil {
		return err
	}

	if err := ptr.store.SetLevel(logLevel.String()); err != nil {
		return err
	}

	logrus.SetLevel(logLevel)
	ptr.restartFlusher()
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

	if err := ptr.store.SetLogRevertTimeout(d); err != nil {
		return err
	}

	if !ptr.store.LogRevertAt().IsZero() {
		return ptr.store.SetLogRevertAt(time.Now().Add(d))
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
	return ptr.store.LogFile()
}

// SetFile applies the given log file path via the output applier hook (if
// configured) and persists it on success. Persistence is skipped when the
// applier fails so a bad path is not retained across restarts.
func (ptr *logManagerT) SetFile(file string) error {
	ptr.lock.Lock()
	defer ptr.lock.Unlock()

	if err := ptr.setLogOutput(file); err != nil {
		return err
	}

	return ptr.store.SetLogFile(file)
}

func (ptr *logManagerT) clearRevertStateLocked() error {
	return ptr.store.SetLogRevertAt(time.Time{})
}
