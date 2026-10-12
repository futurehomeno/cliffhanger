package hub_test

import (
	"testing"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/cliffhanger/hub"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

const clbridgeTopic = "pt:j1/mt:cmd/rt:app/rn:clbridge/ad:1"

func startCloudBridge(t *testing.T, reply func(req *fimpgo.FimpMessage) *fimpgo.FimpMessage) {
	t.Helper()

	mqtt := suite.DefaultMQTT("cliff_test_clbridge", "", "", "")
	require.NoError(t, mqtt.Start(2*time.Second))
	require.NoError(t, mqtt.Subscribe(clbridgeTopic))

	ch := make(fimpgo.MessageCh, 4)
	mqtt.RegisterChannel("clbridge", ch)
	done := make(chan struct{})

	go func() {
		for {
			select {
			case <-done:
				return
			case msg := <-ch:
				if msg.Payload.Interface != "cmd.clbridge.get_auth_token" {
					continue
				}

				_ = mqtt.PublishToTopic(msg.Payload.ResponseToTopic, reply(msg.Payload))
			}
		}
	}()

	t.Cleanup(func() {
		close(done)
		mqtt.UnregisterChannel("clbridge")
		mqtt.Stop()
	})
}

func newLoader(rn fimptype.ResourceNameT) hub.TokenLoader {
	return hub.NewTokenLoader(&hub.TokenLoaderConfig{
		ResourceName:  rn,
		MQTTServerURI: "tcp://127.0.0.1:11883",
		Retry:         1,
		RetryDelay:    time.Millisecond,
	})
}

func TestTokenLoader_ReturnsToken(t *testing.T) { //nolint:paralleltest
	sources := make(chan fimptype.ResourceNameT, 1)

	startCloudBridge(t, func(req *fimpgo.FimpMessage) *fimpgo.FimpMessage {
		sources <- req.Source

		return fimpgo.NewStringMessage("evt.clbridge.auth_token_report", "clbridge", "hub-token", nil, nil, req)
	})

	token, err := newLoader("token_ok").LoadToken()
	require.NoError(t, err)
	assert.Equal(t, "hub-token", token)
	assert.Equal(t, fimptype.ResourceNameT("token_ok"), <-sources)
}

func TestTokenLoader_RejectsWrongReportType(t *testing.T) { //nolint:paralleltest
	startCloudBridge(t, func(req *fimpgo.FimpMessage) *fimpgo.FimpMessage {
		return fimpgo.NewStringMessage("evt.error.report", "clbridge", "nope", nil, nil, req)
	})

	token, err := newLoader("token_wrong_type").LoadToken()
	require.ErrorContains(t, err, "evt.clbridge.auth_token_report")
	assert.ErrorContains(t, err, "evt.error.report")
	assert.Empty(t, token)
}

func TestTokenLoader_RejectsNonStringValue(t *testing.T) { //nolint:paralleltest
	startCloudBridge(t, func(req *fimpgo.FimpMessage) *fimpgo.FimpMessage {
		return fimpgo.NewIntMessage("evt.clbridge.auth_token_report", "clbridge", 7, nil, nil, req)
	})

	token, err := newLoader("token_non_string").LoadToken()
	require.ErrorContains(t, err, "wrong message format")
	assert.Empty(t, token)
}

func TestTokenLoader_NonPositiveRetryUsesDefault(t *testing.T) { //nolint:paralleltest
	startCloudBridge(t, func(req *fimpgo.FimpMessage) *fimpgo.FimpMessage {
		return fimpgo.NewStringMessage("evt.clbridge.auth_token_report", "clbridge", "hub-token", nil, nil, req)
	})

	token, err := hub.NewTokenLoader(&hub.TokenLoaderConfig{
		ResourceName:  "token_negative_retry",
		MQTTServerURI: "tcp://127.0.0.1:11883",
		Retry:         -1,
	}).LoadToken()
	require.NoError(t, err)
	assert.Equal(t, "hub-token", token)
}

func TestTokenLoader_NoDelayAfterFinalAttempt(t *testing.T) { //nolint:paralleltest
	loader := hub.NewTokenLoader(&hub.TokenLoaderConfig{
		ResourceName:  "token_no_bridge",
		MQTTServerURI: "tcp://127.0.0.1:11883",
		Retry:         1,
		RetryDelay:    10 * time.Second,
	})

	start := time.Now()
	_, err := loader.LoadToken()
	require.Error(t, err)
	assert.Less(t, time.Since(start), 10*time.Second)
}

func TestNewTokenLoader_LeavesCallerConfigUntouched(t *testing.T) {
	t.Parallel()

	cfg := &hub.TokenLoaderConfig{ResourceName: "token_cfg"}
	want := *cfg

	hub.NewTokenLoader(cfg)
	hub.NewTokenLoader(cfg)

	assert.Equal(t, want, *cfg)
}
