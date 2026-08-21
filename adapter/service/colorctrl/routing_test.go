package colorctrl_test

import (
	"errors"
	"testing"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/colorctrl"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
	adapterhelper "github.com/futurehomeno/cliffhanger/test/helper/adapter"
	mockedadapter "github.com/futurehomeno/cliffhanger/test/mocks/adapter"
	mockedcolorctrl "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/colorctrl"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

var errTest = errors.New("test")

func TestRouteService(t *testing.T) { //nolint:paralleltest
	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "successful set color routing",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(
					mockedcolorctrl.NewController(t).
						MockSetColorCtrlColor(map[string]int{"red": 255, "green": 0, "blue": 0}, nil, true).
						MockColorCtrlColorReport(map[string]int{"red": 255, "green": 0, "blue": 0}, nil, true),
				),
				Nodes: []*suite.Node{
					{
						Name:    "set color",
						Command: suite.IntMapMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:color_ctrl/ad:2", "cmd.color.set", "color_ctrl", map[string]int{"red": 255, "green": 0, "blue": 0}),
						Expectations: []*suite.Expectation{
							suite.ExpectIntMap("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:color_ctrl/ad:2", "evt.color.report", "color_ctrl", map[string]int{"red": 255, "green": 0, "blue": 0}),
						},
					},
				},
			},
			{
				Name:     "successful get report routing",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(
					mockedcolorctrl.NewController(t).
						MockColorCtrlColorReport(map[string]int{"red": 10, "green": 20, "blue": 30}, nil, true),
				),
				Nodes: []*suite.Node{
					{
						Name:    "get report",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:color_ctrl/ad:2", "cmd.color.get_report", "color_ctrl"),
						Expectations: []*suite.Expectation{
							suite.ExpectIntMap("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:color_ctrl/ad:2", "evt.color.report", "color_ctrl", map[string]int{"red": 10, "green": 20, "blue": 30}),
						},
					},
				},
			},
			{
				Name:     "failed set color routing - setter error",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(
					mockedcolorctrl.NewController(t).
						MockSetColorCtrlColor(map[string]int{"red": 255}, errTest, true),
				),
				Nodes: []*suite.Node{
					{
						Name:    "set color",
						Command: suite.IntMapMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:color_ctrl/ad:2", "cmd.color.set", "color_ctrl", map[string]int{"red": 255}),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:color_ctrl/ad:2", "color_ctrl"),
						},
					},
				},
			},
			{
				Name:     "failed set color routing - report error",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(
					mockedcolorctrl.NewController(t).
						MockSetColorCtrlColor(map[string]int{"red": 255}, nil, true).
						MockColorCtrlColorReport(nil, errTest, true),
				),
				Nodes: []*suite.Node{
					{
						Name:    "set color",
						Command: suite.IntMapMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:color_ctrl/ad:2", "cmd.color.set", "color_ctrl", map[string]int{"red": 255}),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:color_ctrl/ad:2", "color_ctrl"),
						},
					},
				},
			},
			{
				Name:     "failed get report routing - report error",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(
					mockedcolorctrl.NewController(t).
						MockColorCtrlColorReport(nil, errTest, true),
				),
				Nodes: []*suite.Node{
					{
						Name:    "get report",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:color_ctrl/ad:2", "cmd.color.get_report", "color_ctrl"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:color_ctrl/ad:2", "color_ctrl"),
						},
					},
				},
			},
			{
				Name:     "non-existent thing",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup:    routeService(mockedcolorctrl.NewController(t)),
				Nodes: []*suite.Node{
					{
						Name:    "set color on non-existent thing",
						Command: suite.IntMapMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:color_ctrl/ad:3", "cmd.color.set", "color_ctrl", map[string]int{"red": 255}),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:color_ctrl/ad:3", "color_ctrl"),
						},
					},
					{
						Name:    "get report on non-existent thing",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:color_ctrl/ad:3", "cmd.color.get_report", "color_ctrl"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:color_ctrl/ad:3", "color_ctrl"),
						},
					},
				},
			},
			{
				Name:     "malformed payload on set color",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup:    routeService(mockedcolorctrl.NewController(t)),
				Nodes: []*suite.Node{
					{
						Name:    "set color with wrong payload type",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:color_ctrl/ad:2", "cmd.color.set", "color_ctrl", "red"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:color_ctrl/ad:2", "color_ctrl"),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func routeService(controller colorctrl.Controller) suite.BaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
		t.Helper()

		routing, _, mocks := setupService(t, mqtt, controller, 0)

		return routing, nil, mocks
	}
}

func setupService(
	t *testing.T,
	mqtt *fimpgo.MqttTransport,
	controller colorctrl.Controller,
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

	colorCtrlCfg := &colorctrl.Config{
		Specification: colorctrl.Specification(
			"test_adapter",
			"1",
			"2",
			nil,
			[]string{"red", "green", "blue"},
			nil,
		),
		Controller: controller,
	}

	seed := &adapter.ThingSeed{ID: "B", CustomAddress: "2"}

	factory := adapterhelper.FactoryHelper(func(a adapter.Adapter, publisher adapter.Publisher, thingState adapter.ThingState) (adapter.Thing, error) {
		return adapter.NewThing(publisher, thingState, thingCfg, colorctrl.NewService(publisher, colorCtrlCfg)), nil
	})

	ad := adapterhelper.PrepareSeededAdapter(t, "../../testdata/adapter/test_adapter", mqtt, factory, adapter.ThingSeeds{seed})

	return colorctrl.RouteService(ad), task.Combine(colorctrl.TaskReporting(ad, duration)), mocks
}
