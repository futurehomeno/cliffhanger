package config_poll //nolint:testpackage

import (
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/cliffhanger/telemetry/types"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

func newConfig() *Config {
	return New(nil, "src", func(_ bool, _ map[string]types.SuppressedEntry) {})
}

func publishConfigResponse(t *testing.T, mqtt *fimpgo.MqttTransport, body configResponseT) {
	t.Helper()

	raw, err := json.Marshal(body)
	require.NoError(t, err)

	msg := &fimpgo.FimpMessage{
		Interface: EvtConfigReport,
		ValueType: "object",
		ValueObj:  raw,
	}

	require.NoError(t, mqtt.PublishToTopic(ConfigResponseTopic, msg))
}

func TestNew_PopulatesDefaults(t *testing.T) { //nolint:paralleltest
	cfg := newConfig()

	assert.Equal(t, DefaultPollInterval, cfg.fallbackPoll)
	assert.Equal(t, ConfigRequestTopic, cfg.requestTopic)
}

func TestStart_ValidationErrors(t *testing.T) { //nolint:paralleltest
	cases := []struct {
		name   string
		mutate func(*Config)
	}{
		{"non-positive fallback", func(c *Config) { c.fallbackPoll = 0 }},
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

func TestStop_BeforeStart_NoOp(t *testing.T) { //nolint:paralleltest
	cfg := newConfig()

	assert.NotPanics(t, func() {
		cfg.Stop()
	})
}

func TestStart_SubscribesAndStops(t *testing.T) { //nolint:paralleltest
	mqtt := suite.DefaultMQTT("cliff_cfgpoll_start", "", "", "")
	require.NoError(t, mqtt.Start(2*time.Second))
	t.Cleanup(mqtt.Stop)

	cfg := New(mqtt, "src", func(_ bool, _ map[string]types.SuppressedEntry) {})

	require.NoError(t, cfg.Start())
	require.NoError(t, cfg.Start()) // idempotent

	cfg.Stop()
	cfg.Stop() // idempotent
}

func TestListen_AppliesConfigOnMessage(t *testing.T) { //nolint:paralleltest
	mqtt := suite.DefaultMQTT("cliff_cfgpoll_listen", "", "", "")
	require.NoError(t, mqtt.Start(2*time.Second))
	t.Cleanup(mqtt.Stop)

	var applied atomic.Bool

	cfg := New(mqtt, "src", func(enabled bool, _ map[string]types.SuppressedEntry) {
		if enabled {
			applied.Store(true)
		}
	})

	require.NoError(t, cfg.Start())
	t.Cleanup(cfg.Stop)

	publishConfigResponse(t, mqtt, configResponseT{Enabled: true})

	require.Eventually(t, applied.Load, 2*time.Second, 20*time.Millisecond, "applyConfig must be called with enabled=true")
}

func TestListen_SetsLastReceivedAt(t *testing.T) { //nolint:paralleltest
	mqtt := suite.DefaultMQTT("cliff_cfgpoll_last_recv", "", "", "")
	require.NoError(t, mqtt.Start(2*time.Second))
	t.Cleanup(mqtt.Stop)

	cfg := New(mqtt, "src", func(_ bool, _ map[string]types.SuppressedEntry) {})

	require.NoError(t, cfg.Start())
	t.Cleanup(cfg.Stop)

	before := time.Now()

	publishConfigResponse(t, mqtt, configResponseT{Enabled: true})

	require.Eventually(t, func() bool {
		cfg.lock.Lock()
		defer cfg.lock.Unlock()

		return cfg.lastReceivedAt.After(before)
	}, 2*time.Second, 20*time.Millisecond, "lastReceivedAt must be updated")
}

func TestListen_FreshConfigSkipsPoll(t *testing.T) { //nolint:paralleltest
	mqtt := suite.DefaultMQTT("cliff_cfgpoll_fresh", "", "", "")
	require.NoError(t, mqtt.Start(2*time.Second))
	t.Cleanup(mqtt.Stop)

	var pollRequests atomic.Int32

	// Subscribe to the request topic to count outgoing polls.
	require.NoError(t, mqtt.Subscribe(ConfigRequestTopic))

	pollCh := make(fimpgo.MessageCh, 4)
	mqtt.RegisterChannel("poll_count", pollCh)
	t.Cleanup(func() { mqtt.UnregisterChannel("poll_count") })

	cfg := New(mqtt, "src", func(_ bool, _ map[string]types.SuppressedEntry) {})
	cfg.fallbackPoll = 50 * time.Millisecond

	require.NoError(t, cfg.Start())
	t.Cleanup(cfg.Stop)

	// Mark config as freshly received.
	cfg.lock.Lock()
	cfg.lastReceivedAt = time.Now()
	cfg.lock.Unlock()

	// Count any outgoing poll requests in a short window.
	done := make(chan struct{})
	go func() {
		defer close(done)

		deadline := time.After(300 * time.Millisecond)

		for {
			select {
			case msg := <-pollCh:
				if msg != nil && msg.Payload != nil && msg.Payload.Interface == CmdGetConfig {
					pollRequests.Add(1)
				}
			case <-deadline:
				return
			}
		}
	}()

	<-done

	assert.Zero(t, pollRequests.Load(), "poll must be skipped when config is fresh")
}

func TestNextUpdate_FallsBackWhenEmpty(t *testing.T) { //nolint:paralleltest
	cfg := newConfig()

	d := cfg.nextUpdate("")
	assert.GreaterOrEqual(t, d, cfg.fallbackPoll)
	assert.Less(t, d, cfg.fallbackPoll+AdditionalRandomPollIntervalRange+time.Second)
}

func TestNextUpdate_NextUpdateInFuture(t *testing.T) { //nolint:paralleltest
	cfg := newConfig()

	next := time.Now().Add(2 * time.Hour).UTC().Format(time.RFC3339)
	d := cfg.nextUpdate(next)

	assert.Greater(t, d, time.Hour)
	assert.LessOrEqual(t, d, 2*time.Hour+AdditionalRandomPollIntervalRange+time.Second)
}

func TestNextUpdate_NextUpdateExceedsMax_ClampedToMax(t *testing.T) { //nolint:paralleltest
	cfg := newConfig()

	next := time.Now().Add(72 * time.Hour).UTC().Format(time.RFC3339)
	d := cfg.nextUpdate(next)

	assert.LessOrEqual(t, d, MaxPollInterval+AdditionalRandomPollIntervalRange)
}

func TestNextUpdate_InvalidNextUpdate_FallsBack(t *testing.T) { //nolint:paralleltest
	cfg := newConfig()

	d := cfg.nextUpdate("not-a-timestamp")
	assert.GreaterOrEqual(t, d, cfg.fallbackPoll)
}
