package thermostat_test

import (
	"errors"
	"testing"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/thermostat"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
	adapterhelper "github.com/futurehomeno/cliffhanger/test/helper/adapter"
	mockedadapter "github.com/futurehomeno/cliffhanger/test/mocks/adapter"
	mockedthermostat "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/thermostat"
	cliffSuite "github.com/futurehomeno/cliffhanger/test/suite"
)

const (
	evtTopic = "pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:thermostat/ad:2"
	cmdTopic = "pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:thermostat/ad:2"
)

func TestRouteService(t *testing.T) { //nolint:paralleltest
	s := &cliffSuite.Suite{
		Cases: []*cliffSuite.Case{
			{
				Name:     "mode get report",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(mockedthermostat.NewController(t).
					MockThermostatModeReport("heat", nil, true),
				),
				Nodes: []*cliffSuite.Node{
					{
						Name: "cmd.mode.get_report returns mode",
						Command: cliffSuite.NewMessageBuilder().
							NullMessage(cmdTopic, thermostat.CmdModeGetReport, thermostat.Thermostat).
							Build(),
						Expectations: []*cliffSuite.Expectation{
							cliffSuite.ExpectString(evtTopic, thermostat.EvtModeReport, thermostat.Thermostat, "heat"),
						},
					},
				},
			},
			{
				Name:     "mode get report controller error",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(mockedthermostat.NewController(t).
					MockThermostatModeReport("", errors.New("controller error"), true),
				),
				Nodes: []*cliffSuite.Node{
					{
						Name: "cmd.mode.get_report returns error",
						Command: cliffSuite.NewMessageBuilder().
							NullMessage(cmdTopic, thermostat.CmdModeGetReport, thermostat.Thermostat).
							Build(),
						Expectations: []*cliffSuite.Expectation{
							cliffSuite.ExpectError(evtTopic, thermostat.Thermostat),
						},
					},
				},
			},
			{
				Name:     "mode set without setpoint",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(mockedthermostat.NewController(t).
					MockSetThermostatMode(thermostat.ModeOff, nil, true).
					MockThermostatModeReport(thermostat.ModeOff, nil, true),
				),
				Nodes: []*cliffSuite.Node{
					{
						Name: "cmd.mode.set off emits mode report only",
						Command: cliffSuite.NewMessageBuilder().
							StringMessage(cmdTopic, thermostat.CmdModeSet, thermostat.Thermostat, thermostat.ModeOff).
							Build(),
						Expectations: []*cliffSuite.Expectation{
							cliffSuite.ExpectString(evtTopic, thermostat.EvtModeReport, thermostat.Thermostat, thermostat.ModeOff),
						},
					},
				},
			},
			{
				Name:     "mode set with setpoint",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(mockedthermostat.NewController(t).
					MockSetThermostatMode("heat", nil, true).
					MockThermostatModeReport("heat", nil, true).
					MockThermostatSetpointReport("heat", 21.0, thermostat.UnitC, nil, true),
				),
				Nodes: []*cliffSuite.Node{
					{
						Name: "cmd.mode.set heat emits mode and setpoint reports",
						Command: cliffSuite.NewMessageBuilder().
							StringMessage(cmdTopic, thermostat.CmdModeSet, thermostat.Thermostat, "heat").
							Build(),
						Expectations: []*cliffSuite.Expectation{
							cliffSuite.ExpectString(evtTopic, thermostat.EvtModeReport, thermostat.Thermostat, "heat"),
							cliffSuite.ExpectStringMap(evtTopic, thermostat.EvtSetpointReport, thermostat.Thermostat,
								map[string]string{"type": "heat", "temp": "21.0", "unit": thermostat.UnitC}),
						},
					},
				},
			},
			{
				Name:     "mode set controller error",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(mockedthermostat.NewController(t).
					MockSetThermostatMode("heat", errors.New("controller error"), true),
				),
				Nodes: []*cliffSuite.Node{
					{
						Name: "cmd.mode.set returns error",
						Command: cliffSuite.NewMessageBuilder().
							StringMessage(cmdTopic, thermostat.CmdModeSet, thermostat.Thermostat, "heat").
							Build(),
						Expectations: []*cliffSuite.Expectation{
							cliffSuite.ExpectError(evtTopic, thermostat.Thermostat),
						},
					},
				},
			},
			{
				Name:     "setpoint get report",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(mockedthermostat.NewController(t).
					MockThermostatSetpointReport("heat", 21.0, thermostat.UnitC, nil, true),
				),
				Nodes: []*cliffSuite.Node{
					{
						Name: "cmd.setpoint.get_report returns setpoint",
						Command: cliffSuite.NewMessageBuilder().
							StringMessage(cmdTopic, thermostat.CmdSetpointGetReport, thermostat.Thermostat, "heat").
							Build(),
						Expectations: []*cliffSuite.Expectation{
							cliffSuite.ExpectStringMap(evtTopic, thermostat.EvtSetpointReport, thermostat.Thermostat,
								map[string]string{"type": "heat", "temp": "21.0", "unit": thermostat.UnitC}),
						},
					},
				},
			},
			{
				Name:     "setpoint set",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(mockedthermostat.NewController(t).
					MockSetThermostatSetpoint("heat", 22.0, thermostat.UnitC, nil, true).
					MockThermostatSetpointReport("heat", 22.0, thermostat.UnitC, nil, true),
				),
				Nodes: []*cliffSuite.Node{
					{
						Name: "cmd.setpoint.set emits setpoint report",
						Command: cliffSuite.NewMessageBuilder().
							StringMapMessage(cmdTopic, thermostat.CmdSetpointSet, thermostat.Thermostat,
								map[string]string{"type": "heat", "temp": "22.0", "unit": thermostat.UnitC}).
							Build(),
						Expectations: []*cliffSuite.Expectation{
							cliffSuite.ExpectStringMap(evtTopic, thermostat.EvtSetpointReport, thermostat.Thermostat,
								map[string]string{"type": "heat", "temp": "22.0", "unit": thermostat.UnitC}),
						},
					},
				},
			},
			{
				Name:     "state get report",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(mockedthermostat.NewController(t).
					MockThermostatStateReport(thermostat.StateHeat, nil, true),
				),
				Nodes: []*cliffSuite.Node{
					{
						Name: "cmd.state.get_report returns state",
						Command: cliffSuite.NewMessageBuilder().
							NullMessage(cmdTopic, thermostat.CmdStateGetReport, thermostat.Thermostat).
							Build(),
						Expectations: []*cliffSuite.Expectation{
							cliffSuite.ExpectString(evtTopic, thermostat.EvtStateReport, thermostat.Thermostat, thermostat.StateHeat),
						},
					},
				},
			},
			{
				Name:     "service not found",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup:    routeService(mockedthermostat.NewController(t)),
				Nodes: []*cliffSuite.Node{
					{
						Name: "unknown address returns error",
						Command: cliffSuite.NewMessageBuilder().
							NullMessage(
								"pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:thermostat/ad:404",
								thermostat.CmdModeGetReport,
								thermostat.Thermostat,
							).
							Build(),
						Expectations: []*cliffSuite.Expectation{
							cliffSuite.ExpectError(
								"pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:thermostat/ad:404",
								thermostat.Thermostat,
							),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func routeService(controller *mockedthermostat.Controller) cliffSuite.BaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []cliffSuite.Mock) {
		t.Helper()

		return setupService(t, mqtt, controller)
	}
}

func setupService(t *testing.T, mqtt *fimpgo.MqttTransport, controller *mockedthermostat.Controller) ([]*router.Routing, []*task.Task, []cliffSuite.Mock) {
	t.Helper()

	thingCfg := &adapter.ThingConfig{
		InclusionReport: &fimptype.ThingInclusionReport{Address: "2"},
		Connector:       mockedadapter.NewDefaultConnector(t),
	}

	svcCfg := &thermostat.Config{
		Specification: thermostat.Specification(
			"test_adapter",
			"1",
			"2",
			nil,
			[]string{"heat", thermostat.ModeOff},
			[]string{"heat"},
			[]string{thermostat.StateHeat, thermostat.StateIdle},
		),
		Controller: controller,
	}

	seed := &adapter.ThingSeed{ID: "B", CustomAddress: "2"}

	factory := adapterhelper.FactoryHelper(func(a adapter.Adapter, p adapter.Publisher, ts adapter.ThingState) (adapter.Thing, error) {
		return adapter.NewThing(p, ts, thingCfg, thermostat.NewService(p, svcCfg)), nil
	})

	ad := adapterhelper.PrepareSeededAdapter(t, "../../testdata/adapter/test_adapter", mqtt, factory, adapter.ThingSeeds{seed})

	return thermostat.RouteService(ad), nil, nil
}
