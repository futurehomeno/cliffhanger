package discovery_test

import (
	"encoding/json"
	"testing"

	"github.com/futurehomeno/fimpgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/cliffhanger/discovery"
	"github.com/futurehomeno/cliffhanger/lifecycle"
)

func newDiscoveryRequest(t *testing.T) *fimpgo.Message {
	t.Helper()

	addr, err := fimpgo.NewAddressFromString("pt:j1/mt:cmd/rt:discovery")
	require.NoError(t, err)

	return &fimpgo.Message{
		Topic:   addr.Serialize(),
		Addr:    addr,
		Payload: fimpgo.NewNullMessage(discovery.CmdDiscoveryRequest, discovery.Service, nil, nil, nil),
	}
}

func TestHandle_WithoutLifecycle_EmitsReportWithoutStates(t *testing.T) {
	t.Parallel()

	handler := discovery.Handle("my_app", discovery.ResourceTypeApp, "my_app", "1", "1.0.0", nil)

	reply := handler.Handle(newDiscoveryRequest(t))

	require.NotNil(t, reply)
	require.NotNil(t, reply.Payload)
	assert.Equal(t, discovery.EvtDiscoveryReport, reply.Payload.Interface)

	raw, err := json.Marshal(reply.Payload.Value)
	require.NoError(t, err)

	var report map[string]any
	require.NoError(t, json.Unmarshal(raw, &report))

	assert.Equal(t, "my_app", report["resource_name"])
	assert.Equal(t, discovery.ResourceTypeApp, report["resource_type"])
	assert.Equal(t, "my_app", report["package_name"])
	assert.Equal(t, "1", report["instance_id"])
	assert.Equal(t, "1.0.0", report["version"])
	assert.Empty(t, report["states"])
}

func TestHandle_WithLifecycle_EmitsFreshStatesOnEachRequest(t *testing.T) {
	t.Parallel()

	l := lifecycle.New(nil)

	handler := discovery.Handle("my_app", discovery.ResourceTypeApp, "my_app", "1", "1.0.0", l)

	firstReply := handler.Handle(newDiscoveryRequest(t))
	require.NotNil(t, firstReply)

	l.SetAppState(lifecycle.AppStateRunning, nil)

	secondReply := handler.Handle(newDiscoveryRequest(t))
	require.NotNil(t, secondReply)

	unmarshalStates := func(t *testing.T, reply *fimpgo.Message) map[string]any {
		t.Helper()

		raw, err := json.Marshal(reply.Payload.Value)
		require.NoError(t, err)

		var report map[string]any
		require.NoError(t, json.Unmarshal(raw, &report))

		states, ok := report["states"].(map[string]any)
		require.True(t, ok, "states missing or wrong type")

		return states
	}

	firstStates := unmarshalStates(t, firstReply)
	secondStates := unmarshalStates(t, secondReply)

	assert.NotEqual(t, firstStates["app"], secondStates["app"], "second reply must reflect state change")
	assert.Equal(t, string(lifecycle.AppStateRunning), secondStates["app"])
}
