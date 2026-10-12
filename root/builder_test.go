package root_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

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
