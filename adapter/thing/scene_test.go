package thing_test

import (
	"errors"
	"testing"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/scenectrl"
	"github.com/futurehomeno/cliffhanger/adapter/thing"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
	adapterhelper "github.com/futurehomeno/cliffhanger/test/helper/adapter"
	mockedadapter "github.com/futurehomeno/cliffhanger/test/mocks/adapter"
	mockedscenectrl "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/scenectrl"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

const (
	sceneNone        = "none"
	sceneColorloop   = "colorloop"
	sceneUnsupported = "movietime"
)

func TestRouteScene(t *testing.T) { //nolint:paralleltest
	sceneReportColorloop := scenectrl.SceneReport{
		Scene:     sceneColorloop,
		Timestamp: time.Now(),
	}

	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "successful set scene",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeScene(
					mockedscenectrl.NewController(t).
						MockSetSceneCtrlScene(sceneColorloop, nil, true).
						MockSceneCtrlSceneReport(sceneReportColorloop, nil, true),
				),
				Nodes: []*suite.Node{
					{
						Name:    "set scene",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:scene_ctrl/ad:2", "cmd.scene.set", "scene_ctrl", sceneColorloop),
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:scene_ctrl/ad:2", "evt.scene.report", "scene_ctrl", sceneColorloop),
						},
					},
				},
			},
			{
				Name:     "successful get scene",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeScene(
					mockedscenectrl.NewController(t).
						MockSceneCtrlSceneReport(sceneReportColorloop, nil, true),
				),
				Nodes: []*suite.Node{
					{
						Name:    "get scene",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:scene_ctrl/ad:2", "cmd.scene.get_report", "scene_ctrl"),
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:scene_ctrl/ad:2", "evt.scene.report", "scene_ctrl", sceneColorloop),
						},
					},
				},
			},
			{
				Name:     "failed set scene - setting error",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeScene(
					mockedscenectrl.NewController(t).
						MockSetSceneCtrlScene(sceneColorloop, errors.New("error"), true),
				),
				Nodes: []*suite.Node{
					{
						Name:    "controller error",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:scene_ctrl/ad:2", "cmd.scene.set", "scene_ctrl", sceneColorloop),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:scene_ctrl/ad:2", "scene_ctrl"),
						},
					},
					{
						Name:    "invalid value type",
						Command: suite.IntMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:scene_ctrl/ad:2", "cmd.scene.set", "scene_ctrl", 1),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:scene_ctrl/ad:2", "scene_ctrl"),
						},
					},
					{
						Name:    "wrong address",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:scene_ctrl/ad:3", "cmd.scene.set", "scene_ctrl", sceneColorloop),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:scene_ctrl/ad:3", "scene_ctrl"),
						},
					},
					{
						Name:    "unsupported scene",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:scene_ctrl/ad:2", "cmd.scene.set", "scene_ctrl", sceneUnsupported),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:scene_ctrl/ad:2", "scene_ctrl"),
						},
					},
				},
			},
			{
				Name:     "failed set scene - report error",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeScene(
					mockedscenectrl.NewController(t).
						MockSetSceneCtrlScene(sceneColorloop, nil, true).
						MockSceneCtrlSceneReport(sceneReportColorloop, errors.New("error"), true),
				),
				Nodes: []*suite.Node{
					{
						Name:    "report error",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:scene_ctrl/ad:2", "cmd.scene.set", "scene_ctrl", sceneColorloop),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:scene_ctrl/ad:2", "scene_ctrl"),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func TestTaskScene(t *testing.T) { //nolint:paralleltest
	sceneReport1 := scenectrl.SceneReport{
		Scene:     sceneColorloop,
		Timestamp: time.Date(2022, time.January, 1, 1, 1, 1, 1, time.UTC),
	}

	sceneReport2 := scenectrl.SceneReport{
		Scene:     sceneColorloop,
		Timestamp: time.Date(2022, time.January, 2, 2, 2, 2, 2, time.UTC),
	}

	sceneReport3 := scenectrl.SceneReport{
		Scene:     sceneNone,
		Timestamp: time.Date(2022, time.January, 2, 2, 2, 2, 2, time.UTC),
	}

	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "SceneCtrl Tasks",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: taskScene(
					mockedscenectrl.NewController(t).
						MockSceneCtrlSceneReport(sceneReport1, nil, true).
						MockSceneCtrlSceneReport(sceneReport2, nil, true).
						MockSceneCtrlSceneReport(sceneReport3, nil, true).
						MockSceneCtrlSceneReport(scenectrl.SceneReport{}, errors.New("task error"), false),
					100*time.Millisecond,
				),
				Nodes: []*suite.Node{
					{
						Name: "Tasks",
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:scene_ctrl/ad:2", "evt.scene.report", "scene_ctrl", sceneColorloop),
							suite.ExpectString("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:scene_ctrl/ad:2", "evt.scene.report", "scene_ctrl", sceneNone),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func routeScene(
	sceneControlController *mockedscenectrl.Controller,
) suite.BaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
		t.Helper()

		routing, _, mocks := setupScene(t, mqtt, sceneControlController, 0)

		return routing, nil, mocks
	}
}

func taskScene(
	sceneCtrlController *mockedscenectrl.Controller,
	interval time.Duration,
) suite.BaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
		t.Helper()

		routing, tasks, mocks := setupScene(t, mqtt, sceneCtrlController, interval)

		return routing, tasks, mocks
	}
}

func setupScene(
	t *testing.T,
	mqtt *fimpgo.MqttTransport,
	sceneCtrlController *mockedscenectrl.Controller,
	interval time.Duration,
) ([]*router.Routing, []*task.Task, []suite.Mock) {
	t.Helper()

	mocks := []suite.Mock{sceneCtrlController}

	cfg := &thing.SceneConfig{
		ThingConfig: &adapter.ThingConfig{
			InclusionReport: &fimptype.ThingInclusionReport{
				Address: "2",
			},
			Connector: mockedadapter.NewConnector(t),
		},
		SceneCtrlConfig: &scenectrl.Config{
			Specification: scenectrl.Specification(
				"test_adapter",
				"1",
				"2",
				[]string{sceneNone, sceneColorloop},
				nil,
			),
			Controller: sceneCtrlController,
		},
	}

	seed := &adapter.ThingSeed{ID: "B", CustomAddress: "2"}

	factory := adapterhelper.FactoryHelper(func(adapter adapter.Adapter, publisher adapter.Publisher, thingState adapter.ThingState) (adapter.Thing, error) {
		return thing.NewScene(publisher, thingState, cfg), nil
	})

	ad := adapterhelper.PrepareSeededAdapter(t, "../../testdata/adapter/test_adapter", mqtt, factory, adapter.ThingSeeds{seed})

	return thing.RouteScene(ad), thing.TaskScene(ad, interval), mocks
}
