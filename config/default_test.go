package config_test

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/cliffhanger/config"
	"github.com/futurehomeno/cliffhanger/telemetry/types"
)

func TestDefault_DefaultConfigIfMethods(t *testing.T) { //nolint:paralleltest
	d := &config.Default{}

	// GetTelemetry on a fresh value returns zero, no error.
	got, err := d.GetTelemetry()
	require.NoError(t, err)
	assert.Equal(t, types.TelemetryConfig{}, got)

	// SetTelemetry stores a copy by value.
	in := types.TelemetryConfig{Enabled: true, Validity: time.Hour}
	d.SetTelemetry(in)
	require.NotNil(t, d.Telemetry)
	assert.Equal(t, in, *d.Telemetry)

	// Mutating the input afterwards must not affect persisted state.
	in.Enabled = false
	assert.True(t, d.Telemetry.Enabled, "SetTelemetry must store a copy, not retain the caller's reference")

	// SetConfiguredAt formats RFC3339Nano (sub-second precision).
	at := time.Date(2026, 5, 10, 12, 0, 0, 123456789, time.UTC)
	d.SetConfiguredAt(at)
	assert.Equal(t, "2026-05-10T12:00:00.123456789Z", d.ConfiguredAt)

	// IncrementRestartsCount returns the new value.
	assert.Equal(t, 1, d.IncrementRestartsCount())
	assert.Equal(t, 2, d.IncrementRestartsCount())
}

func newTestStore(t *testing.T) (*config.DefaultStore, *config.Default, *atomic.Int32) {
	t.Helper()

	cfg := &config.Default{}

	var saves atomic.Int32

	store := config.NewDefaultStore(
		func() *config.Default { return cfg },
		func() error { saves.Add(1); return nil },
	)

	return store, cfg, &saves
}

func TestDefaultStore_LogFields_RoundTrip(t *testing.T) { //nolint:paralleltest
	store, cfg, saves := newTestStore(t)

	require.NoError(t, store.SetLevel("debug"))
	require.NoError(t, store.SetFormat("json"))
	require.NoError(t, store.SetLogFile("/var/log/x.log"))
	require.NoError(t, store.SetLogRevertTimeout(2*time.Hour))

	at := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	require.NoError(t, store.SetLogRevertAt(at))

	assert.Equal(t, "debug", store.Level())
	assert.Equal(t, "json", store.Format())
	assert.Equal(t, "/var/log/x.log", store.LogFile())
	assert.Equal(t, 2*time.Hour, store.LogRevertTimeout())
	assert.True(t, store.LogRevertAt().Equal(at))

	assert.Equal(t, "debug", cfg.LogLevel)
	assert.Equal(t, "json", cfg.LogFormat)
	assert.Equal(t, "/var/log/x.log", cfg.LogFile)
	assert.Equal(t, 2*time.Hour, cfg.LogRevertTimeout)
	assert.True(t, cfg.LogRevertAt.Equal(at))

	assert.Equal(t, int32(5), saves.Load(), "each setter should call save once")
}

func TestDefaultStore_Telemetry_RoundTrip(t *testing.T) { //nolint:paralleltest
	store, cfg, saves := newTestStore(t)

	_, err := store.Telemetry()
	assert.Error(t, err, "fresh store has no telemetry block")

	tc := &types.TelemetryConfig{Enabled: true, Validity: 24 * time.Hour}
	require.NoError(t, store.SetTelemetry(tc))

	got, err := store.Telemetry()
	require.NoError(t, err)
	assert.Equal(t, *tc, got)
	assert.Same(t, tc, cfg.Telemetry)
	assert.Equal(t, int32(1), saves.Load())
}

func TestDefaultStore_Default_ReturnsSnapshot(t *testing.T) { //nolint:paralleltest
	store, cfg, _ := newTestStore(t)

	snap := store.Default()
	assert.NotSame(t, cfg, snap, "Default() must return a copy, not the live pointer")
	assert.Equal(t, *cfg, *snap)

	cfg.LogLevel = "trace"
	assert.Equal(t, "trace", store.Default().LogLevel, "subsequent calls reflect current state")

	snap.LogLevel = "warn"
	assert.NotEqual(t, "warn", cfg.LogLevel, "mutating the snapshot must not affect the live config")
}

func TestDefaultStore_Save_PropagatesError(t *testing.T) { //nolint:paralleltest
	cfg := &config.Default{}
	store := config.NewDefaultStore(
		func() *config.Default { return cfg },
		func() error { return errors.New("disk full") },
	)

	assert.EqualError(t, store.SetLevel("info"), "disk full")
	assert.EqualError(t, store.Save(), "disk full")
}

func TestDefaultStore_ConcurrentReadsAndWrites_AreRaceFree(t *testing.T) { //nolint:paralleltest
	store, _, _ := newTestStore(t)

	require.NoError(t, store.SetLevel("info"))

	const goroutines = 20
	const iters = 200

	var wg sync.WaitGroup

	wg.Add(goroutines * 2)

	for range goroutines {
		go func() {
			defer wg.Done()
			for j := 0; j < iters; j++ {
				_ = store.Level()
				_ = store.Format()
				_ = store.LogFile()
				_ = store.LogRevertTimeout()
				_ = store.LogRevertAt()
			}
		}()

		go func() {
			defer wg.Done()
			for j := 0; j < iters; j++ {
				_ = store.SetLevel("info")
				_ = store.SetFormat("text")
				_ = store.SetLogFile("/tmp/a.log")
				_ = store.SetLogRevertTimeout(time.Hour)
				_ = store.SetLogRevertAt(time.Now())
			}
		}()
	}

	wg.Wait()
}
