package root_test

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/cliffhanger/discovery"
	"github.com/futurehomeno/cliffhanger/root"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/telemetry"
	mockedroot "github.com/futurehomeno/cliffhanger/test/mocks/root"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

func coreBuilder(clientID string) *root.Builder {
	return root.NewCoreAppBuilder().
		WithMQTT(suite.DefaultMQTT(clientID, "", "", "")).
		WithServiceDiscovery("test_app", discovery.ResourceTypeApp, "test_app", "1", "1.0.0")
}

func TestApp_ResetStopsAtFirstFailingResetter(t *testing.T) { //nolint:paralleltest
	resetErr := errors.New("wipe failed")

	app, err := coreBuilder("root_app_reset_fail").
		WithResetter(mockedroot.NewResetter(t).MockReset(resetErr), mockedroot.NewResetter(t)).
		Build()
	require.NoError(t, err)

	require.NoError(t, app.Start())

	go func() {
		time.Sleep(100 * time.Millisecond)

		_ = app.Reset()
	}()

	assert.ErrorIs(t, app.Wait(), resetErr, "a pending Wait returns the reset result")
	assert.ErrorIs(t, app.Reset(), resetErr, "Reset of a stopped app still runs the resetters")
}

func TestApp_StopIsBestEffortAndReversed(t *testing.T) { //nolint:paralleltest
	var (
		mu    sync.Mutex
		order []string
	)

	record := func(name string) func() {
		return func() {
			mu.Lock()
			defer mu.Unlock()

			order = append(order, name)
		}
	}

	first := mockedroot.NewService(t).MockStart(nil)
	first.EXPECT().Stop().Run(record("first")).Return(nil).Once()

	second := mockedroot.NewService(t).MockStart(nil)
	second.EXPECT().Stop().Run(record("second")).Return(errors.New("second stop")).Once()

	app, err := coreBuilder("root_app_best_effort_stop").WithServices(first, second).Build()
	require.NoError(t, err)

	require.NoError(t, app.Start())

	err = app.Stop()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "stop service[1] err: second stop")
	assert.Equal(t, []string{"second", "first"}, order, "services stop in reverse and a failure does not skip the rest")

	assert.NoError(t, app.Stop(), "the app is marked as stopped despite the failure")
	assert.NoError(t, app.Wait())
}

// failingTelemetry embeds the interface to satisfy its unexported methods; only Start and Stop are called here.
type failingTelemetry struct {
	telemetry.Telemetry

	stopped bool
}

func (f *failingTelemetry) Start() error { return errors.New("no cloud") }

func (f *failingTelemetry) Stop() error {
	f.stopped = true

	return nil
}

func TestApp_TelemetryStartFailureIsNonFatal(t *testing.T) { //nolint:paralleltest
	tel := &failingTelemetry{}

	service := mockedroot.NewService(t).MockStart(nil).MockStop(nil)

	app, err := coreBuilder("root_app_telemetry_fail").WithTelemetry(tel).WithServices(service).Build()
	require.NoError(t, err)

	require.NoError(t, app.Start())
	require.NoError(t, app.Stop())
	assert.True(t, tel.stopped)
}

func TestApp_DiscoveryIsAlwaysWired(t *testing.T) { //nolint:paralleltest
	s := suite.Suite{
		Cases: []*suite.Case{
			{
				Name: "discovery without any routing",
				Setup: suite.ServiceSetup(func(t *testing.T) (suite.Service, []suite.Mock) {
					t.Helper()

					app, err := coreBuilder("root_app_discovery").Build()
					require.NoError(t, err)

					return app, nil
				}),
				Nodes: []*suite.Node{
					{
						Name:    "discovery request is answered",
						Command: suite.NullMessage(discovery.Topic, discovery.CmdDiscoveryRequest, discovery.Service),
						Expectations: []*suite.Expectation{
							suite.NewExpectation(router.ForType(discovery.EvtDiscoveryReport), router.ForService(discovery.Service)).ExactlyOnce(),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}
