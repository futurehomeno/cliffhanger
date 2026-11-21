package root_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/futurehomeno/fimpgo"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/cliffhanger/config"
	"github.com/futurehomeno/cliffhanger/debug"
	"github.com/futurehomeno/cliffhanger/discovery"
	"github.com/futurehomeno/cliffhanger/lifecycle"
	"github.com/futurehomeno/cliffhanger/root"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

func TestBuilder_Build(t *testing.T) {
	t.Parallel()

	mqtt := suite.DefaultMQTT("root_app_builder", "", "", "")

	tcs := []struct {
		name    string
		builder *root.Builder
		wantErr bool
	}{
		{
			name: "Build core without errors",
			builder: root.NewCoreAppBuilder().
				WithMQTT(mqtt).
				WithServiceDiscovery("test_app", discovery.ResourceTypeApp, "test_app", "1", "1.0.0").
				WithTopicSubscription("test").
				WithRouting(&router.Routing{}).
				WithRouterOptions(router.WithAsyncProcessing(3)),
			wantErr: false,
		},
		{
			name: "Missing app version raises an error",
			builder: root.NewCoreAppBuilder().
				WithMQTT(mqtt).
				WithServiceDiscovery("test_app", discovery.ResourceTypeApp, "test_app", "1", "").
				WithTopicSubscription("test").
				WithRouting(&router.Routing{}).
				WithRouterOptions(router.WithAsyncProcessing(3)),
			wantErr: true,
		},
		{
			name: "Missing mqtt client",
			builder: root.NewCoreAppBuilder().
				WithServiceDiscovery("test_app", discovery.ResourceTypeApp, "test_app", "1", "1.0.0"),
			wantErr: true,
		},
		{
			name: "Missing service discovery",
			builder: root.NewCoreAppBuilder().
				WithMQTT(mqtt),
			wantErr: true,
		},
		{
			name: "Core with lifecycle service",
			builder: root.NewCoreAppBuilder().
				WithMQTT(mqtt).
				WithServiceDiscovery("test_app", discovery.ResourceTypeApp, "test_app", "1", "1.0.0").
				WithLifecycle(lifecycle.New(nil)),
			wantErr: true,
		},
		{
			name: "Edge without lifecycle service",
			builder: root.NewEdgeAppBuilder().
				WithMQTT(mqtt).
				WithServiceDiscovery("test_app", discovery.ResourceTypeApp, "test_app", "1", "1.0.0"),
			wantErr: true,
		},
	}

	for _, tc := range tcs {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := tc.builder.Build()

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBuilder_LaterAppendsKeepBuiltDiscovery(t *testing.T) { //nolint:paralleltest
	const otherTopic = "pt:j1/mt:cmd/rt:app/rn:other/ad:1"

	other := router.NewRouting(
		router.NewMessageHandler(router.MessageProcessorFn(func(*fimpgo.Message) (*fimpgo.FimpMessage, error) { return nil, nil })),
		router.ForTopic(otherTopic),
	)

	s := suite.Suite{
		Cases: []*suite.Case{
			{
				Name: "discovery answers after the builder gets another route and topic",
				Setup: suite.ServiceSetup(func(t *testing.T) (suite.Service, []suite.Mock) {
					t.Helper()

					b := root.NewCoreAppBuilder().
						WithMQTT(suite.DefaultMQTT("root_builder_aliasing", "", "", "")).
						WithServiceDiscovery("test_app", discovery.ResourceTypeApp, "test_app", "1", "1.0.0")

					for range 3 {
						b.WithRouting(other).WithTopicSubscription(otherTopic)
					}

					app, err := b.Build()
					require.NoError(t, err)

					b.WithRouting(other).WithTopicSubscription(otherTopic)

					return app, nil
				}),
				Nodes: []*suite.Node{
					{
						Command:      suite.NullMessage(discovery.Topic, discovery.CmdDiscoveryRequest, discovery.Service),
						Expectations: []*suite.Expectation{suite.ExpectMessage("pt:j1/mt:evt/rt:discovery", discovery.EvtDiscoveryReport, discovery.Service)},
					},
				},
			},
		},
	}

	s.Run(t)
}

func TestBuilder_BuildWritesCrashLogMarker(t *testing.T) { //nolint:paralleltest
	dir := t.TempDir()
	cfg := &config.Default{LogLevel: "info", LogFormat: "text", LogFile: filepath.Join(dir, "app.log")}
	store := config.NewDefaultStore(func() *config.Default { return cfg }, func() error { return nil })

	savedOut := logrus.StandardLogger().Out
	t.Cleanup(func() { logrus.SetOutput(savedOut) })

	require.NoError(t, debug.InitializeLogger(store))

	_, err := root.NewCoreAppBuilder().
		WithMQTT(suite.DefaultMQTT("root_builder_crash_log", "", "", "")).
		WithServiceDiscovery("test_app", discovery.ResourceTypeApp, "test_app", "1", "1.0.0").
		Build()
	require.NoError(t, err)

	b, err := os.ReadFile(filepath.Join(dir, "crash.log")) //nolint:gosec
	require.NoError(t, err)
	assert.Contains(t, string(b), "--- Start test_app v1.0.0 ")
}
