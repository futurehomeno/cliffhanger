package bootstrap_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/cliffhanger/bootstrap"
	"github.com/futurehomeno/cliffhanger/config"
	"github.com/futurehomeno/cliffhanger/debug"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
	"github.com/futurehomeno/cliffhanger/telemetry"
	"github.com/futurehomeno/cliffhanger/telemetry/types"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

func TestDefaultRoute_BundlesConfigDebugTelemetry(t *testing.T) { //nolint:paralleltest
	dir := t.TempDir()

	cfg := &config.Default{
		LogLevel:  "info",
		LogFormat: "text",
		LogFile:   filepath.Join(dir, "app.log"),
		Telemetry: &types.TelemetryConfig{Enabled: true, EnabledAt: time.Now(), Validity: time.Hour},
	}
	store := config.NewDefaultStore(
		func() *config.Default { return cfg },
		func() error { return nil },
	)

	saved := logrus.GetLevel()
	t.Cleanup(func() { logrus.SetLevel(saved) })

	require.NoError(t, debug.InitializeLogger(store))

	mqtt := suite.DefaultMQTT("cliff_bootstrap_test", "", "", "")
	require.NoError(t, mqtt.Start(2*time.Second))
	t.Cleanup(mqtt.Stop)

	tel, err := telemetry.New(mqtt, "boot_svc", store)
	require.NoError(t, err)
	t.Cleanup(func() {
		if stop, ok := tel.(interface{ Stop() }); ok {
			stop.Stop()
		}
	})

	routes := bootstrap.DefaultRoute("boot_svc", func() any { return store.Default() }, tel)

	// 1 config-report + 8 debug log routes + 6 telemetry routes = 15.
	assert.Len(t, routes, 15)
}

func TestDefaultRoute_DispatchesConfigDebugAndTelemetry(t *testing.T) { //nolint:paralleltest
	dir := t.TempDir()

	cfg := &config.Default{
		LogLevel:  "info",
		LogFormat: "text",
		LogFile:   filepath.Join(dir, "app.log"),
		Telemetry: &types.TelemetryConfig{Enabled: true, EnabledAt: time.Now(), Validity: time.Hour},
	}
	store := config.NewDefaultStore(
		func() *config.Default { return cfg },
		func() error { return nil },
	)

	saved := logrus.GetLevel()
	t.Cleanup(func() { logrus.SetLevel(saved) })

	require.NoError(t, debug.InitializeLogger(store))

	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name: "config + debug + telemetry routes wire up",
				Setup: suite.BaseSetup(func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
					t.Helper()

					tel, err := telemetry.New(mqtt, "boot_svc", store)
					require.NoError(t, err)
					t.Cleanup(func() {
						if stop, ok := tel.(interface{ Stop() }); ok {
							stop.Stop()
						}
					})

					return bootstrap.DefaultRoute("boot_svc", func() any { return store.Default() }, tel), nil, nil
				}),
				Nodes: []*suite.Node{
					{
						// debug log route reachable through DefaultRoute
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", debug.CmdLogGetLevel, "boot_svc"),
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:app/rn:test/ad:1", debug.EvtLogLevelReport, "boot_svc", "info"),
						},
					},
					{
						// telemetry route reachable through DefaultRoute
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.config.get_telemetry_enabled", "boot_svc"),
						Expectations: []*suite.Expectation{
							suite.ExpectBool("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.config.telemetry_enabled_report", "boot_svc", true),
						},
					},
					{
						// config-report route reachable through DefaultRoute
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.config.get_report", "boot_svc"),
						Expectations: []*suite.Expectation{
							suite.ExpectMessage("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.config.report", "boot_svc"),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func TestDefaultRoute_CustomConfigGetterPayload(t *testing.T) { //nolint:paralleltest
	dir := t.TempDir()

	cfg := &config.Default{
		LogLevel:  "info",
		LogFormat: "text",
		LogFile:   filepath.Join(dir, "app.log"),
		Telemetry: &types.TelemetryConfig{Enabled: true, EnabledAt: time.Now(), Validity: time.Hour},
	}
	store := config.NewDefaultStore(
		func() *config.Default { return cfg },
		func() error { return nil },
	)

	saved := logrus.GetLevel()
	t.Cleanup(func() { logrus.SetLevel(saved) })

	require.NoError(t, debug.InitializeLogger(store))

	type customPayload struct {
		Tag string `json:"tag"`
	}
	custom := customPayload{Tag: "custom"}

	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name: "custom config getter payload is reported",
				Setup: suite.BaseSetup(func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
					t.Helper()

					tel, err := telemetry.New(mqtt, "boot_svc", store)
					require.NoError(t, err)
					t.Cleanup(func() {
						if stop, ok := tel.(interface{ Stop() }); ok {
							stop.Stop()
						}
					})

					return bootstrap.DefaultRoute("boot_svc", func() any { return custom }, tel), nil, nil
				}),
				Nodes: []*suite.Node{
					{
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.config.get_report", "boot_svc"),
						Expectations: []*suite.Expectation{
							suite.ExpectObject("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.config.report", "boot_svc", &customPayload{Tag: "custom"}),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}
