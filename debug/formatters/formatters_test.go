package formatters_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/cliffhanger/debug/formatters"
)

func fire(t *testing.T, h *formatters.ErrorHook, level logrus.Level, at time.Time, msg string) {
	t.Helper()

	require.NoError(t, h.Fire(&logrus.Entry{Logger: logrus.New(), Level: level, Time: at, Message: msg, Data: logrus.Fields{}}))
}

func TestErrorHook_KeepsMostRecentEntriesInOrder(t *testing.T) {
	t.Parallel()

	h := formatters.NewErrorHook()
	now := time.Now()

	for i := range 100 {
		fire(t, h, logrus.WarnLevel, now, fmt.Sprintf("warn-%03d", i))
	}

	report, err := h.ErrorsReport()
	require.NoError(t, err)
	require.Len(t, report, formatters.MaxLogEntries)

	for i, line := range report {
		assert.Contains(t, line, fmt.Sprintf("warn-%03d", 100-formatters.MaxLogEntries+i))
		assert.False(t, strings.HasSuffix(line, "\n"))
	}
}

func TestErrorHook_DropsEntriesOlderThanRetention(t *testing.T) {
	t.Parallel()

	h := formatters.NewErrorHook()
	now := time.Now()

	fire(t, h, logrus.ErrorLevel, now.Add(-formatters.LogRetention-time.Hour), "stale-1")
	fire(t, h, logrus.ErrorLevel, now.Add(-formatters.LogRetention-time.Minute), "stale-2")
	fire(t, h, logrus.ErrorLevel, now.Add(-time.Hour), "fresh")

	report, err := h.ErrorsReport()
	require.NoError(t, err)
	require.Len(t, report, 1)
	assert.Contains(t, report[0], "fresh")

	fire(t, h, logrus.WarnLevel, now, "next")

	report, err = h.ErrorsReport()
	require.NoError(t, err)
	require.Len(t, report, 2, "purged slots must be freed, not reported again")
	assert.Contains(t, report[1], "next")
}

func TestErrorHook_CapturesOnlyWarnAndAbove(t *testing.T) {
	t.Parallel()

	logger := logrus.New()
	logger.SetOutput(&strings.Builder{})
	logger.SetLevel(logrus.TraceLevel)

	h := formatters.NewErrorHook()
	logger.AddHook(h)

	logger.Info("info-line")
	logger.Debug("debug-line")
	logger.Warn("warn-line")
	logger.Error("error-line")

	report, err := h.ErrorsReport()
	require.NoError(t, err)
	require.Len(t, report, 2)
	assert.Contains(t, report[0], "warn-line")
	assert.Contains(t, report[1], "error-line")
	assert.NotContains(t, report[0], "\x1b[", "entries must be rendered without colors")
}

func TestBudzikFormatter_LineShape(t *testing.T) {
	t.Parallel()

	at := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)

	tt := []struct {
		level logrus.Level
		want  string
	}{
		{logrus.PanicLevel, "PANIC"},
		{logrus.FatalLevel, "FATAL"},
		{logrus.ErrorLevel, "E"},
		{logrus.WarnLevel, "W"},
		{logrus.InfoLevel, "I"},
		{logrus.DebugLevel, "D"},
		{logrus.TraceLevel, "T"},
		{logrus.Level(42), "D"},
	}

	for _, tc := range tt {
		b, err := formatters.NewBudzikFormatter().Format(&logrus.Entry{
			Time:    at,
			Level:   tc.level,
			Message: "hello",
			Data:    logrus.Fields{"b": 2, "a": 1},
		})
		require.NoError(t, err)
		assert.Equal(t, "26 03-04 05:06:07 "+tc.want+" hello a=1 b=2\n", string(b))
	}
}
