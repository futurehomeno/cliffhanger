package formatters

import (
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
)

// MaxLogEntries is the capacity of the ErrorHook ring buffer.
const MaxLogEntries = 64

type logEntry struct {
	level logrus.Level
	msg   string
}

// ErrorHook is a logrus hook that captures Warn and Error level entries into a
// ring buffer of MaxLogEntries. It implements lifecycle.LogStatsProvider and
// diagnostic.ErrorsReporter so it can be wired directly to both.
type ErrorHook struct {
	mu      sync.Mutex
	entries []logEntry
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
	f := &logrus.TextFormatter{DisableColors: true}

	b, err := f.Format(entry)
	if err != nil {
		return err
	}

	e := logEntry{
		level: entry.Level,
		msg:   strings.TrimRight(string(b), "\n"),
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if len(h.entries) >= MaxLogEntries {
		h.entries = h.entries[1:]
	}

	h.entries = append(h.entries, e)

	return nil
}

// ErrorsReport implements diagnostic.ErrorsReporter.
func (h *ErrorHook) ErrorsReport() ([]string, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	result := make([]string, len(h.entries))
	for i, e := range h.entries {
		result[i] = e.msg
	}

	return result, nil
}

// ErrorsCount implements lifecycle.LogStatsProvider.
func (h *ErrorHook) ErrorsCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()

	count := 0
	for _, e := range h.entries {
		if e.level == logrus.ErrorLevel {
			count++
		}
	}

	return count
}

// WarningsCount implements lifecycle.LogStatsProvider.
func (h *ErrorHook) WarningsCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()

	count := 0
	for _, e := range h.entries {
		if e.level == logrus.WarnLevel {
			count++
		}
	}

	return count
}
