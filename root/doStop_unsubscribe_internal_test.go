package root

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/cliffhanger/task"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

func stoppingApp(t *testing.T, clientID string, connect bool, topics ...string) *app {
	t.Helper()

	mqtt := suite.DefaultMQTT(clientID, "", "", "")
	if connect {
		require.NoError(t, mqtt.Start(5*time.Second))
	}

	tm := task.NewManager()
	require.NoError(t, tm.Start())

	return &app{
		running:            true,
		mqtt:               mqtt,
		messageRouter:      noopRouter{},
		taskManager:        tm,
		topicSubscriptions: topics,
	}
}

func TestDoStop_DisconnectedSkipsUnsubscribe(t *testing.T) { //nolint:paralleltest
	a := stoppingApp(t, "root_doStop_disconnected", false, "pt:j1/mt:cmd/rt:app/rn:test_app/ad:1")

	assert.NoError(t, a.doStop(), "a stop after broker loss has nothing to unsubscribe from")
	assert.False(t, a.running)
}

func TestDoStop_ConnectedUnsubscribeErrorIsReturned(t *testing.T) { //nolint:paralleltest
	// An invalid filter makes the broker drop the connection, failing the unsubscribe while connected.
	a := stoppingApp(t, "root_doStop_connected", true, "a/#/b")

	err := a.doStop()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsubscribe topic=a/#/b err:")
}
