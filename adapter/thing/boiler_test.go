package thing_test

import (
	"errors"
	"testing"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/waterheater"
	"github.com/futurehomeno/cliffhanger/adapter/thing"
	"github.com/futurehomeno/cliffhanger/router"
	mockedwaterheater "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/waterheater"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

func TestRouteBoiler(t *testing.T) { //nolint:paralleltest
	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name: "Successful boiler query",
				Setup: setupBoiler(
					mockedwaterheater.MockController().
						MockWaterHeaterModeReport("test_mode_a", nil, true).
						MockWaterHeaterSetpointReport("test_mode_a", 60, "C", nil, true).
						MockWaterHeaterStateReport("idle", nil, true),
				),
				Nodes: []*suite.Node{
					{
						Name:    "Mode",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_service/ad:1/sv:water_heater/ad:2", "cmd.mode.get_report", "water_heater"),
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:dev/rn:test_service/ad:1/sv:water_heater/ad:2", "evt.mode.report", "water_heater", "test_mode_a"),
						},
					},
					{
						Name:    "Setpoint",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_service/ad:1/sv:water_heater/ad:2", "cmd.setpoint.get_report", "water_heater", "test_mode_a"),
						Expectations: []*suite.Expectation{
							suite.ExpectObject("pt:j1/mt:evt/rt:dev/rn:test_service/ad:1/sv:water_heater/ad:2", "evt.setpoint.report", "water_heater", waterheater.Setpoint{Type: "test_mode_a", Temperature: 60, Unit: "C"}),
						},
					},
					{
						Name:    "State",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_service/ad:1/sv:water_heater/ad:2", "cmd.state.get_report", "water_heater"),
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:dev/rn:test_service/ad:1/sv:water_heater/ad:2", "evt.state.report", "water_heater", "idle"),
						},
					},
				},
			},
			{
				Name: "Failed boiler query",
				Setup: setupBoiler(
					mockedwaterheater.MockController().
						MockWaterHeaterModeReport("test_mode_a", errors.New("test"), true).
						MockWaterHeaterSetpointReport("test_mode_a", 0, "", errors.New("test"), true).
						MockWaterHeaterStateReport("", errors.New("test"), true),
				),
				Nodes: []*suite.Node{
					{
						Name:    "Controller error when retrieving mode",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_service/ad:1/sv:water_heater/ad:2", "cmd.mode.get_report", "water_heater"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_service/ad:1/sv:water_heater/ad:2", "water_heater"),
						},
					},
					{
						Name:    "Controller error when retrieving setpoint",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_service/ad:1/sv:water_heater/ad:2", "cmd.setpoint.get_report", "water_heater", "test_mode_a"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_service/ad:1/sv:water_heater/ad:2", "water_heater"),
						},
					},
					{
						Name:    "Controller error when retrieving state",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_service/ad:1/sv:water_heater/ad:2", "cmd.state.get_report", "water_heater"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_service/ad:1/sv:water_heater/ad:2", "water_heater"),
						},
					},
					{
						Name:    "Querying for mode of non existent thing",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_service/ad:1/sv:water_heater/ad:3", "cmd.mode.get_report", "water_heater"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_service/ad:1/sv:water_heater/ad:3", "water_heater"),
						},
					},
					{
						Name:    "Querying for setpoint of non existent thing",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_service/ad:1/sv:water_heater/ad:3", "cmd.setpoint.get_report", "water_heater", "test_mode_a"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_service/ad:1/sv:water_heater/ad:3", "water_heater"),
						},
					},
					{
						Name:    "Querying for state of non existent thing",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_service/ad:1/sv:water_heater/ad:3", "cmd.state.get_report", "water_heater"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_service/ad:1/sv:water_heater/ad:3", "water_heater"),
						},
					},
					{
						Name:    "Querying for setpoint for unsupported mode",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_service/ad:1/sv:water_heater/ad:2", "cmd.setpoint.get_report", "water_heater", "unsupported_mode"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_service/ad:1/sv:water_heater/ad:2", "water_heater"),
						},
					},
					{
						Name:    "Querying for setpoint with wrong type of value",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_service/ad:1/sv:water_heater/ad:2", "cmd.setpoint.get_report", "water_heater"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_service/ad:1/sv:water_heater/ad:2", "water_heater"),
						},
					},
				},
			},
			{
				Name:  "",
				Setup: setupBoiler(
					mockedwaterheater.MockController().
						MockSetWaterHeaterMode("test_mode_c", nil, true).
						MockWaterHeaterModeReport("test_mode_c", nil, true).
						MockSetWaterHeaterMode("test_mode_a", nil, true).
						MockWaterHeaterModeReport("test_mode_a", nil, true).
						MockWaterHeaterSetpointReport("test_mode_a", 60, "C", nil, true).
						MockSetWaterHeaterSetpoint("test_mode_a", 70, "C", nil, true).
						MockWaterHeaterSetpointReport("test_mode_a", 70, "C", nil, true),
				),
				Nodes: []*suite.Node{
					{
						Name:    "Set mode not supporting a setpoint",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_service/ad:1/sv:water_heater/ad:2", "cmd.mode.set", "water_heater", "test_mode_c"),
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:dev/rn:test_service/ad:1/sv:water_heater/ad:2", "evt.mode.report", "water_heater", "test_mode_c"),
						},
					},
					{
						Name:    "Set mode  supporting a setpoint",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_service/ad:1/sv:water_heater/ad:2", "cmd.mode.set", "water_heater", "test_mode_a"),
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:dev/rn:test_service/ad:1/sv:water_heater/ad:2", "evt.mode.report", "water_heater", "test_mode_a"),
							suite.ExpectObject("pt:j1/mt:evt/rt:dev/rn:test_service/ad:1/sv:water_heater/ad:2", "evt.setpoint.report", "water_heater", waterheater.Setpoint{Type: "test_mode_a", Temperature: 60, Unit: "C"}),

						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func setupBoiler(waterHeaterController *mockedwaterheater.Controller) suite.CaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []suite.Mock) {
		cfg := &thing.BoilerConfig{
			InclusionReport: &fimptype.ThingInclusionReport{
				Address: "2",
			},
			WaterHeaterConfig: &waterheater.Config{
				Specification: waterheater.Specification(
					"test_service",
					"1",
					"2",
					nil,
					[]string{"test_mode_a", "test_mode_b", "test_mode_c", "off"},
					[]string{"test_mode_a", "test_mode_b"},
					[]string{"heat", "idle"},
					&waterheater.Range{Min: 20, Max: 80},
					map[string]waterheater.Range{
						"test_mode_a": {Min: 25, Max: 75},
					},
					0.5,
				),
				Controller: waterHeaterController,
			},
		}

		b := thing.NewBoiler(
			mqtt,
			cfg,
		)

		ad := adapter.NewAdapter(nil, "test_service", "1")
		ad.RegisterThing(b)

		return thing.RouteBoiler(ad), []suite.Mock{waterHeaterController}
	}

}
