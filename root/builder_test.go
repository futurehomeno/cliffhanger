package root_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/discovery"
	"github.com/futurehomeno/cliffhanger/lifecycle"
	"github.com/futurehomeno/cliffhanger/root"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
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
				WithVersion("test").
				WithMQTT(mqtt).
				WithServiceDiscovery(&discovery.Resource{}).
				WithTopicSubscription("test").
				WithRouting(&router.Routing{}).
				WithRouterOptions(router.WithAsyncProcessing(3)).
				WithTask(&task.Task{}),
			wantErr: false,
		},
		{
			name: "If the app version is missing, we don't raise an error",
			builder: root.NewCoreAppBuilder().
				WithMQTT(mqtt).
				WithServiceDiscovery(&discovery.Resource{}).
				WithTopicSubscription("test").
				WithRouting(&router.Routing{}).
				WithRouterOptions(router.WithAsyncProcessing(3)).
				WithTask(&task.Task{}),
			wantErr: false,
		},
		{
			name: "Missing mqtt client",
			builder: root.NewCoreAppBuilder().
				WithServiceDiscovery(&discovery.Resource{}),
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
				WithServiceDiscovery(&discovery.Resource{}).
				WithLifecycle(lifecycle.New()),
			wantErr: true,
		},
		{
			name: "Edge without lifecycle service",
			builder: root.NewEdgeAppBuilder().
				WithMQTT(mqtt).
				WithServiceDiscovery(&discovery.Resource{}),
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
