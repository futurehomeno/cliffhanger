package waterheater_test

import (
	"errors"
	"testing"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/waterheater"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
	adapterhelper "github.com/futurehomeno/cliffhanger/test/helper/adapter"
	mockedadapter "github.com/futurehomeno/cliffhanger/test/mocks/adapter"
	mockedwaterheater "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/waterheater"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

var (
	errTest = errors.New("test")

	testModes     = []string{"heat", "eco", "off"}
	testSetpoints = []string{"heat", "eco"}
	testStates    = []string{"heat", "idle"}
	testRange     = &waterheater.Range{Min: 10, Max: 80}
	testStep      = 0.5
)

func TestRouteService(t *testing.T) { //nolint:paralleltest
	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "successful mode set - with setpoint support",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(
					mockedwaterheater.NewController(t).
						MockSetWaterHeaterMode("heat", nil, true).
						MockWaterHeaterModeReport("heat", nil, true).
						MockWaterHeaterSetpointReport("heat", 65.5, "C", nil, true),
					testModes, testSetpoints, testStates, testRange, nil, testStep,
				),
				Nodes: []*suite.Node{
					{
						Name:    "set mode",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "cmd.mode.set", "water_heater", "heat"),
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "evt.mode.report", "water_heater", "heat"),
							suite.ExpectObject("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "evt.setpoint.report", "water_heater", waterheater.NewSetpoint("heat", 65.5, "C")),
						},
					},
				},
			},
			{
				Name:     "successful mode set - without setpoint support",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(
					mockedwaterheater.NewController(t).
						MockSetWaterHeaterMode("off", nil, true).
						MockWaterHeaterModeReport("off", nil, true),
					testModes, testSetpoints, testStates, testRange, nil, testStep,
				),
				Nodes: []*suite.Node{
					{
						Name:    "set mode",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "cmd.mode.set", "water_heater", "off"),
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "evt.mode.report", "water_heater", "off"),
							suite.ExpectMessage("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "evt.setpoint.report", "water_heater").Never(),
						},
					},
				},
			},
			{
				Name:     "failed mode set - unsupported mode",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(
					mockedwaterheater.NewController(t),
					testModes, testSetpoints, testStates, testRange, nil, testStep,
				),
				Nodes: []*suite.Node{
					{
						Name:    "set mode",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "cmd.mode.set", "water_heater", "dummy"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "water_heater"),
						},
					},
				},
			},
			{
				Name:     "successful setpoint set",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(
					mockedwaterheater.NewController(t).
						MockSetWaterHeaterSetpoint("heat", 65, "C", nil, true).
						MockWaterHeaterSetpointReport("heat", 65, "C", nil, true),
					testModes, testSetpoints, testStates, testRange, nil, testStep,
				),
				Nodes: []*suite.Node{
					{
						Name:    "set setpoint",
						Command: suite.ObjectMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "cmd.setpoint.set", "water_heater", waterheater.NewSetpoint("heat", 65, "C")),
						Expectations: []*suite.Expectation{
							suite.ExpectObject("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "evt.setpoint.report", "water_heater", waterheater.NewSetpoint("heat", 65, "C")),
						},
					},
				},
			},
			{
				Name:     "failed setpoint set - out of range value",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(
					mockedwaterheater.NewController(t),
					testModes, testSetpoints, testStates, testRange, nil, testStep,
				),
				Nodes: []*suite.Node{
					{
						Name:    "set setpoint",
						Command: suite.ObjectMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "cmd.setpoint.set", "water_heater", waterheater.NewSetpoint("heat", 999, "C")),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "water_heater"),
						},
					},
				},
			},
			{
				Name:     "failed setpoint set - unsupported mode",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(
					mockedwaterheater.NewController(t),
					testModes, testSetpoints, testStates, testRange, nil, testStep,
				),
				Nodes: []*suite.Node{
					{
						Name:    "set setpoint",
						Command: suite.ObjectMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "cmd.setpoint.set", "water_heater", waterheater.NewSetpoint("dummy", 50, "C")),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "water_heater"),
						},
					},
				},
			},
			{
				Name:     "mode get_report - success and error",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(
					mockedwaterheater.NewController(t).
						MockWaterHeaterModeReport("eco", nil, true).
						MockWaterHeaterModeReport("", errTest, true),
					testModes, testSetpoints, testStates, testRange, nil, testStep,
				),
				Nodes: []*suite.Node{
					{
						Name:    "get mode report",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "cmd.mode.get_report", "water_heater"),
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "evt.mode.report", "water_heater", "eco"),
						},
					},
					{
						Name:    "get mode report - error",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "cmd.mode.get_report", "water_heater"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "water_heater"),
						},
					},
				},
			},
			{
				Name:     "setpoint get_report - success, error and unsupported mode",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(
					mockedwaterheater.NewController(t).
						MockWaterHeaterSetpointReport("heat", 45, "C", nil, true).
						MockWaterHeaterSetpointReport("eco", 0, "", errTest, true),
					testModes, testSetpoints, testStates, testRange, nil, testStep,
				),
				Nodes: []*suite.Node{
					{
						Name:    "get setpoint report",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "cmd.setpoint.get_report", "water_heater", "heat"),
						Expectations: []*suite.Expectation{
							suite.ExpectObject("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "evt.setpoint.report", "water_heater", waterheater.NewSetpoint("heat", 45, "C")),
						},
					},
					{
						Name:    "get setpoint report - error",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "cmd.setpoint.get_report", "water_heater", "eco"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "water_heater"),
						},
					},
					{
						Name:    "get setpoint report - unsupported mode",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "cmd.setpoint.get_report", "water_heater", "dummy"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "water_heater"),
						},
					},
				},
			},
			{
				Name:     "state get_report - success and error",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(
					mockedwaterheater.NewController(t).
						MockWaterHeaterStateReport("idle", nil, true).
						MockWaterHeaterStateReport("", errTest, true),
					testModes, testSetpoints, testStates, testRange, nil, testStep,
				),
				Nodes: []*suite.Node{
					{
						Name:    "get state report",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "cmd.state.get_report", "water_heater"),
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "evt.state.report", "water_heater", "idle"),
						},
					},
					{
						Name:    "get state report - error",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "cmd.state.get_report", "water_heater"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "water_heater"),
						},
					},
				},
			},
			{
				Name:     "malformed payloads",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(
					mockedwaterheater.NewController(t),
					testModes, testSetpoints, testStates, testRange, nil, testStep,
				),
				Nodes: []*suite.Node{
					{
						Name:    "non-string value on setting mode",
						Command: suite.BoolMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "cmd.mode.set", "water_heater", true),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "water_heater"),
						},
					},
					{
						Name:    "non-object value on setting setpoint",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "cmd.setpoint.set", "water_heater", "heat"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "water_heater"),
						},
					},
					{
						Name:    "non-string value on getting setpoint report",
						Command: suite.BoolMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "cmd.setpoint.get_report", "water_heater", true),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "water_heater"),
						},
					},
				},
			},
			{
				Name:     "service not found",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(
					mockedwaterheater.NewController(t),
					testModes, testSetpoints, testStates, testRange, nil, testStep,
				),
				Nodes: []*suite.Node{
					{
						Name:    "non-existent thing on setting mode",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:3", "cmd.mode.set", "water_heater", "heat"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:3", "water_heater"),
						},
					},
					{
						Name:    "non-existent thing on setting setpoint",
						Command: suite.ObjectMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:3", "cmd.setpoint.set", "water_heater", waterheater.NewSetpoint("heat", 50, "C")),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:3", "water_heater"),
						},
					},
					{
						Name:    "non-existent thing on mode report",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:3", "cmd.mode.get_report", "water_heater"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:3", "water_heater"),
						},
					},
					{
						Name:    "non-existent thing on setpoint report",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:3", "cmd.setpoint.get_report", "water_heater", "heat"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:3", "water_heater"),
						},
					},
					{
						Name:    "non-existent thing on state report",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:3", "cmd.state.get_report", "water_heater"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:3", "water_heater"),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func routeService(
	controller waterheater.Controller,
	modes, setpoints, states []string,
	supportedRange *waterheater.Range,
	supportedRanges map[string]waterheater.Range,
	step float64,
) suite.BaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
		t.Helper()

		routing, _, mocks := setupService(t, mqtt, controller, modes, setpoints, states, supportedRange, supportedRanges, step, 0)

		return routing, nil, mocks
	}
}

func setupService(
	t *testing.T,
	mqtt *fimpgo.MqttTransport,
	controller waterheater.Controller,
	modes, setpoints, states []string,
	supportedRange *waterheater.Range,
	supportedRanges map[string]waterheater.Range,
	step float64,
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

	waterHeaterCfg := &waterheater.Config{
		Specification: waterheater.Specification(
			"test_adapter",
			"1",
			"2",
			nil,
			modes,
			setpoints,
			states,
			supportedRange,
			supportedRanges,
			step,
		),
		Controller: controller,
	}

	seed := &adapter.ThingSeed{ID: "B", CustomAddress: "2"}

	factory := adapterhelper.FactoryHelper(func(a adapter.Adapter, publisher adapter.Publisher, thingState adapter.ThingState) (adapter.Thing, error) {
		return adapter.NewThing(publisher, thingState, thingCfg, waterheater.NewService(publisher, waterHeaterCfg)), nil
	})

	ad := adapterhelper.PrepareSeededAdapter(t, "../../testdata/adapter/test_adapter", mqtt, factory, adapter.ThingSeeds{seed})

	return waterheater.RouteService(ad), task.Combine(waterheater.TaskReporting(ad, duration)), mocks
}
