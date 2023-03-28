package colorctrl_test

import (
	"errors"
	"testing"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/colorctrl"
	"github.com/futurehomeno/cliffhanger/event"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
	mockedadapter "github.com/futurehomeno/cliffhanger/test/mocks/adapter"
	mockedcolorctrl "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/colorctrl"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

func TestRouteColorCtrl(t *testing.T) { //nolint:paralleltest
	validColor := map[string]int64{"red": 255, "green": 55, "blue": 100}

	invalidColor := map[string]float64{"red": 255.0, "green": 55.0, "blue": 100.0}

	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name: "successful set color",
				Setup: routeColorCtrl(
					mockedcolorctrl.NewController(t).
						MockSetColorCtrlColor(validColor, nil, true).
						MockColorCtrlColorReport(validColor, nil, true),
				),
				Nodes: []*suite.Node{
					{
						Name:    "set color",
						Command: suite.IntMapMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:color_ctrl/ad:2", "cmd.color.set", "color_ctrl", validColor),
						Expectations: []*suite.Expectation{
							suite.ExpectIntMap("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:color_ctrl/ad:2", "evt.color.report", "color_ctrl", validColor),
						},
					},
				},
			},
			{
				Name: "successful get color",
				Setup: routeColorCtrl(
					mockedcolorctrl.NewController(t).
						MockColorCtrlColorReport(validColor, nil, true),
				),
				Nodes: []*suite.Node{
					{
						Name:    "get color",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:color_ctrl/ad:2", "cmd.color.get_report", "color_ctrl"),
						Expectations: []*suite.Expectation{
							suite.ExpectIntMap("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:color_ctrl/ad:2", "evt.color.report", "color_ctrl", validColor),
						},
					},
				},
			},
			{
				Name: "failed set level - setting error",
				Setup: routeColorCtrl(
					mockedcolorctrl.NewController(t).
						MockSetColorCtrlColor(validColor, errors.New("error"), true),
				),
				Nodes: []*suite.Node{
					{
						Name:    "controller error",
						Command: suite.IntMapMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:color_ctrl/ad:2", "cmd.color.set", "color_ctrl", validColor),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:color_ctrl/ad:2", "color_ctrl"),
						},
					},
					{
						Name:    "wrong colorValue type",
						Command: suite.FloatMapMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:color_ctrl/ad:2", "cmd.color.set", "color_ctrl", invalidColor),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:color_ctrl/ad:2", "color_ctrl"),
						},
					},
					{
						Name:    "wrong address",
						Command: suite.IntMapMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:color_ctrl/ad:3", "cmd.color.set", "color_ctrl", validColor),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:color_ctrl/ad:3", "color_ctrl"),
						},
					},
				},
			},
			{
				Name: "failed set level - report error",
				Setup: routeColorCtrl(
					mockedcolorctrl.NewController(t).
						MockSetColorCtrlColor(validColor, nil, true).
						MockColorCtrlColorReport(validColor, errors.New("error"), true),
				),
				Nodes: []*suite.Node{
					{
						Name:    "report error",
						Command: suite.IntMapMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:color_ctrl/ad:2", "cmd.color.set", "color_ctrl", validColor),
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

func TestTaskColorCtrl(t *testing.T) { //nolint:paralleltest
	val1 := make(map[string]int64)
	val1["red"] = 255
	val1["green"] = 55
	val1["blue"] = 100

	val2 := make(map[string]int64)
	val2["red"] = 55
	val2["green"] = 155
	val2["blue"] = 255

	val3 := make(map[string]int64)
	val3["red"] = 100
	val3["green"] = 200
	val3["blue"] = 0

	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name: "ColorCtrl Tasks",
				Setup: taskColorCtrl(
					mockedcolorctrl.NewController(t).
						MockColorCtrlColorReport(val1, nil, true).
						MockColorCtrlColorReport(val1, errors.New("task error"), true).
						MockColorCtrlColorReport(val2, nil, true).
						MockColorCtrlColorReport(val3, nil, true).
						MockColorCtrlColorReport(val3, nil, false),
					100*time.Millisecond,
				),
				Nodes: []*suite.Node{
					{
						Name: "Tasks",
						Expectations: []*suite.Expectation{
							suite.ExpectIntMap("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:color_ctrl/ad:2", "evt.color.report", "color_ctrl", val1),
							suite.ExpectIntMap("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:color_ctrl/ad:2", "evt.color.report", "color_ctrl", val2),
							suite.ExpectIntMap("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:color_ctrl/ad:2", "evt.color.report", "color_ctrl", val3),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func routeColorCtrl(
	colorCtrlController *mockedcolorctrl.Controller,
) suite.BaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
		t.Helper()

		routing, _, mocks := setupColorCtrl(t, mqtt, colorCtrlController, 0)

		return routing, nil, mocks
	}
}

func taskColorCtrl(
	colorCtrlController *mockedcolorctrl.Controller,
	interval time.Duration,
) suite.BaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
		t.Helper()

		routing, tasks, mocks := setupColorCtrl(t, mqtt, colorCtrlController, interval)

		return routing, tasks, mocks
	}
}

func setupColorCtrl(
	t *testing.T,
	mqtt *fimpgo.MqttTransport,
	colorCtrlController *mockedcolorctrl.Controller,
	interval time.Duration,
) ([]*router.Routing, []*task.Task, []suite.Mock) {
	t.Helper()

	mocks := []suite.Mock{colorCtrlController}

	cfg := &colorCtrlThingConfig{
		ThingConfig: &adapter.ThingConfig{
			InclusionReport: &fimptype.ThingInclusionReport{
				Address: "2",
			},
			Connector: mockedadapter.NewConnector(t),
		},
		ColorCtrlConfig: &colorctrl.Config{
			Specification: colorctrl.Specification(
				"test_adapter",
				"1",
				"2",
				nil,
				[]string{"red", "green", "blue"},
				map[string]int64{"min": 180, "max": 7620, "step": 60},
			),
			Controller: colorCtrlController,
		},
	}

	ad := adapter.NewAdapter(mqtt, event.NewManager(), nil, nil, "test_adapter", "1")

	colorCtrlThing := newColorCtrlThing(ad, cfg)

	ad.RegisterThing(colorCtrlThing)

	return routeColorCtrlThing(ad), taskColorCtrlThing(ad, interval), mocks
}

// colorCtrlThingConfig represents a config for testing color control service.
type colorCtrlThingConfig struct {
	ThingConfig     *adapter.ThingConfig
	ColorCtrlConfig *colorctrl.Config
}

// newColorCtrlThing creates a thing that can be used for testing color control service.
func newColorCtrlThing(
	a adapter.Adapter,
	cfg *colorCtrlThingConfig,
) adapter.Thing {
	services := []adapter.Service{
		colorctrl.NewService(a, cfg.ColorCtrlConfig),
	}

	return adapter.NewThing(a, nil, cfg.ThingConfig, services...)
}

// routeColorCtrlThing creates a thing that can be used for testing color control service.
func routeColorCtrlThing(adapter adapter.Adapter) []*router.Routing {
	return router.Combine(
		colorctrl.RouteService(adapter),
	)
}

// taskColorCtrlThing creates background tasks specific for testing color control service.
func taskColorCtrlThing(
	adapter adapter.Adapter,
	interval time.Duration,
	voter ...task.Voter,
) []*task.Task {
	return []*task.Task{
		colorctrl.TaskReporting(adapter, interval, voter...),
	}
}
