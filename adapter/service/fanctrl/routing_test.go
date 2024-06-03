package fanctrl_test

import (
	"errors"
	"testing"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/fanctrl"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
	adapterhelper "github.com/futurehomeno/cliffhanger/test/helper/adapter"
	mockedadapter "github.com/futurehomeno/cliffhanger/test/mocks/adapter"
	mockedfanctrl "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/fanctrl"
	cliffSuite "github.com/futurehomeno/cliffhanger/test/suite"
)

func TestRouteService(t *testing.T) { //nolint:paralleltest
	s := &cliffSuite.Suite{
		Cases: []*cliffSuite.Case{
			{
				Name:     "fan ctrl get report",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(mockedfanctrl.NewController(t).
					MockGetMode("normal", nil, true),
				),
				Nodes: []*cliffSuite.Node{
					{
						Name: "Cmd mode get report",
						Commands: []*fimpgo.Message{
							cliffSuite.NewMessageBuilder().
								NullMessage(
									"pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:fan_ctrl/ad:2",
									"cmd.mode.get_report",
									"fan_ctrl",
								).
								Build(),
						},
						Expectations: []*cliffSuite.Expectation{
							cliffSuite.ExpectString("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:fan_ctrl/ad:2", "evt.mode.report", "fan_ctrl", "normal"),
						},
					},
				},
			},
			{
				Name:     "fan ctrl set report",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(mockedfanctrl.NewController(t).
					MockSetMode("night", nil, true).
					MockGetMode("night", nil, true),
				),
				Nodes: []*cliffSuite.Node{
					{
						Name: "Cmd mode set",
						Commands: []*fimpgo.Message{cliffSuite.NewMessageBuilder().
							StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:fan_ctrl/ad:2", "cmd.mode.set", "fan_ctrl", "night").
							Build()},
						Expectations: []*cliffSuite.Expectation{
							cliffSuite.ExpectString("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:fan_ctrl/ad:2", "evt.mode.report", "fan_ctrl", "night"),
						},
					},
					{
						Name: "Device does not exists",
						Commands: []*fimpgo.Message{cliffSuite.NewMessageBuilder().
							StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:fan_ctrl/ad:404", "cmd.mode.set", "fan_ctrl", "night").
							Build()},
						Expectations: []*cliffSuite.Expectation{
							cliffSuite.ExpectMessage("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:fan_ctrl/ad:404", "evt.error.report", "fan_ctrl"),
						},
					},
					{
						Name: "Wrong message type",
						Commands: []*fimpgo.Message{cliffSuite.NewMessageBuilder().
							FloatMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:fan_ctrl/ad:2", "cmd.mode.set", "fan_ctrl", 1).
							Build()},
						Expectations: []*cliffSuite.Expectation{
							cliffSuite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:fan_ctrl/ad:2", "fan_ctrl"),
						},
					},
				},
			},
			{
				Name:     "broken set mode in controller",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(mockedfanctrl.NewController(t).
					MockSetMode("night", errors.New("broken test controller"), true),
				),
				Nodes: []*cliffSuite.Node{
					{
						Name: "Cmd mode set with error",
						Commands: []*fimpgo.Message{cliffSuite.NewMessageBuilder().
							StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:fan_ctrl/ad:2", "cmd.mode.set", "fan_ctrl", "night").
							Build()},
						Expectations: []*cliffSuite.Expectation{
							cliffSuite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:fan_ctrl/ad:2", "fan_ctrl"),
						},
					},
				},
			},
			{
				Name:     "broken get mode in controller",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(mockedfanctrl.NewController(t).
					MockGetMode("night", errors.New("broken test controller"), true),
				),
				Nodes: []*cliffSuite.Node{
					{
						Name: "Cmd mode get_report with error",
						Commands: []*fimpgo.Message{cliffSuite.NewMessageBuilder().
							StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:fan_ctrl/ad:2", "cmd.mode.get_report", "fan_ctrl", "night").
							Build()},
						Expectations: []*cliffSuite.Expectation{
							cliffSuite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:fan_ctrl/ad:2", "fan_ctrl"),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func routeService(controller *mockedfanctrl.Controller) cliffSuite.BaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []cliffSuite.Mock) {
		t.Helper()

		return setupService(t, mqtt, controller)
	}
}

func setupService(t *testing.T, mqtt *fimpgo.MqttTransport, controller *mockedfanctrl.Controller) ([]*router.Routing, []*task.Task, []cliffSuite.Mock) {
	t.Helper()

	thingCfg := &adapter.ThingConfig{
		InclusionReport: &fimptype.ThingInclusionReport{
			Address: "2",
		},
		Connector: mockedadapter.NewDefaultConnector(t),
	}

	fanCfg := &fanctrl.Config{
		Specification: fanctrl.Specification(
			"test_adapter",
			"1",
			"2",
			nil,
			[]string{"normal", "night", "away", "boost"},
		),
		Controller: controller,
	}

	seed := &adapter.ThingSeed{ID: "B", CustomAddress: "2"}

	factory := adapterhelper.FactoryHelper(func(_ adapter.Adapter, p adapter.Publisher, ts adapter.ThingState) (adapter.Thing, error) {
		return adapter.NewThing(p, ts, thingCfg, fanctrl.NewService(p, fanCfg)), nil
	})
	ad := adapterhelper.PrepareSeededAdapter(t, "../../testdata/adapter/test_adapter", mqtt, factory, adapter.ThingSeeds{seed})
	fanctrl.RouteService(ad)

	return fanctrl.RouteService(ad), nil, nil
}
