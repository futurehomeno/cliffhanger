package app_test

import (
	"errors"
	"testing"

	"github.com/futurehomeno/fimpgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/cliffhanger/app"
	"github.com/futurehomeno/cliffhanger/lifecycle"
)

const testDiagService = "test_app"

// stubLogProvider is a test double for app.LogProvider.
type stubLogProvider struct {
	entries []string
	err     error
}

func (s *stubLogProvider) ErrorsReport() ([]string, error) {
	return s.entries, s.err
}

func newDiagRequest(t *testing.T) *fimpgo.Message {
	t.Helper()

	addr, err := fimpgo.NewAddressFromString("pt:j1/mt:cmd/rt:app/rn:" + testDiagService + "/ad:1")
	require.NoError(t, err)

	return &fimpgo.Message{
		Topic:   addr.Serialize(),
		Addr:    addr,
		Payload: fimpgo.NewNullMessage(app.CmdAppDiagGetReport, testDiagService, nil, nil, nil),
	}
}

func TestHandleCmdAppDiagGetReport_EmitsFullReport(t *testing.T) {
	t.Parallel()

	l := lifecycle.New()
	l.SetRestartCount(3)

	logs := &stubLogProvider{entries: []string{"ERR one", "WARN two"}}

	handler := app.HandleCmdAppDiagGetReport(testDiagService, l, logs)

	reply := handler.Handle(newDiagRequest(t))

	require.NotNil(t, reply)
	require.NotNil(t, reply.Payload)

	assert.Equal(t, app.EvtAppDiagReport, reply.Payload.Interface)
	assert.Equal(t, "object", string(reply.Payload.ValueType))

	// The handler builds the payload in-memory, so the struct lives in
	// Payload.Value (no MQTT round-trip has serialized it into ValueObj yet).
	got, ok := reply.Payload.Value.(*app.DiagReport)
	require.True(t, ok, "expected *app.DiagReport payload, got %T", reply.Payload.Value)

	assert.GreaterOrEqual(t, got.Uptime, 0)
	assert.Equal(t, 3, got.RestartsCount)
	assert.Equal(t, []string{"ERR one", "WARN two"}, got.Errors)
}

func TestHandleCmdAppDiagGetReport_PropagatesLogProviderError(t *testing.T) {
	t.Parallel()

	l := lifecycle.New()
	boom := errors.New("log read boom")
	logs := &stubLogProvider{err: boom}

	handler := app.HandleCmdAppDiagGetReport(testDiagService, l, logs)

	reply := handler.Handle(newDiagRequest(t))

	// On error the default handler emits an evt.error.report reply rather than
	// the diag report.
	require.NotNil(t, reply)
	require.NotNil(t, reply.Payload)
	assert.NotEqual(t, app.EvtAppDiagReport, reply.Payload.Interface, "must not emit diag report on error")
}
