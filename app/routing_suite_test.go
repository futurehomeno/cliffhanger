package app_test

import (
	"errors"
	"testing"

	"github.com/futurehomeno/fimpgo"

	"github.com/futurehomeno/cliffhanger/app"
	"github.com/futurehomeno/cliffhanger/lifecycle"
	"github.com/futurehomeno/cliffhanger/manifest"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/storage"
	"github.com/futurehomeno/cliffhanger/task"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

const (
	appCmdTopic = "pt:j1/mt:cmd/rt:app/rn:" + testDiagService + "/ad:1"
	appEvtTopic = "pt:j1/mt:evt/rt:app/rn:" + testDiagService + "/ad:1"
)

func routeAppSetup(application app.App) suite.BaseSetup {
	return suite.BaseSetup(func(t *testing.T, _ *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
		t.Helper()

		routing := app.RouteApp[*handlerConfig](
			testDiagService, lifecycle.New(nil), storage.New(&handlerConfig{}, t.TempDir(), "config.json"),
			func() *handlerConfig { return &handlerConfig{} },
			router.NewMessageHandlerLocker(),
			application,
			nil,
		)

		return routing, nil, nil
	})
}

func TestRouteApp_Routing(t *testing.T) { //nolint:paralleltest
	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:  "reset success asks the UI to reload",
				Setup: routeAppSetup(&recordingApp{}),
				Nodes: []*suite.Node{
					{
						Command: suite.NullMessage(appCmdTopic, app.CmdAppReset, testDiagService),
						Expectations: []*suite.Expectation{
							suite.ExpectObject(appEvtTopic, app.EvtAppConfigActionReport, testDiagService,
								&manifest.ButtonActionResponse{Operation: app.CmdAppReset, OperationStatus: "ok", Next: "reload"}),
						},
					},
				},
			},
			{
				Name:  "reset failure reports a fixed error text",
				Setup: routeAppSetup(&recordingApp{resetErr: errors.New("boom")}),
				Nodes: []*suite.Node{
					{
						Command: suite.NullMessage(appCmdTopic, app.CmdAppReset, testDiagService),
						Expectations: []*suite.Expectation{
							suite.ExpectObject(appEvtTopic, app.EvtAppConfigActionReport, testDiagService,
								&manifest.ButtonActionResponse{
									Operation: app.CmdAppReset, OperationStatus: "error", Next: "ok", ErrorText: "failed to reset the application",
								}),
						},
					},
				},
			},
			{
				Name:  "app without reset ignores cmd.app.reset",
				Setup: routeAppSetup(stubLoginApp{}),
				Nodes: []*suite.Node{
					{
						Command: suite.NullMessage(appCmdTopic, app.CmdAppReset, testDiagService),
						Expectations: []*suite.Expectation{
							suite.ExpectMessage(appEvtTopic, app.EvtAppConfigActionReport, testDiagService).Never(),
							suite.ExpectError(appEvtTopic, testDiagService).Never(),
						},
					},
				},
			},
			{
				Name:  "command for another service is ignored",
				Setup: routeAppSetup(&recordingApp{}),
				Nodes: []*suite.Node{
					{
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:app/rn:other/ad:1", app.CmdAppGetState, "other"),
						Expectations: []*suite.Expectation{
							suite.NewExpectation().ExpectType(app.EvtAppStateReport).Never(),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}
