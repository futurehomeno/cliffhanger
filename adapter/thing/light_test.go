package thing_test

import (
	"errors"
	"testing"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/outlvlswitch"
	"github.com/futurehomeno/cliffhanger/adapter/thing"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
	mockedoutlvlswitch "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/outlvlswitch"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

func TestRouteLight(t *testing.T) { //nolint:paralleltest
	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name: "successful set level routing",
				Setup: routeLight(
					mockedoutlvlswitch.NewController(t).
						MockSetLevelSwitchLevel(99, nil, false).
						MockLevelSwitchLevelReport(99, nil, false),
				),
				Nodes: []*suite.Node{
					{
						Name:    "set level",
						Command: suite.IntMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "cmd.lvl.set", "out_lvl_switch", 99),
						Expectations: []*suite.Expectation{
							suite.ExpectInt("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "evt.lvl.report", "out_lvl_switch", 99),
						},
					},
				},
			},
			{
				Name: "successful set binary routing",
				Setup: routeLight(
					mockedoutlvlswitch.NewController(t).
						MockSetLevelSwitchBinaryState(true, nil, false).
						MockLevelSwitchBinaryReport(true, nil, false),
				),
				Nodes: []*suite.Node{
					{
						Name:    "set binary",
						Command: suite.BoolMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "cmd.binary.set", "out_lvl_switch", true),
						Expectations: []*suite.Expectation{
							suite.ExpectBool("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "evt.binary.report", "out_lvl_switch", true),
						},
					},
				},
			},
			{
				Name: "successful set level routing with duration",
				Setup: routeLight(
					mockedoutlvlswitch.NewController(t).
						MockSetLevelSwitchLevelWithDuration(99, 1, nil, false).
						MockLevelSwitchLevelReport(99, nil, false),
				),
				Nodes: []*suite.Node{
					{
						Name: "set level",
						Command: suite.NewMessageBuilder().
							IntMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "cmd.lvl.set", "out_lvl_switch", 99).
							AddProperty(outlvlswitch.Duration, "1").
							Build(),
						Expectations: []*suite.Expectation{
							suite.ExpectInt("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "evt.lvl.report", "out_lvl_switch", 99),
						},
					},
				},
			},
			{
				Name: "successful get report",
				Setup: routeLight(
					mockedoutlvlswitch.NewController(t).
						MockLevelSwitchLevelReport(99, nil, false),
				),
				Nodes: []*suite.Node{
					{
						Name: "get report",
						Command: suite.NewMessageBuilder().
							NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "cmd.lvl.get_report", "out_lvl_switch").
							Build(),
						Expectations: []*suite.Expectation{
							suite.ExpectInt("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "evt.lvl.report", "out_lvl_switch", 99),
						},
					},
				},
			},
			{
				Name: "failed set level - setting error",
				Setup: routeLight(
					mockedoutlvlswitch.NewController(t).
						MockSetLevelSwitchLevel(99, errors.New("setting error"), false),
				),
				Nodes: []*suite.Node{
					{
						Name:    "set level",
						Command: suite.IntMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "cmd.lvl.set", "out_lvl_switch", 99),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "out_lvl_switch"),
						},
					},
					{
						Name:    "wrong value type",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "cmd.lvl.set", "out_lvl_switch", "99"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "out_lvl_switch"),
						},
					},
					{
						Name:    "wrong address",
						Command: suite.IntMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:3", "cmd.lvl.set", "out_lvl_switch", 99),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:3", "out_lvl_switch"),
						},
					},
				},
			},
			{
				Name: "failed set level with duration - setting error",
				Setup: routeLight(
					mockedoutlvlswitch.NewController(t).
						MockSetLevelSwitchLevelWithDuration(99, 1, errors.New("setting error"), false),
				),
				Nodes: []*suite.Node{
					{
						Name: "set level with duration",
						Command: suite.NewMessageBuilder().
							IntMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "cmd.lvl.set", "out_lvl_switch", 99).
							AddProperty(outlvlswitch.Duration, "1").
							Build(),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "out_lvl_switch"),
						},
					},
				},
			},
			{
				Name: "failed set level - level report error",
				Setup: routeLight(
					mockedoutlvlswitch.NewController(t).
						MockSetLevelSwitchLevel(99, nil, false).
						MockLevelSwitchLevelReport(99, errors.New("report error"), false),
				),
				Nodes: []*suite.Node{
					{
						Name:    "set level",
						Command: suite.IntMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "cmd.lvl.set", "out_lvl_switch", 99),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "out_lvl_switch"),
						},
					},
				},
			},
			{
				Name: "failed set binary - setting error",
				Setup: routeLight(
					mockedoutlvlswitch.NewController(t).
						MockSetLevelSwitchBinaryState(true, errors.New("setting error"), false),
				),
				Nodes: []*suite.Node{
					{
						Name:    "set binary",
						Command: suite.BoolMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "cmd.binary.set", "out_lvl_switch", true),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "out_lvl_switch"),
						},
					},
					{
						Name:    "wrong value type",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "cmd.binary.set", "out_lvl_switch", "true"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "out_lvl_switch"),
						},
					},
					{
						Name:    "wrong address",
						Command: suite.BoolMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:3", "cmd.binary.set", "out_lvl_switch", true),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:3", "out_lvl_switch"),
						},
					},
				},
			},
			{
				Name: "failed set binary - binary report error",
				Setup: routeLight(
					mockedoutlvlswitch.NewController(t).
						MockSetLevelSwitchBinaryState(true, nil, false).
						MockLevelSwitchBinaryReport(true, errors.New("report error"), false),
				),
				Nodes: []*suite.Node{
					{
						Name:    "set binary",
						Command: suite.BoolMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "cmd.binary.set", "out_lvl_switch", true),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "out_lvl_switch"),
						},
					},
				},
			},
			{
				Name: "failed get level - send level error",
				Setup: routeLight(
					mockedoutlvlswitch.NewController(t).
						MockLevelSwitchLevelReport(99, errors.New("sending error"), false),
				),
				Nodes: []*suite.Node{
					{
						Name:    "get level",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "cmd.lvl.get_report", "out_lvl_switch"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "out_lvl_switch"),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func TestTaskLight(t *testing.T) { //nolint:paralleltest
	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name: "Light tasks",
				Setup: taskLight(
					mockedoutlvlswitch.NewController(t).
						MockLevelSwitchLevelReport(99, nil, true).
						MockLevelSwitchBinaryReport(false, nil, true),
				),
				Nodes: []*suite.Node{
					{
						Name: "One error and one change",
						Expectations: []*suite.Expectation{
							suite.ExpectInt("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "evt.lvl.report", "out_lvl_switch", 99),
							suite.ExpectBool("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "evt.binary.report", "out_lvl_switch", false),
						},
					},
				},
			},
		},
	}

	s.Run(t)

}

func routeLight(
	lightController *mockedoutlvlswitch.Controller,
) suite.BaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
		t.Helper()

		routing, _, mocks := setupLight(t, mqtt, lightController, 0)

		return routing, nil, mocks
	}
}

func taskLight(
	lightController *mockedoutlvlswitch.Controller,
) suite.BaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
		t.Helper()

		_, tasks, mocks := setupLight(t, mqtt, lightController, 0)

		return nil, tasks, mocks
	}
}

func setupLight(
	t *testing.T,
	mqtt *fimpgo.MqttTransport,
	lightController *mockedoutlvlswitch.Controller,
	duration time.Duration,
) ([]*router.Routing, []*task.Task, []suite.Mock) {
	t.Helper()

	mocks := []suite.Mock{lightController}

	cfg := &thing.LightConfig{
		InclusionReport: &fimptype.ThingInclusionReport{
			Address: "2",
		},
		OutLvlSwitchConfig: &outlvlswitch.Config{
			Specification: outlvlswitch.Specification(
				"test_adapter",
				"1",
				"2",
				"99",
				"0",
				outlvlswitch.TypeOnAndOff,
				nil,
			),
			Controller: lightController,
		},
	}

	light := thing.NewLight(mqtt, cfg)
	ad := adapter.NewAdapter(nil, "test_adapter", "1")
	ad.RegisterThing(light)

	return thing.RouteLight(ad), thing.TaskLight(ad, duration), mocks
}
