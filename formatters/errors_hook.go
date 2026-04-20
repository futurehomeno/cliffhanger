package formatters

import (
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// MaxLogEntries is the capacity of the ErrorHook ring buffer.
const MaxLogEntries = 64

// LogRetention is the maximum age of an entry kept in the ErrorHook ring buffer.
const LogRetention = 30 * 24 * time.Hour

var errorHookFormatter = &logrus.TextFormatter{DisableColors: true}

type logEntry struct {
	level logrus.Level
	msg   string
	time  time.Time
}

// ErrorHook is a logrus hook that captures Warn and Error level entries into a
// ring buffer of MaxLogEntries. It implements lifecycle.LogStatsProvider and
// diagnostic.ErrorsReporter so it can be wired directly to both.
type ErrorHook struct {
	mu      sync.Mutex
	entries [MaxLogEntries]logEntry
	head    int
	count   int
}

// NewErrorHook creates a new ErrorHook with MaxLogEntries capacity.
func NewErrorHook() *ErrorHook {
	return &ErrorHook{}
}

// Levels implements logrus.Hook.
func (h *ErrorHook) Levels() []logrus.Level {
	return []logrus.Level{logrus.WarnLevel, logrus.ErrorLevel}
}

// Fire implements logrus.Hook.
func (h *ErrorHook) Fire(entry *logrus.Entry) error {
	b, err := errorHookFormatter.Format(entry)
	if err != nil {
		return err
	}

	e := logEntry{
		level: entry.Level,
		msg:   strings.TrimRight(string(b), "\n"),
		time:  entry.Time,
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if h.count < MaxLogEntries {
		h.entries[(h.head+h.count)%MaxLogEntries] = e
		h.count++
	} else {
		h.entries[h.head] = e
		h.head = (h.head + 1) % MaxLogEntries
	}

	return nil
}

// ErrorsReport implements diagnostic.ErrorsReporter.
func (h *ErrorHook) ErrorsReport() ([]string, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.purgeExpired(time.Now())

	result := make([]string, h.count)
	for i := 0; i < h.count; i++ {
		result[i] = h.entries[(h.head+i)%MaxLogEntries].msg
	}

	return result, nil
}

// ErrorsCount implements lifecycle.LogStatsProvider.
func (h *ErrorHook) ErrorsCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.purgeExpired(time.Now())

	count := 0
	for i := 0; i < h.count; i++ {
		if h.entries[(h.head+i)%MaxLogEntries].level == logrus.ErrorLevel {
			count++
		}
	}

	return count
}

// WarningsCount implements lifecycle.LogStatsProvider.
func (h *ErrorHook) WarningsCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.purgeExpired(time.Now())

	count := 0
	for i := 0; i < h.count; i++ {
		if h.entries[(h.head+i)%MaxLogEntries].level == logrus.WarnLevel {
			count++
		}
	}

	return count
}

// purgeExpired drops entries older than LogRetention from the head of the ring
// buffer. Callers must hold h.mu. Entries are inserted chronologically, so any
// expired entries are contiguous at the head.
func (h *ErrorHook) purgeExpired(now time.Time) {
	cutoff := now.Add(-LogRetention)
	for h.count > 0 && h.entries[h.head].time.Before(cutoff) {
		h.entries[h.head] = logEntry{}
		h.head = (h.head + 1) % MaxLogEntries
		h.count--
	}
}
