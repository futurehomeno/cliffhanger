package scenectrl_test

import (
	"errors"
	"testing"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/scenectrl"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
	adapterhelper "github.com/futurehomeno/cliffhanger/test/helper/adapter"
	mockedadapter "github.com/futurehomeno/cliffhanger/test/mocks/adapter"
	mockedscenectrl "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/scenectrl"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

var errTest = errors.New("test")

func TestRouteService(t *testing.T) { //nolint:paralleltest
	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "successful set scene routing",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(
					mockedscenectrl.NewController(t).
						MockSetSceneCtrlScene("scene1", nil, true).
						MockSceneCtrlSceneReport(scenectrl.SceneReport{Scene: "scene1"}, nil, true),
					[]string{"scene1", "scene2"},
				),
				Nodes: []*suite.Node{
					{
						Name:    "set scene",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:scene_ctrl/ad:2", "cmd.scene.set", "scene_ctrl", "scene1"),
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:scene_ctrl/ad:2", "evt.scene.report", "scene_ctrl", "scene1"),
						},
					},
				},
			},
			{
				Name:     "successful get report routing",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(
					mockedscenectrl.NewController(t).
						MockSceneCtrlSceneReport(scenectrl.SceneReport{Scene: "scene1"}, nil, true),
					[]string{"scene1", "scene2"},
				),
				Nodes: []*suite.Node{
					{
						Name:    "get report",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:scene_ctrl/ad:2", "cmd.scene.get_report", "scene_ctrl"),
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:scene_ctrl/ad:2", "evt.scene.report", "scene_ctrl", "scene1"),
						},
					},
				},
			},
			{
				Name:     "failed set scene routing - unsupported scene",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(
					mockedscenectrl.NewController(t),
					[]string{"scene1", "scene2"},
				),
				Nodes: []*suite.Node{
					{
						Name:    "set scene",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:scene_ctrl/ad:2", "cmd.scene.set", "scene_ctrl", "unknown"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:scene_ctrl/ad:2", "scene_ctrl"),
						},
					},
				},
			},
			{
				Name:     "failed set scene routing - controller error",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(
					mockedscenectrl.NewController(t).
						MockSetSceneCtrlScene("scene1", errTest, true),
					[]string{"scene1", "scene2"},
				),
				Nodes: []*suite.Node{
					{
						Name:    "set scene",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:scene_ctrl/ad:2", "cmd.scene.set", "scene_ctrl", "scene1"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:scene_ctrl/ad:2", "scene_ctrl"),
						},
					},
				},
			},
			{
				Name:     "failed set scene routing - report error",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(
					mockedscenectrl.NewController(t).
						MockSetSceneCtrlScene("scene1", nil, true).
						MockSceneCtrlSceneReport(scenectrl.SceneReport{}, errTest, true),
					[]string{"scene1", "scene2"},
				),
				Nodes: []*suite.Node{
					{
						Name:    "set scene",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:scene_ctrl/ad:2", "cmd.scene.set", "scene_ctrl", "scene1"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:scene_ctrl/ad:2", "scene_ctrl"),
						},
					},
				},
			},
			{
				Name:     "failed set scene routing - non-string payload",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(
					mockedscenectrl.NewController(t),
					[]string{"scene1", "scene2"},
				),
				Nodes: []*suite.Node{
					{
						Name:    "set scene",
						Command: suite.IntMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:scene_ctrl/ad:2", "cmd.scene.set", "scene_ctrl", 1),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:scene_ctrl/ad:2", "scene_ctrl"),
						},
					},
				},
			},
			{
				Name:     "failed get report routing - controller error",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(
					mockedscenectrl.NewController(t).
						MockSceneCtrlSceneReport(scenectrl.SceneReport{}, errTest, true),
					[]string{"scene1", "scene2"},
				),
				Nodes: []*suite.Node{
					{
						Name:    "get report",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:scene_ctrl/ad:2", "cmd.scene.get_report", "scene_ctrl"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:scene_ctrl/ad:2", "scene_ctrl"),
						},
					},
				},
			},
			{
				Name:     "failed routing - service not found",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(
					mockedscenectrl.NewController(t),
					[]string{"scene1", "scene2"},
				),
				Nodes: []*suite.Node{
					{
						Name:    "non-existent thing on set scene",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:scene_ctrl/ad:3", "cmd.scene.set", "scene_ctrl", "scene1"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:scene_ctrl/ad:3", "scene_ctrl"),
						},
					},
					{
						Name:    "non-existent thing on get report",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:scene_ctrl/ad:3", "cmd.scene.get_report", "scene_ctrl"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:scene_ctrl/ad:3", "scene_ctrl"),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func routeService(
	controller scenectrl.Controller,
	supportedScenes []string,
) suite.BaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
		t.Helper()

		routing, _, mocks := setupService(t, mqtt, controller, supportedScenes, 0)

		return routing, nil, mocks
	}
}

func setupService(
	t *testing.T,
	mqtt *fimpgo.MqttTransport,
	controller scenectrl.Controller,
	supportedScenes []string,
	duration time.Duration,
) ([]*router.Routing, []*task.Task, []suite.Mock) {
	t.Helper()

	mockedController, ok := controller.(suite.Mock)
	if !ok {
		t.Fatal("controller is not a mock")
	}

	mocks := []suite.Mock{mockedController}

	thingCfg := &adapter.ThingConfig{
		InclusionReport: &fimptype.ThingInclusionReport{
			Address: "2",
		},
		Connector: mockedadapter.NewDefaultConnector(t),
	}

	sceneCtrlCfg := &scenectrl.Config{
		Specification: scenectrl.Specification(
			"test_adapter",
			"1",
			"2",
			supportedScenes,
			nil,
		),
		Controller: controller,
	}

	seed := &adapter.ThingSeed{ID: "B", CustomAddress: "2"}

	factory := adapterhelper.FactoryHelper(func(a adapter.Adapter, publisher adapter.Publisher, thingState adapter.ThingState) (adapter.Thing, error) {
		return adapter.NewThing(publisher, thingState, thingCfg, scenectrl.NewService(publisher, sceneCtrlCfg)), nil
	})

	ad := adapterhelper.PrepareSeededAdapter(t, "../../testdata/adapter/test_adapter", mqtt, factory, adapter.ThingSeeds{seed})

	return scenectrl.RouteService(ad), task.Combine(scenectrl.TaskReporting(ad, duration)), mocks
}
