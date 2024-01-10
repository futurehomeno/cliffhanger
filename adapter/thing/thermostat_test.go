package thing_test

import (
	"errors"
	"testing"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/numericmeter"
	"github.com/futurehomeno/cliffhanger/adapter/service/numericsensor"
	"github.com/futurehomeno/cliffhanger/adapter/service/thermostat"
	"github.com/futurehomeno/cliffhanger/adapter/thing"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
	adapterhelper "github.com/futurehomeno/cliffhanger/test/helper/adapter"
	mockedadapter "github.com/futurehomeno/cliffhanger/test/mocks/adapter"
	mockednumericmeter "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/numericmeter"
	mockednumericsensor "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/numericsensor"
	mockedthermostat "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/thermostat"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

func TestRouteThermostat(t *testing.T) { //nolint:paralleltest
	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "Successful thermostat reporting",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeThermostat(
					mockedthermostat.NewController(t).
						MockThermostatModeReport("test_mode_a", nil, true).
						MockThermostatSetpointReport("test_mode_a", 21, "C", nil, true).
						MockThermostatStateReport("idle", nil, true),
					mockednumericsensor.NewReporter(t).
						MockNumericSensorReport("C", 21.5, nil, false),
					mockednumericmeter.NewReporter(t).
						MockMeterReport("W", 2, nil, false).
						MockMeterReport("kWh", 123.45, nil, false),
				),
				Nodes: []*suite.Node{
					{
						Name:    "Mode",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:thermostat/ad:2", "cmd.mode.get_report", "thermostat"),
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:thermostat/ad:2", "evt.mode.report", "thermostat", "test_mode_a"),
						},
					},
					{
						Name:    "Setpoint",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:thermostat/ad:2", "cmd.setpoint.get_report", "thermostat", "test_mode_a"),
						Expectations: []*suite.Expectation{
							suite.ExpectStringMap("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:thermostat/ad:2", "evt.setpoint.report", "thermostat", map[string]string{"type": "test_mode_a", "temp": "21.0", "unit": "C"}),
						},
					},
					{
						Name:    "State",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:thermostat/ad:2", "cmd.state.get_report", "thermostat"),
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:thermostat/ad:2", "evt.state.report", "thermostat", "idle"),
						},
					},
					{
						Name:    "Temperature",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:sensor_temp/ad:2", "cmd.sensor.get_report", "sensor_temp", "C"),
						Expectations: []*suite.Expectation{
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:sensor_temp/ad:2", "evt.sensor.report", "sensor_temp", 21.5).
								ExpectProperty("unit", "C"),
						},
					},
					{
						Name:    "All sensor units",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:sensor_temp/ad:2", "cmd.sensor.get_report", "sensor_temp", ""),
						Expectations: []*suite.Expectation{
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:sensor_temp/ad:2", "evt.sensor.report", "sensor_temp", 21.5).
								ExpectProperty("unit", "C"),
						},
					},
					{
						Name:    "All sensor units with null",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:sensor_temp/ad:2", "cmd.sensor.get_report", "sensor_temp"),
						Expectations: []*suite.Expectation{
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:sensor_temp/ad:2", "evt.sensor.report", "sensor_temp", 21.5).
								ExpectProperty("unit", "C"),
						},
					},
					{
						Name:    "Power",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "cmd.meter.get_report", "meter_elec", "W"),
						Expectations: []*suite.Expectation{
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "evt.meter.report", "meter_elec", 2).
								ExpectProperty("unit", "W"),
						},
					},
					{
						Name:    "Energy",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "cmd.meter.get_report", "meter_elec", "kWh"),
						Expectations: []*suite.Expectation{
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "evt.meter.report", "meter_elec", 123.45).
								ExpectProperty("unit", "kWh"),
						},
					},
					{
						Name:    "All electricity units",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "cmd.meter.get_report", "meter_elec", ""),
						Expectations: []*suite.Expectation{
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "evt.meter.report", "meter_elec", 2).
								ExpectProperty("unit", "W"),
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "evt.meter.report", "meter_elec", 123.45).
								ExpectProperty("unit", "kWh"),
						},
					},
					{
						Name:    "All electricity units with null",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "cmd.meter.get_report", "meter_elec"),
						Expectations: []*suite.Expectation{
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "evt.meter.report", "meter_elec", 2).
								ExpectProperty("unit", "W"),
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "evt.meter.report", "meter_elec", 123.45).
								ExpectProperty("unit", "kWh"),
						},
					},
				},
			},
			{
				Name:     "Failed thermostat reporting",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeThermostat(
					mockedthermostat.NewController(t).
						MockThermostatModeReport("test_mode_a", errors.New("test"), true).
						MockThermostatSetpointReport("test_mode_a", 0, "", errors.New("test"), true).
						MockThermostatStateReport("", errors.New("test"), true),
					mockednumericsensor.NewReporter(t).
						MockNumericSensorReport("C", 0, errors.New("test"), true),
					mockednumericmeter.NewReporter(t).
						MockMeterReport("W", 0, errors.New("test"), true),
				),
				Nodes: []*suite.Node{
					{
						Name:    "Controller error on mode report",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:thermostat/ad:2", "cmd.mode.get_report", "thermostat"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:thermostat/ad:2", "thermostat"),
						},
					},
					{
						Name:    "Non existent thing on mode report",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:thermostat/ad:3", "cmd.mode.get_report", "thermostat"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:thermostat/ad:3", "thermostat"),
						},
					},
					{
						Name:    "Controller error on setpoint report",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:thermostat/ad:2", "cmd.setpoint.get_report", "thermostat", "test_mode_a"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:thermostat/ad:2", "thermostat"),
						},
					},
					{
						Name:    "Non existent thing on setpoint report",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:thermostat/ad:3", "cmd.setpoint.get_report", "thermostat", "test_mode_a"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:thermostat/ad:3", "thermostat"),
						},
					},
					{
						Name:    "Unsupported mode on setpoint report",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:thermostat/ad:2", "cmd.setpoint.get_report", "thermostat", "unsupported_mode"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:thermostat/ad:2", "thermostat"),
						},
					},
					{
						Name:    "Wrong value type on setpoint report",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:thermostat/ad:2", "cmd.setpoint.get_report", "thermostat"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:thermostat/ad:2", "thermostat"),
						},
					},
					{
						Name:    "Controller error on state report",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:thermostat/ad:2", "cmd.state.get_report", "thermostat"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:thermostat/ad:2", "thermostat"),
						},
					},
					{
						Name:    "Non existent thing on state report",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:thermostat/ad:3", "cmd.state.get_report", "thermostat"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:thermostat/ad:3", "thermostat"),
						},
					},
					{
						Name:    "Reporter error on numeric sensor report",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:sensor_temp/ad:2", "cmd.sensor.get_report", "sensor_temp", "C"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:sensor_temp/ad:2", "sensor_temp"),
						},
					},
					{
						Name:    "Wrong value type on numeric sensor report",
						Command: suite.FloatMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:sensor_temp/ad:2", "cmd.sensor.get_report", "sensor_temp", 0),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:sensor_temp/ad:2", "sensor_temp"),
						},
					},
					{
						Name:    "Unsupported unit on numeric sensor report",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:sensor_temp/ad:2", "cmd.sensor.get_report", "sensor_temp", "F"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:sensor_temp/ad:2", "sensor_temp"),
						},
					},
					{
						Name:    "Non existent thing on numeric sensor report",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:sensor_temp/ad:3", "cmd.sensor.get_report", "sensor_temp"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:sensor_temp/ad:3", "sensor_temp"),
						},
					},
					{
						Name:    "Wrong sensor on numeric sensor report",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:sensor_temp/ad:2", "cmd.sensor.get_report", "sensor_wattemp"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:sensor_temp/ad:2", "sensor_wattemp"),
						},
					},
					{
						Name:    "Reporter error on electricity meter report",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "cmd.meter.get_report", "meter_elec", "W"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "meter_elec"),
						},
					},
					{
						Name:    "Wrong value type on electricity meter report",
						Command: suite.FloatMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "cmd.meter.get_report", "meter_elec", 0),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "meter_elec"),
						},
					},
					{
						Name:    "Unsupported unit on electricity meter report",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "cmd.meter.get_report", "meter_elec", "A"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "meter_elec"),
						},
					},
					{
						Name:    "Non existent thing on electricity meter report",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:3", "cmd.meter.get_report", "meter_elec", "W"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:3", "meter_elec"),
						},
					},
				},
			},
			{
				Name:     "Successful thermostat configuration",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeThermostat(
					mockedthermostat.NewController(t).
						MockSetThermostatMode("test_mode_c", nil, true).
						MockThermostatModeReport("test_mode_c", nil, true).
						MockSetThermostatMode("test_mode_a", nil, true).
						MockThermostatModeReport("test_mode_a", nil, true).
						MockThermostatSetpointReport("test_mode_a", 21, "C", nil, true).
						MockSetThermostatSetpoint("test_mode_a", 20, "C", nil, true).
						MockThermostatSetpointReport("test_mode_a", 20, "C", nil, true),
					nil, nil,
				),
				Nodes: []*suite.Node{
					{
						Name:    "Set mode not supporting a setpoint",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:thermostat/ad:2", "cmd.mode.set", "thermostat", "test_mode_c"),
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:thermostat/ad:2", "evt.mode.report", "thermostat", "test_mode_c"),
						},
					},
					{
						Name:    "Set mode supporting a setpoint",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:thermostat/ad:2", "cmd.mode.set", "thermostat", "test_mode_a"),
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:thermostat/ad:2", "evt.mode.report", "thermostat", "test_mode_a"),
							suite.ExpectStringMap("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:thermostat/ad:2", "evt.setpoint.report", "thermostat", map[string]string{"type": "test_mode_a", "temp": "21.0", "unit": "C"}),
						},
					},
					{
						Name:    "Set setpoint",
						Command: suite.StringMapMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:thermostat/ad:2", "cmd.setpoint.set", "thermostat", map[string]string{"type": "test_mode_a", "temp": "20.0", "unit": "C"}),
						Expectations: []*suite.Expectation{
							suite.ExpectStringMap("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:thermostat/ad:2", "evt.setpoint.report", "thermostat", map[string]string{"type": "test_mode_a", "temp": "20.0", "unit": "C"}),
						},
					},
				},
			},
			{
				Name:     "Failed thermostat configuration",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeThermostat(
					mockedthermostat.NewController(t).
						MockSetThermostatMode("test_mode_a", errors.New("test"), true).
						MockSetThermostatSetpoint("test_mode_a", 20, "C", errors.New("test"), true),
					nil, nil,
				),
				Nodes: []*suite.Node{
					{
						Name:    "Controller error when setting mode",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:thermostat/ad:2", "cmd.mode.set", "thermostat", "test_mode_a"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:thermostat/ad:2", "thermostat"),
						},
					},
					{
						Name:    "Unsupported mode",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:thermostat/ad:2", "cmd.mode.set", "thermostat", "unsupported_mode"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:thermostat/ad:2", "thermostat"),
						},
					},
					{
						Name:    "Setting mode with with wrong type of value",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:thermostat/ad:2", "cmd.mode.set", "thermostat"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:thermostat/ad:2", "thermostat"),
						},
					},
					{
						Name:    "Setting mode of non-existent thing",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:thermostat/ad:3", "cmd.mode.set", "thermostat", "test_mode_a"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:thermostat/ad:3", "thermostat"),
						},
					},
					{
						Name:    "Controller error when setting setpoint",
						Command: suite.StringMapMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:thermostat/ad:2", "cmd.setpoint.set", "thermostat", map[string]string{"type": "test_mode_a", "temp": "20.0", "unit": "C"}),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:thermostat/ad:2", "thermostat"),
						},
					},
					{
						Name:    "Unsupported mode when setting setpoint",
						Command: suite.StringMapMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:thermostat/ad:2", "cmd.setpoint.set", "thermostat", map[string]string{"type": "test_mode_c", "temp": "20.0", "unit": "C"}),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:thermostat/ad:2", "thermostat"),
						},
					},
					{
						Name:    "Missing unit when setting setpoint",
						Command: suite.StringMapMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:thermostat/ad:2", "cmd.setpoint.set", "thermostat", map[string]string{"type": "test_mode_a", "temp": "20.0"}),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:thermostat/ad:2", "thermostat"),
						},
					},
					{
						Name:    "Missing temperature when setting setpoint",
						Command: suite.StringMapMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:thermostat/ad:2", "cmd.setpoint.set", "thermostat", map[string]string{"type": "test_mode_a", "unit": "C"}),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:thermostat/ad:2", "thermostat"),
						},
					},
					{
						Name:    "Missing mode when setting setpoint",
						Command: suite.StringMapMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:thermostat/ad:2", "cmd.setpoint.set", "thermostat", map[string]string{"temp": "20.0", "unit": "C"}),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:thermostat/ad:2", "thermostat"),
						},
					},
					{
						Name:    "Invalid temperature when setting setpoint",
						Command: suite.StringMapMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:thermostat/ad:2", "cmd.setpoint.set", "thermostat", map[string]string{"type": "test_mode_a", "temp": "20C", "unit": "C"}),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:thermostat/ad:2", "thermostat"),
						},
					},
					{
						Name:    "Setting setpoint with wrong type of value",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:thermostat/ad:2", "cmd.setpoint.set", "thermostat", "22"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:thermostat/ad:2", "thermostat"),
						},
					},
					{
						Name:    "Setting setpoint of non-existent thing",
						Command: suite.StringMapMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:thermostat/ad:3", "cmd.setpoint.set", "thermostat", map[string]string{"type": "test_mode_a", "temp": "21.0", "unit": "C"}),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:thermostat/ad:3", "thermostat"),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func TestTaskThermostat(t *testing.T) { //nolint:paralleltest
	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "Thermostat tasks",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: taskThermostat(
					mockedthermostat.NewController(t).
						MockThermostatModeReport("test_mode_a", nil, true).
						MockThermostatModeReport("", errors.New("test"), true).
						MockThermostatModeReport("test_mode_a", nil, true).
						MockThermostatModeReport("test_mode_b", nil, false).
						MockThermostatSetpointReport("test_mode_a", 21, "C", nil, true).
						MockThermostatSetpointReport("test_mode_a", 0, "", errors.New("test"), true).
						MockThermostatSetpointReport("test_mode_a", 21, "C", nil, true).
						MockThermostatSetpointReport("test_mode_a", 21, "C", nil, false).
						MockThermostatSetpointReport("test_mode_b", 22, "C", nil, true).
						MockThermostatSetpointReport("test_mode_b", 0, "", errors.New("test"), true).
						MockThermostatSetpointReport("test_mode_b", 22, "C", nil, true).
						MockThermostatSetpointReport("test_mode_b", 23, "C", nil, false).
						MockThermostatStateReport("idle", nil, true).
						MockThermostatStateReport("", errors.New("test"), true).
						MockThermostatStateReport("idle", nil, true).
						MockThermostatStateReport("heat", nil, false),
					mockednumericsensor.NewReporter(t).
						MockNumericSensorReport("C", 21, nil, true).
						MockNumericSensorReport("C", 0, errors.New("test"), true).
						MockNumericSensorReport("C", 21, nil, true).
						MockNumericSensorReport("C", 21.5, nil, false),
					mockednumericmeter.NewReporter(t).
						MockMeterReport("W", 2, nil, true).
						MockMeterReport("W", 0, errors.New("test"), true).
						MockMeterReport("W", 2, nil, true).
						MockMeterReport("W", 1500, nil, false).
						MockMeterReport("kWh", 123.45, nil, true).
						MockMeterReport("kWh", 0, errors.New("test"), true).
						MockMeterReport("kWh", 123.45, nil, true).
						MockMeterReport("kWh", 123.56, nil, false),
					100*time.Millisecond,
				),
				Nodes: []*suite.Node{
					{
						Name: "One change and one error during three report cycles",
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:thermostat/ad:2", "evt.mode.report", "thermostat", "test_mode_a").ExactlyOnce(),
							suite.ExpectString("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:thermostat/ad:2", "evt.mode.report", "thermostat", "test_mode_b").ExactlyOnce(),
							suite.ExpectStringMap("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:thermostat/ad:2", "evt.setpoint.report", "thermostat", map[string]string{"type": "test_mode_a", "temp": "21.0", "unit": "C"}).ExactlyOnce(),
							suite.ExpectStringMap("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:thermostat/ad:2", "evt.setpoint.report", "thermostat", map[string]string{"type": "test_mode_b", "temp": "22.0", "unit": "C"}).ExactlyOnce(),
							suite.ExpectStringMap("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:thermostat/ad:2", "evt.setpoint.report", "thermostat", map[string]string{"type": "test_mode_b", "temp": "23.0", "unit": "C"}).ExactlyOnce(),
							suite.ExpectString("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:thermostat/ad:2", "evt.state.report", "thermostat", "idle").ExactlyOnce(),
							suite.ExpectString("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:thermostat/ad:2", "evt.state.report", "thermostat", "heat").ExactlyOnce(),
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:sensor_temp/ad:2", "evt.sensor.report", "sensor_temp", 21).ExpectProperty("unit", "C").ExactlyOnce(),
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:sensor_temp/ad:2", "evt.sensor.report", "sensor_temp", 21.5).ExpectProperty("unit", "C").ExactlyOnce(),
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "evt.meter.report", "meter_elec", 2).ExpectProperty("unit", "W").ExactlyOnce(),
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "evt.meter.report", "meter_elec", 123.45).ExpectProperty("unit", "kWh").ExactlyOnce(),
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "evt.meter.report", "meter_elec", 1500).ExpectProperty("unit", "W").ExactlyOnce(),
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "evt.meter.report", "meter_elec", 123.56).ExpectProperty("unit", "kWh").ExactlyOnce(),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func routeThermostat(
	thermostatController *mockedthermostat.Controller,
	sensorTempReporter *mockednumericsensor.Reporter,
	meterElecReporter *mockednumericmeter.Reporter,
) suite.BaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
		t.Helper()

		routing, _, mocks := setupThermostat(t, mqtt, thermostatController, sensorTempReporter, meterElecReporter, 0)

		return routing, nil, mocks
	}
}

func taskThermostat(
	thermostatController *mockedthermostat.Controller,
	sensorTempReporter *mockednumericsensor.Reporter,
	meterElecReporter *mockednumericmeter.Reporter,
	interval time.Duration,
) suite.BaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
		t.Helper()

		_, tasks, mocks := setupThermostat(t, mqtt, thermostatController, sensorTempReporter, meterElecReporter, interval)

		return nil, tasks, mocks
	}
}

func setupThermostat(
	t *testing.T,
	mqtt *fimpgo.MqttTransport,
	thermostatController *mockedthermostat.Controller,
	sensorTempReporter *mockednumericsensor.Reporter,
	meterElecReporter *mockednumericmeter.Reporter,
	duration time.Duration,
) ([]*router.Routing, []*task.Task, []suite.Mock) {
	t.Helper()

	mocks := []suite.Mock{thermostatController}

	cfg := &thing.ThermostatConfig{
		ThingConfig: &adapter.ThingConfig{
			InclusionReport: &fimptype.ThingInclusionReport{
				Address: "2",
			},
			Connector: mockedadapter.NewDefaultConnector(t),
		},
		ThermostatConfig: &thermostat.Config{
			Specification: thermostat.Specification(
				"test_adapter",
				"1",
				"2",
				nil,
				[]string{"test_mode_a", "test_mode_b", "test_mode_c", "off"},
				[]string{"test_mode_a", "test_mode_b"},
				[]string{"heat", "idle"},
			),
			Controller: thermostatController,
		},
	}

	if sensorTempReporter != nil {
		cfg.SensorTempConfig = &numericsensor.Config{
			Specification: numericsensor.Specification(
				"test_adapter",
				"1",
				numericsensor.SensorTemp,
				"2",
				nil,
				[]string{"C"},
			),
			Reporter: sensorTempReporter,
		}

		mocks = append(mocks, sensorTempReporter)
	}

	if meterElecReporter != nil {
		cfg.MeterElecConfig = &numericmeter.Config{
			Specification: numericmeter.Specification(
				numericmeter.MeterElec,
				"test_adapter",
				"1",
				"2",
				nil,
				[]numericmeter.Unit{"W", "kWh"},
			),
			Reporter: meterElecReporter,
		}

		mocks = append(mocks, meterElecReporter)
	}

	seed := &adapter.ThingSeed{ID: "B", CustomAddress: "2"}

	factory := adapterhelper.FactoryHelper(func(adapter adapter.Adapter, publisher adapter.Publisher, thingState adapter.ThingState) (adapter.Thing, error) {
		return thing.NewThermostat(publisher, thingState, cfg), nil
	})

	ad := adapterhelper.PrepareSeededAdapter(t, "../../testdata/adapter/test_adapter", mqtt, factory, adapter.ThingSeeds{seed})

	return thing.RouteThermostat(ad), thing.TaskThermostat(ad, duration), mocks
}
