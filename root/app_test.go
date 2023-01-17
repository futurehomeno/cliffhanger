package root_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/discovery"
	"github.com/futurehomeno/cliffhanger/lifecycle"
	"github.com/futurehomeno/cliffhanger/root"
	mockedroot "github.com/futurehomeno/cliffhanger/test/mocks/root"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

func TestApp_Run(t *testing.T) { //nolint:paralleltest
	tcs := []struct {
		name         string
		service      *mockedroot.Service
		resetter     *mockedroot.Resetter
		triggerReset bool
		triggerStop  bool
		wantErr      bool
	}{
		{
			name:        "Start and stop without errors",
			service:     mockedroot.NewService(t).MockStart(nil).MockStop(nil),
			resetter:    mockedroot.NewResetter(t),
			triggerStop: true,
			wantErr:     false,
		},
		{
			name:        "Start with errors",
			service:     mockedroot.NewService(t).MockStart(errors.New("test")),
			resetter:    mockedroot.NewResetter(t),
			triggerStop: false,
			wantErr:     true,
		},
		{
			name:        "Start and stop with errors",
			service:     mockedroot.NewService(t).MockStart(nil).MockStop(errors.New("test")),
			resetter:    mockedroot.NewResetter(t),
			triggerStop: true,
			wantErr:     true,
		},
		{
			name:         "Start and reset without errors",
			service:      mockedroot.NewService(t).MockStart(nil).MockStop(nil),
			resetter:     mockedroot.NewResetter(t).MockReset(nil),
			triggerReset: true,
			wantErr:      false,
		},
		{
			name:         "Start and reset without errors",
			service:      mockedroot.NewService(t).MockStart(nil).MockStop(nil),
			resetter:     mockedroot.NewResetter(t).MockReset(nil),
			triggerReset: true,
			wantErr:      false,
		},
		{
			name:         "Start and reset with errors",
			service:      mockedroot.NewService(t).MockStart(nil).MockStop(nil),
			resetter:     mockedroot.NewResetter(t).MockReset(errors.New("test")),
			triggerReset: true,
			wantErr:      true,
		},
		{
			name:         "Start and reset without errors with stop error",
			service:      mockedroot.NewService(t).MockStart(nil).MockStop(errors.New("test")),
			resetter:     mockedroot.NewResetter(t).MockReset(nil),
			triggerReset: true,
			wantErr:      false,
		},
	}

	for _, tc := range tcs { //nolint:paralleltest
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			mqtt := suite.DefaultMQTT("root_app", "", "", "")

			defer mqtt.Stop()

			app, err := root.NewEdgeAppBuilder().
				WithMQTT(mqtt).
				WithLifecycle(lifecycle.New()).
				WithServiceDiscovery(&discovery.Resource{}).
				WithServices(tc.service).
				WithResetter(tc.resetter).
				Build()

			assert.NoError(t, err)

			go func() {
				time.Sleep(100 * time.Millisecond)
				if tc.triggerStop {
					_ = app.Stop()
				}

				if tc.triggerReset {
					_ = app.Reset()
				}
			}()

			err = app.Run()
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			tc.resetter.AssertExpectations(t)
			tc.service.AssertExpectations(t)
		})
	}
}

func TestApp_Reset(t *testing.T) { //nolint:paralleltest
	tc := suite.Suite{
		Cases: []*suite.Case{
			{
				Name: "Receive and handle reset command",
				Setup: suite.ServiceSetup(func(t *testing.T) (service suite.Service, mocks []suite.Mock) {
					t.Helper()

					mqtt := suite.DefaultMQTT("root_app", "", "", "")

					resetter := mockedroot.NewResetter(t).MockReset(nil)

					app, err := root.NewCoreAppBuilder().
						WithMQTT(mqtt).
						WithServiceDiscovery(&discovery.Resource{}).
						WithResetter(resetter).
						Build()

					assert.NoError(t, err)

					return app, []suite.Mock{resetter}
				}),
				Nodes: []*suite.Node{
					{
						Command: suite.NullMessage(root.GatewayEvtTopic, root.EvtGatewayFactoryReset, "gateway"),
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()

								time.Sleep(100 * time.Millisecond)
							},
						},
					},
				},
			},
		},
	}

	tc.Run(t)
}
