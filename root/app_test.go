package root_test

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/cliffhanger/discovery"
	"github.com/futurehomeno/cliffhanger/lifecycle"
	"github.com/futurehomeno/cliffhanger/notification"
	"github.com/futurehomeno/cliffhanger/root"
	"github.com/futurehomeno/cliffhanger/task"
	mockedroot "github.com/futurehomeno/cliffhanger/test/mocks/root"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

type fakeAppNotifier struct {
	mu    sync.Mutex
	count int
}

func (f *fakeAppNotifier) Event(*notification.Event) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.count++

	return nil
}

func (f *fakeAppNotifier) Message(string) error { return nil }

func (f *fakeAppNotifier) EventWithProps(e *notification.Event, _ map[string]string) error {
	return f.Event(e)
}

func (f *fakeAppNotifier) sent() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.count
}

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
			name:         "Start and reset without errors",
			service:      mockedroot.NewService(t).MockStart(nil).MockStop(nil),
			resetter:     mockedroot.NewResetter(t).MockReset(nil),
			triggerReset: true,
			wantErr:      false,
		},
		{
			name:        "Start and stop with errors",
			service:     mockedroot.NewService(t).MockStart(nil).MockStop(errors.New("test")),
			resetter:    mockedroot.NewResetter(t),
			triggerStop: true,
			wantErr:     true,
		},
		{
			name:        "Start with errors",
			service:     mockedroot.NewService(t).MockStart(errors.New("test")),
			resetter:    mockedroot.NewResetter(t),
			triggerStop: false,
			wantErr:     true,
		},
	}

	for _, tc := range tcs { //nolint:paralleltest
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			mqtt := suite.DefaultMQTT("root_app", "", "", "")

			app, err := root.NewEdgeAppBuilder().
				WithMQTT(mqtt).
				WithLifecycle(lifecycle.New(nil)).
				WithServiceDiscovery("test_app", discovery.ResourceTypeApp, "test_app", "1", "1.0.0").
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

// TestApp_Run_AuthLossWatcherSubscribesBeforeFirstTaskProbe guards against a startup race:
// the auth-loss watcher must subscribe before the task manager starts, so a
// WhenAppIsRunning-gated task cannot fire and lose auth before the watcher can observe it.
func TestApp_Run_AuthLossWatcherSubscribesBeforeFirstTaskProbe(t *testing.T) { //nolint:paralleltest
	mqtt := suite.DefaultMQTT("root_app_authloss_race", "", "", "")

	lc := lifecycle.New(nil)
	lc.SetAuthState(lifecycle.AuthStateAuthenticated)

	notifier := &fakeAppNotifier{}

	// Simulates a service that eagerly restores a persisted, already-authenticated session
	// before the task manager (and its WhenAppIsRunning-gated tasks) starts.
	restoringService := mockedroot.NewService(t).MockStop(nil)
	restoringService.EXPECT().Start().RunAndReturn(func() error {
		lc.SetAppHealth(lifecycle.AppHealthRunning, nil)

		return nil
	})

	probe := task.New(func() {
		lc.SetAuthState(lifecycle.AuthStateLost)
	}, 0, task.WhenAppIsRunning(lc))

	app, err := root.NewEdgeAppBuilder().
		WithMQTT(mqtt).
		WithLifecycle(lc).
		WithServiceDiscovery("test_app", discovery.ResourceTypeApp, "test_app", "1", "1.0.0").
		WithServices(restoringService).
		WithTask(probe).
		WithAuthLossNotification(notifier, &notification.Event{EventName: "test_status_offline"}).
		Build()
	require.NoError(t, err)

	go func() {
		time.Sleep(200 * time.Millisecond)
		_ = app.Stop()
	}()

	require.NoError(t, app.Run())

	assert.Equal(t, 1, notifier.sent(),
		"the already-authenticated-at-boot session lost by the first task probe must be "+
			"reported exactly once; a subscribe race would silently drop it")
}

func TestApp_Reset(t *testing.T) { //nolint:paralleltest
	tc := suite.Suite{
		Cases: []*suite.Case{
			{
				Name: "Receive and handle reset command",
				Setup: suite.ServiceSetup(func(t *testing.T) (service suite.Service, mocks []suite.Mock) {
					t.Helper()

					// A distinct client ID keeps a late auto-reconnect of the previous test's
					// client from taking over this session mid-subscribe.
					mqtt := suite.DefaultMQTT("root_app_reset", "", "", "")

					resetter := mockedroot.NewResetter(t).MockReset(nil)

					app, err := root.NewCoreAppBuilder().
						WithMQTT(mqtt).
						WithServiceDiscovery("test_app", discovery.ResourceTypeApp, "test_app", "1", "1.0.0").
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
