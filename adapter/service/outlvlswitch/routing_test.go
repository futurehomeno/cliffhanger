package outlvlswitch_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/outlvlswitch"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
	adapterhelper "github.com/futurehomeno/cliffhanger/test/helper/adapter"
	mockedadapter "github.com/futurehomeno/cliffhanger/test/mocks/adapter"
	mockedoutlvlswitch "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/outlvlswitch"
	"github.com/futurehomeno/cliffhanger/test/suite"
	"github.com/futurehomeno/cliffhanger/utils"
)

func TestRouteService(t *testing.T) { //nolint:paralleltest
	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "Should succeed processing commands",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(
					mockedoutlvlswitch.NewMockedOutSwitchLvl(
						mockedoutlvlswitch.NewController(t).
							MockLevelSwitchLevelReport(2, nil, true).
							MockLevelSwitchLevelReport(3, nil, true).
							MockLevelSwitchLevelReport(4, nil, true).
							MockSetLevelSwitchBinaryState(true, nil, false).
							MockSetLevelSwitchLevel(1, 0, nil, false),
						mockedoutlvlswitch.NewLevelTransitionController(t).
							MockStartLevelTransition("up", outlvlswitch.LevelTransitionParams{}, nil).
							MockStopLevelTransition(nil),
					),
				),
				Nodes: []*suite.Node{
					{
						Name:         "Start level transition",
						Command:      suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "cmd.lvl.start", "out_lvl_switch", "up"),
						Expectations: []*suite.Expectation{},
					},
					{
						Name:    "Stop level transition",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "cmd.lvl.stop", "out_lvl_switch", ""),
						Expectations: []*suite.Expectation{
							suite.ExpectInt("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "evt.lvl.report", "out_lvl_switch", 2),
						},
					},
					{
						Name:    "Switch binary",
						Command: suite.BoolMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "cmd.binary.set", "out_lvl_switch", true),
						Expectations: []*suite.Expectation{
							suite.ExpectInt("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "evt.lvl.report", "out_lvl_switch", 3),
						},
					},
					{
						Name:    "Set level",
						Command: suite.IntMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "cmd.lvl.set", "out_lvl_switch", 1),
						Expectations: []*suite.Expectation{
							suite.ExpectInt("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "evt.lvl.report", "out_lvl_switch", 4),
						},
					},
					{
						Name: "Start level transition. Should avoid processing properties when not supported.",
						Command: suite.NewMessageBuilder().
							StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "cmd.lvl.start", "out_lvl_switch", "up").
							AddProperty("duration", "5").
							AddProperty("start_lvl", "4").
							Build(),
						Expectations: []*suite.Expectation{},
					},
				},
			},
			{
				Name:     "Should succeed processing commands for devices with properties",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(
					mockedoutlvlswitch.NewMockedOutSwitchLvl(
						mockedoutlvlswitch.NewController(t).
							MockLevelSwitchLevelReport(3, nil, false).
							MockSetLevelSwitchLevel(1, time.Second, nil, false),
						mockedoutlvlswitch.NewLevelTransitionController(t).
							MockStartLevelTransition("up", outlvlswitch.LevelTransitionParams{StartLvl: utils.Ptr(4), Duration: utils.Ptr(5. * time.Second)}, nil),
					),
					outlvlswitch.WithSupportedDuration(),
					outlvlswitch.WithSupportedStartLevel(),
				),
				Nodes: []*suite.Node{
					{
						Name: "Start level transition",
						Command: suite.NewMessageBuilder().
							StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "cmd.lvl.start", "out_lvl_switch", "up").
							AddProperty("duration", "5").
							AddProperty("start_lvl", "4").
							Build(),
						Expectations: []*suite.Expectation{},
					},
					{
						Name: "Set level",
						Command: suite.NewMessageBuilder().
							IntMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "cmd.lvl.set", "out_lvl_switch", 1).
							AddProperty("duration", "1").
							Build(),
						Expectations: []*suite.Expectation{
							suite.ExpectInt("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "evt.lvl.report", "out_lvl_switch", 3),
						},
					},
				},
			},
			{
				Name:     "Should not send reports when an error occurred",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(
					mockedoutlvlswitch.NewMockedOutSwitchLvl(
						mockedoutlvlswitch.NewController(t).
							MockSetLevelSwitchBinaryState(true, fmt.Errorf("some error"), false).
							MockSetLevelSwitchLevel(1, 0, fmt.Errorf("error"), false),
						mockedoutlvlswitch.NewLevelTransitionController(t).
							MockStartLevelTransition("up", outlvlswitch.LevelTransitionParams{}, fmt.Errorf("error")).
							MockStopLevelTransition(fmt.Errorf("some error")),
					),
				),
				Nodes: []*suite.Node{
					{
						Name:    "Start level transition",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "cmd.lvl.start", "out_lvl_switch", "up"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "out_lvl_switch"),
						},
					},
					{
						Name:    "Start level transition. Not string value",
						Command: suite.BoolMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "cmd.lvl.start", "out_lvl_switch", true),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "out_lvl_switch"),
						},
					},
					{
						Name:    "Start level transition. Incorrect value",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "cmd.lvl.start", "out_lvl_switch", "invalid"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "out_lvl_switch"),
						},
					},
					{
						Name: "Start level transition with invalid duration property",
						Command: suite.NewMessageBuilder().
							StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "cmd.lvl.start", "out_lvl_switch", "up").
							AddProperty("duration", "invalid").
							Build(),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "out_lvl_switch"),
						},
					},
					{
						Name: "Start level transition  with invalid start_lvl property",
						Command: suite.NewMessageBuilder().
							StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "cmd.lvl.start", "out_lvl_switch", "up").
							AddProperty("duration", "3").
							AddProperty("start_lvl", "invalid int").
							Build(),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "out_lvl_switch"),
						},
					},
					{
						Name: "Start level transition with start_lvl property out of range",
						Command: suite.NewMessageBuilder().
							StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "cmd.lvl.start", "out_lvl_switch", "up").
							AddProperty("duration", "4").
							AddProperty("start_lvl", "109").
							Build(),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "out_lvl_switch"),
						},
					},
					{
						Name:    "Stop level transition",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "cmd.lvl.stop", "out_lvl_switch", ""),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "out_lvl_switch"),
						},
					},
					{
						Name:    "Set binary cmd",
						Command: suite.BoolMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "cmd.binary.set", "out_lvl_switch", true),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "out_lvl_switch"),
						},
					},
					{
						Name:    "Set binary cmd. Not boolean value",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "cmd.binary.set", "out_lvl_switch", "invalid"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "out_lvl_switch"),
						},
					},
					{
						Name:    "Set level",
						Command: suite.IntMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "cmd.lvl.set", "out_lvl_switch", 1),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "out_lvl_switch"),
						},
					},
					{
						Name:    "Set level. Invalid value.",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "cmd.lvl.set", "out_lvl_switch", "invalid"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "out_lvl_switch"),
						},
					},
					{
						Name: "Set level. Invalid duration property",
						Command: suite.NewMessageBuilder().
							IntMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "cmd.lvl.set", "out_lvl_switch", 1).
							AddProperty("duration", "invalid").
							Build(),
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

func routeService(controller outlvlswitch.Controller, options ...adapter.SpecificationOption) suite.BaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
		t.Helper()

		routing, _, mocks := setupService(t, mqtt, controller, 0, options...)

		return routing, nil, mocks
	}
}

func setupService(
	t *testing.T,
	mqtt *fimpgo.MqttTransport,
	controller outlvlswitch.Controller,
	duration time.Duration,
	options ...adapter.SpecificationOption,
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
		Connector: mockedadapter.NewConnector(t),
	}

	switchCfg := &outlvlswitch.Config{
		Specification: outlvlswitch.Specification(
			"test_adapter",
			"1",
			"2",
			outlvlswitch.SwitchTypeOnAndOff,
			99,
			0,
			nil,
			options...,
		),
		Controller: controller,
	}

	seed := &adapter.ThingSeed{ID: "B", CustomAddress: "2"}

	factory := adapterhelper.FactoryHelper(func(a adapter.Adapter, publisher adapter.Publisher, thingState adapter.ThingState) (adapter.Thing, error) {
		return adapter.NewThing(publisher, thingState, thingCfg, outlvlswitch.NewService(publisher, switchCfg)), nil
	})

	ad := adapterhelper.PrepareSeededAdapter(t, "../../testdata/adapter/test_adapter", mqtt, factory, adapter.ThingSeeds{seed})

	return outlvlswitch.RouteService(ad), task.Combine(outlvlswitch.TaskReporting(ad, duration)), mocks
}
