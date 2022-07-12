package color_ctrl_test

import (
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
	m := make(map[string]int64)
	m["red"] = 255
	m["green"] = 255
	m["blue"] = 255

	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name: "successful set color",

				Setup: routeColorCtrl(
					mockedcolorctrl.NewController(t).
						MockSetColorCtrlColor(m, nil, true).
						MockColorCtrlColorReport(m, nil, true),
				),
				Nodes: []*suite.Node{
					{
						Name:    "set color",
						Command: suite.IntMapMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:color_ctrl/ad:2", "cmd.color.set", "color_ctrl", m),
						Expectations: []*suite.Expectation{
							suite.ExpectIntMap("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:color_ctrl/ad:2", "evt.color.report", "color_ctrl", m),
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
