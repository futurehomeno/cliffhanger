package telemetry_test

import (
	"errors"
	"testing"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/cliffhanger/telemetry"
)

// mockSyncRequester implements telemetry.SyncRequester for testing.
type mockSyncRequester struct {
	response *fimpgo.FimpMessage
	err      error
	calls    int
}

func (m *mockSyncRequester) SendFimp(_ string, _ *fimpgo.FimpMessage, _ int) (*fimpgo.FimpMessage, error) {
	m.calls++

	if m.err != nil {
		return nil, m.err
	}

	return m.response, nil
}

func (m *mockSyncRequester) AddSubscription(_ string) error { return nil }
func (m *mockSyncRequester) Stop()                          {}

func configResponse(t *testing.T, enabled bool, suppressed []string, nextUpdate string) *fimpgo.FimpMessage {
	t.Helper()

	resp := &telemetry.ConfigResponse{
		Enabled:    enabled,
		Suppressed: suppressed,
		NextUpdate: nextUpdate,
	}

	msg := fimpgo.NewObjectMessage(telemetry.EvtConfigReport, telemetry.Service, resp, nil, nil, nil)

	// Round-trip through JSON to simulate MQTT serialization so that
	// GetObjectValue can parse the payload.
	raw, err := msg.SerializeToJson()
	require.NoError(t, err)

	parsed, err := fimpgo.NewMessageFromBytes(raw)
	require.NoError(t, err)

	return parsed
}

func TestNewConfigPull_Validation(t *testing.T) {
	t.Parallel()

	t.Run("nil mqtt", func(t *testing.T) {
		t.Parallel()

		_, err := telemetry.NewConfigPull(nil, "source", nil)
		assert.Error(t, err)
	})

	t.Run("empty source", func(t *testing.T) {
		t.Parallel()

		_, err := telemetry.NewConfigPull(&fimpgo.MqttTransport{}, "", nil)
		assert.Error(t, err)
	})

	t.Run("nil reporter", func(t *testing.T) {
		t.Parallel()

		_, err := telemetry.NewConfigPull(&fimpgo.MqttTransport{}, "source", nil)
		assert.Error(t, err)
	})
}

func TestConfigPull_AppliesConfig(t *testing.T) {
	t.Parallel()

	mock := &mockSyncRequester{
		response: configResponse(t, true, []string{"other-app"}, ""),
	}

	store := telemetry.NewMemoryStore(false)
	tel, err := telemetry.New(&fimpgo.MqttTransport{}, testSource, store)
	require.NoError(t, err)

	cp, err := telemetry.NewConfigPull(&fimpgo.MqttTransport{}, testSource, tel,
		telemetry.WithSyncRequester(mock),
	)
	require.NoError(t, err)

	require.NoError(t, cp.Start())

	// Give the async poll time to complete.
	time.Sleep(100 * time.Millisecond)

	assert.True(t, tel.IsEnabled(), "config should enable telemetry")
	assert.False(t, tel.IsSuppressed(), "source not in suppressed list")
	assert.GreaterOrEqual(t, mock.calls, 1)

	require.NoError(t, cp.Stop())
}

func TestConfigPull_SuppressesMatchingSource(t *testing.T) {
	t.Parallel()

	mock := &mockSyncRequester{
		response: configResponse(t, true, []string{testSource, "other-app"}, ""),
	}

	store := telemetry.NewMemoryStore(true)
	tel, err := telemetry.New(&fimpgo.MqttTransport{}, testSource, store)
	require.NoError(t, err)

	cp, err := telemetry.NewConfigPull(&fimpgo.MqttTransport{}, testSource, tel,
		telemetry.WithSyncRequester(mock),
	)
	require.NoError(t, err)

	require.NoError(t, cp.Start())

	time.Sleep(100 * time.Millisecond)

	assert.True(t, tel.IsSuppressed(), "source should be suppressed")

	require.NoError(t, cp.Stop())
}

func TestConfigPull_ErrorUsesFallback(t *testing.T) {
	t.Parallel()

	mock := &mockSyncRequester{
		err: errors.New("cloud unreachable"),
	}

	store := telemetry.NewMemoryStore(true)
	tel, err := telemetry.New(&fimpgo.MqttTransport{}, testSource, store)
	require.NoError(t, err)

	cp, err := telemetry.NewConfigPull(&fimpgo.MqttTransport{}, testSource, tel,
		telemetry.WithSyncRequester(mock),
		telemetry.WithFallbackPoll(50*time.Millisecond),
	)
	require.NoError(t, err)

	require.NoError(t, cp.Start())

	// Wait for initial poll + one retry.
	time.Sleep(150 * time.Millisecond)

	assert.GreaterOrEqual(t, mock.calls, 2, "should retry after fallback interval")

	require.NoError(t, cp.Stop())
}

func TestConfigPull_StopCancelsPending(t *testing.T) {
	t.Parallel()

	mock := &mockSyncRequester{
		response: configResponse(t, true, nil, time.Now().Add(time.Hour).Format(time.RFC3339)),
	}

	store := telemetry.NewMemoryStore(true)
	tel, err := telemetry.New(&fimpgo.MqttTransport{}, testSource, store)
	require.NoError(t, err)

	cp, err := telemetry.NewConfigPull(&fimpgo.MqttTransport{}, testSource, tel,
		telemetry.WithSyncRequester(mock),
	)
	require.NoError(t, err)

	require.NoError(t, cp.Start())

	time.Sleep(100 * time.Millisecond)

	callsBefore := mock.calls
	require.NoError(t, cp.Stop())

	// After stop, no more polls should fire.
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, callsBefore, mock.calls, "no more polls after Stop")
}

func TestConfigPull_PastNextUpdateUsesFallback(t *testing.T) {
	t.Parallel()

	pastTime := time.Now().Add(-time.Hour).Format(time.RFC3339)
	mock := &mockSyncRequester{
		response: configResponse(t, true, nil, pastTime),
	}

	store := telemetry.NewMemoryStore(true)
	tel, err := telemetry.New(&fimpgo.MqttTransport{}, testSource, store)
	require.NoError(t, err)

	cp, err := telemetry.NewConfigPull(&fimpgo.MqttTransport{}, testSource, tel,
		telemetry.WithSyncRequester(mock),
		telemetry.WithFallbackPoll(50*time.Millisecond),
	)
	require.NoError(t, err)

	require.NoError(t, cp.Start())

	// Wait for initial poll + fallback retry.
	time.Sleep(150 * time.Millisecond)

	assert.GreaterOrEqual(t, mock.calls, 2, "past next_update should use fallback interval")

	require.NoError(t, cp.Stop())
}
