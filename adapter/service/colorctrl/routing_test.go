package color_ctrl_test

import (
	"errors"
	"testing"
	"time"

	"github.com/futurehomeno/cliffhanger/adapter"
	color_ctrl "github.com/futurehomeno/cliffhanger/adapter/service/colorctrl"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
	mockedcolorctrl "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/colorctrl"
	"github.com/futurehomeno/cliffhanger/test/suite"
	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
)

func TestRouteColorCtrl(t *testing.T) {
	val := make(map[string]int64)
	val["red"] = 255
	val["green"] = 55
	val["blue"] = 100

	wrongValueType := make(map[string]float64)
	wrongValueType["red"] = 255.0
	wrongValueType["green"] = 55.0
	wrongValueType["blue"] = 100.0

	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name: "successful set color",

				Setup: routeColorCtrl(
					mockedcolorctrl.NewController(t).
						MockSetColorCtrlColor(val, nil, true).
						MockColorCtrlColorReport(val, nil, true),
				),
				Nodes: []*suite.Node{
					{
						Name:    "set color",
						Command: suite.IntMapMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:color_ctrl/ad:2", "cmd.color.set", "color_ctrl", val),
						Expectations: []*suite.Expectation{
							suite.ExpectIntMap("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:color_ctrl/ad:2", "evt.color.report", "color_ctrl", val),
						},
					},
				},
			},
			{
				Name: "successful get color",

				Setup: routeColorCtrl(
					mockedcolorctrl.NewController(t).
						MockColorCtrlColorReport(val, nil, true),
				),
				Nodes: []*suite.Node{
					{
						Name:    "get color",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:color_ctrl/ad:2", "cmd.color.get_report", "color_ctrl"),
						Expectations: []*suite.Expectation{
							suite.ExpectIntMap("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:color_ctrl/ad:2", "evt.color.report", "color_ctrl", val),
						},
					},
				},
			},
			{
				Name: "failed set level - setting error",

				Setup: routeColorCtrl(
					mockedcolorctrl.NewController(t).
						MockSetColorCtrlColor(val, errors.New("error"), false),
				),
				Nodes: []*suite.Node{
					{
						Name:    "setting error",
						Command: suite.IntMapMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:color_ctrl/ad:2", "cmd.color.set", "color_ctrl", val),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:color_ctrl/ad:2", "color_ctrl"),
						},
					},
					{
						Name:    "wrong value type",
						Command: suite.FloatMapMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:color_ctrl/ad:2", "cmd.color.set", "color_ctrl", wrongValueType),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:color_ctrl/ad:2", "color_ctrl"),
						},
					},
					{
						Name:    "wrong address",
						Command: suite.IntMapMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:color_ctrl/ad:3", "cmd.color.set", "color_ctrl", val),
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
						MockSetColorCtrlColor(val, nil, true).
						MockColorCtrlColorReport(val, errors.New("error"), false),
				),
				Nodes: []*suite.Node{
					{
						Name:    "report error",
						Command: suite.IntMapMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:color_ctrl/ad:2", "cmd.color.set", "color_ctrl", val),
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

func TestTaskColorCtrl(t *testing.T) {
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

	cfg := &ColorCtrlThingConfig{
		InclusionReport: &fimptype.ThingInclusionReport{
			Address: "2",
		},
		ColorCtrlConfig: &color_ctrl.Config{
			Specification: color_ctrl.Specification(
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

	colorCtrl := newColorCtrlThing(mqtt, cfg)
	ad := adapter.NewAdapter(nil, "test_adapter", "1")
	ad.RegisterThing(colorCtrl)

	return routeColorCtrlThing(ad), taskColorCtrlThing(ad, interval), mocks
}

// ThingConfig represents a config for testing colorctrl service.
type ColorCtrlThingConfig struct {
	InclusionReport *fimptype.ThingInclusionReport
	ColorCtrlConfig *color_ctrl.Config
}

// newColorCtrlThing creates a thinng that can be used for testing colorctrl service
func newColorCtrlThing(
	mqtt *fimpgo.MqttTransport,
	cfg *ColorCtrlThingConfig,
) adapter.Thing {
	services := []adapter.Service{
		color_ctrl.NewService(mqtt, cfg.ColorCtrlConfig),
	}

	return adapter.NewThing(cfg.InclusionReport, services...)
}

// routeColorCtrlThing creates a thing that can be used for testing colorctrl service
func routeColorCtrlThing(adapter adapter.Adapter) []*router.Routing {
	return router.Combine(
		color_ctrl.RouteService(adapter),
	)
}

// taskColorCtrlThing creates background tasks specific for colorctrl service
func taskColorCtrlThing(
	adapter adapter.Adapter,
	interval time.Duration,
	voter ...task.Voter,
) []*task.Task {
	return []*task.Task{
		color_ctrl.TaskReporting(adapter, interval, voter...),
	}
}
