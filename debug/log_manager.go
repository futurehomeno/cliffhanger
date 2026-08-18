package debug

import (
	"bufio"
	"errors"
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
	previous := logManager
	previousFormatter := logrus.StandardLogger().Formatter
	previousLevel := logrus.GetLevel()

	logManager = &logManagerT{
		store: store,
	}

	setLogFormat(store.Format())

	// applyPersistedLevel returns an error when the persisted level is
	// unparseable, but it has already fallen back to InfoLevel by then.
	// Swallow it so a bad log_level value in config.json does not prevent
	// startup.
	_ = logManager.applyPersistedLevel()

	// The new output is validated (and, on success, wired into logrus) before the previous
	// manager is torn down. On failure, restore it along with the format/level globals set above:
	// logrus keeps pointing at a manager that is still open and still flushing, using the same
	// settings it had before this call, rather than one this call already closed with a mix of
	// old and new global state. A first initialization has nothing to restore and keeps the new
	// manager instead of leaving the global nil, which Route panics on: an application that logs
	// the error and carries on then runs without file logging rather than crash looping.
	if err := logManager.setLogOutput(store.LogFile()); err != nil {
		if previous != nil {
			logManager = previous
			logrus.SetFormatter(previousFormatter)
			logrus.SetLevel(previousLevel)
		}

		return err
	}

	logManager.startFlusher()

	if previous != nil {
		previous.stopFlusher()

		// Close (which flushes) rather than drop: the old logOutput may hold up to
		// logBufferSize of not-yet-written lines, and nothing else keeps a reference to it
		// now that logrus has been switched to the new one above.
		if previous.logOutput != nil {
			if err := previous.logOutput.Close(); err != nil {
				logrus.Errorf("[cliff] Close previous log output err: %v", err)
			}
		}
	}

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
		logrus.Errorf("[cliff] Flush log output err: %v", err)
	}
}

// flushInterval returns the interval the periodic flusher should currently use: short while
// debug/trace is active and its revert deadline (if any) has not yet elapsed, the configured (or
// default) one otherwise. Called fresh on every tick by the flusher goroutine (see
// startFlusherLocked), so a deadline elapsing mid-run is noticed on its own within one
// debug-interval tick - not just the next time something else happens to call restartFlusher.
func (ptr *logManagerT) flushInterval() time.Duration {
	if logrus.GetLevel() >= logrus.DebugLevel {
		revertAt := ptr.store.LogRevertAt()
		if revertAt.IsZero() || time.Now().Before(revertAt) {
			return debugLogFlushInterval
		}
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

				// The revert deadline (or log_flush_interval itself) can change without any
				// SetLevel/SetFile call in between - just time passing. Recheck each tick so the
				// ticker adjusts on its own instead of staying fast indefinitely once armed.
				if next := ptr.flushInterval(); next != interval {
					interval = next
					ticker.Reset(interval)
				}
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
		logrus.Warnf("[cliff] Invalid log level %q, revert to %s", ptr.store.Level(), logrus.InfoLevel)

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
				logrus.Warnf("[cliff] Clear log revert state err: %v", err)
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

	// Both run: returning on a failed flush (a full disk being the likely cause) would leak the
	// rotating file, which nothing else holds a reference to once logrus is switched away.
	return errors.Join(w.buf.Flush(), w.file.Close())
}

func (ptr *logManagerT) setLogOutput(logFile string) error {
	if logFile == "" {
		return fmt.Errorf("log file not set")
	}

	if err := os.MkdirAll(filepath.Dir(logFile), 0755); err != nil { //nolint:gosec
		return fmt.Errorf("create log dir=%s err: %w", filepath.Dir(logFile), err)
	}

	// Opened for writing, exactly as lumberjack will: a read-only probe accepts a file whose
	// first buffered flush then fails, long after this call reported the path usable.
	if f, err := os.OpenFile(logFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644); err != nil { //nolint:gosec
		return fmt.Errorf("open log file=%s err: %w", logFile, err)
	} else if cerr := f.Close(); cerr != nil {
		logrus.Errorf("[cliff] Close err: %v", cerr)
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
			logrus.Errorf("[cliff] Close previous log output err: %v", err)
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

	// Only a change to the flush cadence itself justifies restarting the ticker: doing it
	// unconditionally on every SetLevel (e.g. info -> warn, or re-setting the same level) would
	// reset an already-close-to-due flush back to a full interval for no reason.
	oldInterval := ptr.flushInterval()

	if logLevel < logrus.DebugLevel {
		if err := ptr.store.SetLevel(logLevel.String()); err != nil {
			return err
		}

		if err := ptr.clearRevertStateLocked(); err != nil {
			logrus.Warnf("[cliff] Clear log revert state err: %v, retry next startup", err)
		}

		logrus.SetLevel(logLevel)

		if ptr.flushInterval() != oldInterval {
			ptr.restartFlusher()
		}

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

	if ptr.flushInterval() != oldInterval {
		ptr.restartFlusher()
	}

	logrus.Infof("[cliff] Log level updated to %s, revert to info after %s", logLevel, timeout)

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

	// The command only carries a plain file name, which used to be handed to lumberjack as it is
	// and so resolved against the process working directory - under systemd, wherever that happens
	// to be. Keep the new file next to the current one and persist it absolute, so a restart does
	// not send the logs somewhere else again. A current path that is itself relative (an older
	// install) or missing falls back to the working directory, resolved once here rather than
	// left to drift with it.
	if !filepath.IsAbs(file) {
		if current := ptr.store.LogFile(); current != "" {
			file = filepath.Join(filepath.Dir(current), file)
		}

		if abs, err := filepath.Abs(file); err == nil {
			file = abs
		}
	}

	if err := ptr.setLogOutput(file); err != nil {
		return err
	}

	// A first initialization that could not open its log file keeps the manager but never reaches
	// startFlusher, so this is the recovery path: without it everything below error level stays in
	// the buffer until it fills, as only urgent writes flush on their own.
	if ptr.flushStop == nil {
		ptr.startFlusherLocked()
	}

	return ptr.store.SetLogFile(file)
}

func (ptr *logManagerT) clearRevertStateLocked() error {
	return ptr.store.SetLogRevertAt(time.Time{})
}
