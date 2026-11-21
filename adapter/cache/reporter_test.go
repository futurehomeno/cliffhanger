package cache_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/adapter/cache"
)

func TestReportingStrategies(t *testing.T) {
	t.Parallel()

	recent := time.Now()
	old := time.Now().Add(-2 * time.Minute)

	tcs := []struct {
		name         string
		strategy     cache.ReportingStrategy
		hasChanged   bool
		lastReported time.Time
		want         bool
	}{
		{"always, unchanged", cache.ReportAlways(), false, recent, true},
		{"on change only, unchanged", cache.ReportOnChangeOnly(), false, old, false},
		{"on change only, changed", cache.ReportOnChangeOnly(), true, recent, true},
		{"at least every, unchanged within interval", cache.ReportAtLeastEvery(time.Minute), false, recent, false},
		{"at least every, unchanged past interval", cache.ReportAtLeastEvery(time.Minute), false, old, true},
		{"at least every, changed within interval", cache.ReportAtLeastEvery(time.Minute), true, recent, true},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, tc.strategy.ReportRequired(tc.hasChanged, tc.lastReported))
		})
	}
}

func TestReportingCache_FirstReportIgnoresStrategy(t *testing.T) {
	t.Parallel()

	never := cache.ReportingStrategyFn(func(bool, time.Time) bool { return false })

	c := cache.NewReportingCache()
	assert.True(t, c.ReportRequired(never, "key", "sub", 1), "an unseen key must be reported")
	assert.True(t, c.HasChanged("key", "sub", 1))

	c.Reported("key", "sub", 1)
	assert.False(t, c.ReportRequired(never, "key", "sub", 1), "a seen key must defer to the strategy")
	assert.True(t, c.ReportRequired(never, "key", "other", 1), "sub keys must be tracked separately")
	assert.True(t, c.ReportRequired(never, "other", "sub", 1), "keys must be tracked separately")
}

func TestReportingCache_ComparesByValue(t *testing.T) {
	t.Parallel()

	type report struct {
		Status string
		Nodes  []string
	}

	c := cache.NewReportingCache()
	c.Reported("key", "", &report{Status: "UP", Nodes: []string{"a"}})

	assert.False(t, c.HasChanged("key", "", &report{Status: "UP", Nodes: []string{"a"}}), "an equal value behind a new pointer is unchanged")
	assert.False(t, c.ReportRequired(cache.ReportOnChangeOnly(), "key", "", &report{Status: "UP", Nodes: []string{"a"}}))
	assert.True(t, c.ReportRequired(cache.ReportOnChangeOnly(), "key", "", &report{Status: "DOWN", Nodes: []string{"a"}}))

	c.Reported("key", "", &report{Status: "DOWN"})
	assert.False(t, c.HasChanged("key", "", &report{Status: "DOWN"}), "Reported must replace the stored value")
}
