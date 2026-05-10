// Internal-package tests so they can drive the unexported poll method and
// configResponseT type directly. Reach for testpackage isolation in a
// separate _test.go if/when those are exposed.
package config_pull //nolint:testpackage

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	mocked "github.com/futurehomeno/cliffhanger/test/mocks/telemetry/config_pull"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

func newConfig() *Config {
	return New(nil, "src", func(_ bool, _ []string) {})
}

func makeResponse(t *testing.T, body configResponseT) *fimpgo.FimpMessage {
	t.Helper()

	raw, err := json.Marshal(body)
	require.NoError(t, err)

	return &fimpgo.FimpMessage{
		Interface: EvtConfigReport,
		ValueType: "object",
		ValueObj:  raw,
	}
}

func TestPoll_OK_AppliesConfig(t *testing.T) { //nolint:paralleltest
	cfg := newConfig()

	want := configResponseT{Enabled: true, SuppressedDomains: []string{"x"}}

	client := mocked.NewSyncRequester(t)
	client.EXPECT().
		SendFimp(mock.Anything, mock.Anything, mock.Anything).
		Return(makeResponse(t, want), nil)

	got := cfg.poll(client)

	require.NotNil(t, got.cfg)
	assert.Equal(t, want.Enabled, got.cfg.Enabled)
	assert.Equal(t, want.SuppressedDomains, got.cfg.SuppressedDomains)
	assert.Equal(t, cfg.fallbackPoll, got.delay, "no next_update -> fallback")
}

func TestPoll_SendFimpError_ReturnsFallback(t *testing.T) { //nolint:paralleltest
	cfg := newConfig()

	client := mocked.NewSyncRequester(t)
	client.EXPECT().
		SendFimp(mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("timeout"))

	got := cfg.poll(client)

	assert.Nil(t, got.cfg)
	assert.Equal(t, cfg.fallbackPoll, got.delay)
}

func TestPoll_UnexpectedResponseType_ReturnsFallback(t *testing.T) { //nolint:paralleltest
	cfg := newConfig()

	client := mocked.NewSyncRequester(t)
	resp := &fimpgo.FimpMessage{Interface: "evt.something.else", ValueType: "string", Value: "x"}
	client.EXPECT().
		SendFimp(mock.Anything, mock.Anything, mock.Anything).
		Return(resp, nil)

	got := cfg.poll(client)

	assert.Nil(t, got.cfg)
	assert.Equal(t, cfg.fallbackPoll, got.delay)
}

func TestPoll_BadObjectValue_ReturnsFallback(t *testing.T) { //nolint:paralleltest
	cfg := newConfig()

	client := mocked.NewSyncRequester(t)
	// Garbage raw bytes - GetObjectValue will fail to unmarshal.
	resp := &fimpgo.FimpMessage{
		Interface: EvtConfigReport,
		ValueType: "object",
		ValueObj:  []byte(`not-json`),
	}
	client.EXPECT().
		SendFimp(mock.Anything, mock.Anything, mock.Anything).
		Return(resp, nil)

	got := cfg.poll(client)

	assert.Nil(t, got.cfg)
	assert.Equal(t, cfg.fallbackPoll, got.delay)
}

func TestPoll_NextUpdateInFuture_UsedAsDelay(t *testing.T) { //nolint:paralleltest
	cfg := newConfig()

	next := time.Now().Add(2 * time.Hour).UTC().Format(time.RFC3339)

	client := mocked.NewSyncRequester(t)
	client.EXPECT().
		SendFimp(mock.Anything, mock.Anything, mock.Anything).
		Return(makeResponse(t, configResponseT{Enabled: true, NextUpdate: next}), nil)

	got := cfg.poll(client)

	require.NotNil(t, got.cfg)
	// Allow a small buffer for scheduling latency.
	assert.Greater(t, got.delay, time.Hour)
	assert.LessOrEqual(t, got.delay, 2*time.Hour+time.Second)
}

func TestPoll_NextUpdateInvalid_FallsBack(t *testing.T) { //nolint:paralleltest
	cfg := newConfig()

	client := mocked.NewSyncRequester(t)
	client.EXPECT().
		SendFimp(mock.Anything, mock.Anything, mock.Anything).
		Return(makeResponse(t, configResponseT{Enabled: true, NextUpdate: "not-a-timestamp"}), nil)

	got := cfg.poll(client)

	require.NotNil(t, got.cfg, "config still applied even when next_update is bad")
	assert.Equal(t, cfg.fallbackPoll, got.delay)
}

func TestPoll_NextUpdateExceedsMax_ClampedToMax(t *testing.T) { //nolint:paralleltest
	cfg := newConfig()

	next := time.Now().Add(72 * time.Hour).UTC().Format(time.RFC3339)

	client := mocked.NewSyncRequester(t)
	client.EXPECT().
		SendFimp(mock.Anything, mock.Anything, mock.Anything).
		Return(makeResponse(t, configResponseT{NextUpdate: next}), nil)

	got := cfg.poll(client)

	assert.Equal(t, MaxPollInterval, got.delay)
}

func TestStart_ValidationErrors(t *testing.T) { //nolint:paralleltest
	cases := []struct {
		name   string
		mutate func(*Config)
	}{
		{"non-positive fallback", func(c *Config) { c.fallbackPoll = 0 }},
		{"non-positive timeout", func(c *Config) { c.timeout = 0 }},
		{"empty topic", func(c *Config) { c.requestTopic = "" }},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) { //nolint:paralleltest
			cfg := newConfig()
			tc.mutate(cfg)
			assert.Error(t, cfg.Start())
		})
	}
}

func TestNew_PopulatesDefaults(t *testing.T) { //nolint:paralleltest
	cfg := newConfig()

	assert.Equal(t, DefaultPollInterval, cfg.fallbackPoll)
	assert.Equal(t, ConfigRequestTopic, cfg.requestTopic)
	assert.Equal(t, 30, cfg.timeout)
}

// TestStop_BeforeStart_NoOp ensures Stop is safe even when Start was never
// called (no goroutine, no client).
func TestStop_BeforeStart_NoOp(t *testing.T) { //nolint:paralleltest
	cfg := newConfig()

	assert.NotPanics(t, func() {
		cfg.Stop()
	})
}

// TestStart_RealClient_SchedulesAndStops drives the full Start/Stop path
// end-to-end against a real MQTT broker so we cover scheduleLocked and
// the goroutine bookkeeping in Stop.
func TestStart_RealClient_SchedulesAndStops(t *testing.T) { //nolint:paralleltest
	mqtt := suite.DefaultMQTT("cliff_cfgpull_start", "", "", "")
	require.NoError(t, mqtt.Start(2*time.Second))
	t.Cleanup(mqtt.Stop)

	cfg := New(mqtt, "src", func(_ bool, _ []string) {})

	require.NoError(t, cfg.Start())
	// Calling Start a second time is a no-op when already running.
	require.NoError(t, cfg.Start())

	cfg.Stop()
	// Stop is idempotent.
	cfg.Stop()
}

// TestSchedule_RepeatsOnFallbackPoll verifies that scheduleLocked re-fires
// after fallbackPoll: a single Start+wait must observe at least two SendFimp
// calls. Without this, a regression that broke the self-rescheduling tail of
// scheduleLocked (e.g. a missed scheduleLocked call after poll) would still
// pass TestStart_RealClient_SchedulesAndStops, since that one only proves
// Start does not error.
func TestSchedule_RepeatsOnFallbackPoll(t *testing.T) { //nolint:paralleltest
	var calls atomic.Int32

	client := mocked.NewSyncRequester(t)
	client.EXPECT().
		SendFimp(mock.Anything, mock.Anything, mock.Anything).
		Run(func(_ string, _ *fimpgo.FimpMessage, _ int) {
			calls.Add(1)
		}).
		Return(makeResponse(t, configResponseT{Enabled: true}), nil)
	client.EXPECT().Stop().Return().Maybe()

	cfg := newConfig()
	cfg.fallbackPoll = 30 * time.Millisecond
	cfg.client = client

	cfg.lock.Lock()
	cfg.scheduleLocked(0)
	cfg.lock.Unlock()

	t.Cleanup(cfg.Stop)

	require.Eventually(t,
		func() bool { return calls.Load() >= 2 },
		1*time.Second,
		10*time.Millisecond,
		"scheduleLocked must re-fire after fallbackPoll; got %d calls",
		calls.Load(),
	)
}

// TestPoll_ResponseTopicFormat verifies the SendFimp call uses the correct
// request topic and the message has the expected ResponseToTopic.
func TestPoll_ResponseTopicFormat(t *testing.T) { //nolint:paralleltest
	cfg := newConfig()

	client := mocked.NewSyncRequester(t)
	client.EXPECT().
		SendFimp(ConfigRequestTopic, mock.MatchedBy(func(msg *fimpgo.FimpMessage) bool {
			expected := fmt.Sprintf("pt:j1/mt:evt/rt:cloud/rn:%s/ad:telemetry-config", cfg.sourceRn)

			return msg.Interface == CmdGetConfig && msg.ResponseToTopic == expected
		}), 30).
		Return(makeResponse(t, configResponseT{Enabled: true}), nil)

	cfg.poll(client)
}
