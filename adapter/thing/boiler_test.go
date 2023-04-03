package thing_test

import (
	"errors"
	"testing"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/meterelec"
	"github.com/futurehomeno/cliffhanger/adapter/service/numericsensor"
	"github.com/futurehomeno/cliffhanger/adapter/service/waterheater"
	"github.com/futurehomeno/cliffhanger/adapter/thing"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
	adapterhelper "github.com/futurehomeno/cliffhanger/test/helper/adapter"
	mockedadapter "github.com/futurehomeno/cliffhanger/test/mocks/adapter"
	mockedmeterelec "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/meterelec"
	mockednumericsensor "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/numericsensor"
	mockedwaterheater "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/waterheater"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

func TestRouteBoiler(t *testing.T) { //nolint:paralleltest
	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "Successful boiler reporting",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeBoiler(
					mockedwaterheater.NewController(t).
						MockWaterHeaterModeReport("test_mode_a", nil, true).
						MockWaterHeaterSetpointReport("test_mode_a", 60, "C", nil, true).
						MockWaterHeaterStateReport("idle", nil, true),
					mockednumericsensor.NewReporter(t).
						MockNumericSensorReport("C", 60, nil, false),
					mockedmeterelec.NewReporter(t).
						MockElectricityMeterReport("W", 1500, nil, false).
						MockElectricityMeterReport("kWh", 31.5, nil, false),
				),
				Nodes: []*suite.Node{
					{
						Name:    "Mode",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "cmd.mode.get_report", "water_heater"),
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "evt.mode.report", "water_heater", "test_mode_a"),
						},
					},
					{
						Name:    "Setpoint",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "cmd.setpoint.get_report", "water_heater", "test_mode_a"),
						Expectations: []*suite.Expectation{
							suite.ExpectObject("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "evt.setpoint.report", "water_heater", waterheater.Setpoint{Type: "test_mode_a", Temperature: 60, Unit: "C"}),
						},
					},
					{
						Name:    "State",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "cmd.state.get_report", "water_heater"),
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "evt.state.report", "water_heater", "idle"),
						},
					},
					{
						Name:    "Temperature",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:sensor_wattemp/ad:2", "cmd.sensor.get_report", "sensor_wattemp", "C"),
						Expectations: []*suite.Expectation{
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:sensor_wattemp/ad:2", "evt.sensor.report", "sensor_wattemp", 60).
								ExpectProperty("unit", "C"),
						},
					},
					{
						Name:    "All sensor units",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:sensor_wattemp/ad:2", "cmd.sensor.get_report", "sensor_wattemp", ""),
						Expectations: []*suite.Expectation{
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:sensor_wattemp/ad:2", "evt.sensor.report", "sensor_wattemp", 60).
								ExpectProperty("unit", "C"),
						},
					},
					{
						Name:    "All sensor units with null",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:sensor_wattemp/ad:2", "cmd.sensor.get_report", "sensor_wattemp"),
						Expectations: []*suite.Expectation{
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:sensor_wattemp/ad:2", "evt.sensor.report", "sensor_wattemp", 60).
								ExpectProperty("unit", "C"),
						},
					},
					{
						Name:    "Power",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "cmd.meter.get_report", "meter_elec", "W"),
						Expectations: []*suite.Expectation{
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "evt.meter.report", "meter_elec", 1500).
								ExpectProperty("unit", "W"),
						},
					},
					{
						Name:    "Energy",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "cmd.meter.get_report", "meter_elec", "kWh"),
						Expectations: []*suite.Expectation{
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "evt.meter.report", "meter_elec", 31.5).
								ExpectProperty("unit", "kWh"),
						},
					},
					{
						Name:    "All electricity units",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "cmd.meter.get_report", "meter_elec", ""),
						Expectations: []*suite.Expectation{
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "evt.meter.report", "meter_elec", 1500).
								ExpectProperty("unit", "W"),
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "evt.meter.report", "meter_elec", 31.5).
								ExpectProperty("unit", "kWh"),
						},
					},
					{
						Name:    "All electricity units with null",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "cmd.meter.get_report", "meter_elec"),
						Expectations: []*suite.Expectation{
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "evt.meter.report", "meter_elec", 1500).
								ExpectProperty("unit", "W"),
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "evt.meter.report", "meter_elec", 31.5).
								ExpectProperty("unit", "kWh"),
						},
					},
				},
			},
			{
				Name:     "Failed boiler reporting",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeBoiler(
					mockedwaterheater.NewController(t).
						MockWaterHeaterModeReport("test_mode_a", errors.New("test"), true).
						MockWaterHeaterSetpointReport("test_mode_a", 0, "", errors.New("test"), true).
						MockWaterHeaterStateReport("", errors.New("test"), true),
					mockednumericsensor.NewReporter(t).
						MockNumericSensorReport("C", 0, errors.New("test"), true),
					mockedmeterelec.NewReporter(t).
						MockElectricityMeterReport("W", 0, errors.New("test"), true),
				),
				Nodes: []*suite.Node{
					{
						Name:    "Controller error on mode report",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "cmd.mode.get_report", "water_heater"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "water_heater"),
						},
					},
					{
						Name:    "Non existent thing on mode report",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:3", "cmd.mode.get_report", "water_heater"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:3", "water_heater"),
						},
					},
					{
						Name:    "Controller error on setpoint report",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "cmd.setpoint.get_report", "water_heater", "test_mode_a"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "water_heater"),
						},
					},
					{
						Name:    "Non existent thing on setpoint report",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:3", "cmd.setpoint.get_report", "water_heater", "test_mode_a"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:3", "water_heater"),
						},
					},
					{
						Name:    "Unsupported mode on setpoint report",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "cmd.setpoint.get_report", "water_heater", "unsupported_mode"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "water_heater"),
						},
					},
					{
						Name:    "Wrong value type on setpoint report",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "cmd.setpoint.get_report", "water_heater"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "water_heater"),
						},
					},
					{
						Name:    "Controller error on state report",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "cmd.state.get_report", "water_heater"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "water_heater"),
						},
					},
					{
						Name:    "Non existent thing on state report",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:3", "cmd.state.get_report", "water_heater"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:3", "water_heater"),
						},
					},
					{
						Name:    "Reporter error on numeric sensor report",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:sensor_wattemp/ad:2", "cmd.sensor.get_report", "sensor_wattemp", "C"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:sensor_wattemp/ad:2", "sensor_wattemp"),
						},
					},
					{
						Name:    "Wrong value type on numeric sensor report",
						Command: suite.FloatMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:sensor_wattemp/ad:2", "cmd.sensor.get_report", "sensor_wattemp", 0),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:sensor_wattemp/ad:2", "sensor_wattemp"),
						},
					},
					{
						Name:    "Unsupported unit on numeric sensor report",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:sensor_wattemp/ad:2", "cmd.sensor.get_report", "sensor_wattemp", "F"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:sensor_wattemp/ad:2", "sensor_wattemp"),
						},
					},
					{
						Name:    "Non existent thing on numeric sensor report",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:sensor_wattemp/ad:3", "cmd.sensor.get_report", "sensor_wattemp"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:sensor_wattemp/ad:3", "sensor_wattemp"),
						},
					},
					{
						Name:    "Wrong sensor on numeric sensor report",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:sensor_wattemp/ad:2", "cmd.sensor.get_report", "sensor_temp"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:sensor_wattemp/ad:2", "sensor_temp"),
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
				Name:     "Successful boiler configuration",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeBoiler(
					mockedwaterheater.NewController(t).
						MockSetWaterHeaterMode("test_mode_c", nil, true).
						MockWaterHeaterModeReport("test_mode_c", nil, true).
						MockSetWaterHeaterMode("test_mode_a", nil, true).
						MockWaterHeaterModeReport("test_mode_a", nil, true).
						MockWaterHeaterSetpointReport("test_mode_a", 60, "C", nil, true).
						MockSetWaterHeaterSetpoint("test_mode_a", 70, "C", nil, true).
						MockWaterHeaterSetpointReport("test_mode_a", 70, "C", nil, true),
					nil, nil,
				),
				Nodes: []*suite.Node{
					{
						Name:    "Set mode not supporting a setpoint",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "cmd.mode.set", "water_heater", "test_mode_c"),
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "evt.mode.report", "water_heater", "test_mode_c"),
						},
					},
					{
						Name:    "Set mode supporting a setpoint",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "cmd.mode.set", "water_heater", "test_mode_a"),
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "evt.mode.report", "water_heater", "test_mode_a"),
							suite.ExpectObject("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "evt.setpoint.report", "water_heater", waterheater.Setpoint{Type: "test_mode_a", Temperature: 60, Unit: "C"}),
						},
					},
					{
						Name:    "Set setpoint",
						Command: suite.ObjectMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "cmd.setpoint.set", "water_heater", waterheater.Setpoint{Type: "test_mode_a", Temperature: 70, Unit: "C"}),
						Expectations: []*suite.Expectation{
							suite.ExpectObject("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "evt.setpoint.report", "water_heater", waterheater.Setpoint{Type: "test_mode_a", Temperature: 70, Unit: "C"}),
						},
					},
				},
			},
			{
				Name:     "Failed boiler configuration",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeBoiler(
					mockedwaterheater.NewController(t).
						MockSetWaterHeaterMode("test_mode_a", errors.New("test"), true).
						MockSetWaterHeaterSetpoint("test_mode_a", 70, "C", errors.New("test"), true),
					nil, nil,
				),
				Nodes: []*suite.Node{
					{
						Name:    "Controller error when setting mode",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "cmd.mode.set", "water_heater", "test_mode_a"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "water_heater"),
						},
					},
					{
						Name:    "Unsupported mode",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "cmd.mode.set", "water_heater", "unsupported_mode"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "water_heater"),
						},
					},
					{
						Name:    "Setting mode with with wrong type of value",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "cmd.mode.set", "water_heater"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "water_heater"),
						},
					},
					{
						Name:    "Setting mode of non-existent thing",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:3", "cmd.mode.set", "water_heater", "test_mode_a"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:3", "water_heater"),
						},
					},
					{
						Name:    "Controller error when setting setpoint",
						Command: suite.ObjectMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "cmd.setpoint.set", "water_heater", waterheater.Setpoint{Type: "test_mode_a", Temperature: 70, Unit: "C"}),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "water_heater"),
						},
					},
					{
						Name:    "Setpoint out of specific range",
						Command: suite.ObjectMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "cmd.setpoint.set", "water_heater", waterheater.Setpoint{Type: "test_mode_a", Temperature: 80, Unit: "C"}),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "water_heater"),
						},
					},
					{
						Name:    "Setpoint out of generic range",
						Command: suite.ObjectMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "cmd.setpoint.set", "water_heater", waterheater.Setpoint{Type: "test_mode_b", Temperature: 85, Unit: "C"}),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "water_heater"),
						},
					},
					{
						Name:    "Setpoint unsupported by mode",
						Command: suite.ObjectMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "cmd.setpoint.set", "water_heater", waterheater.Setpoint{Type: "test_mode_c", Temperature: 60, Unit: "C"}),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "water_heater"),
						},
					},
					{
						Name:    "Setting setpoint with with wrong type of value",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "cmd.setpoint.set", "water_heater", "60"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "water_heater"),
						},
					},
					{
						Name:    "Setting setpoint of non-existent thing",
						Command: suite.ObjectMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:3", "cmd.setpoint.set", "water_heater", waterheater.Setpoint{Type: "test_mode_c", Temperature: 85, Unit: "C"}),
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

func TestTaskBoiler(t *testing.T) { //nolint:paralleltest
	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "Boiler tasks",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: taskBoiler(
					mockedwaterheater.NewController(t).
						MockWaterHeaterModeReport("test_mode_a", nil, true).
						MockWaterHeaterModeReport("", errors.New("test"), true).
						MockWaterHeaterModeReport("test_mode_a", nil, true).
						MockWaterHeaterModeReport("test_mode_b", nil, false).
						MockWaterHeaterSetpointReport("test_mode_a", 60, "C", nil, true).
						MockWaterHeaterSetpointReport("test_mode_a", 0, "", errors.New("test"), true).
						MockWaterHeaterSetpointReport("test_mode_a", 60, "C", nil, true).
						MockWaterHeaterSetpointReport("test_mode_a", 60, "C", nil, false).
						MockWaterHeaterSetpointReport("test_mode_b", 70, "C", nil, true).
						MockWaterHeaterSetpointReport("test_mode_b", 0, "", errors.New("test"), true).
						MockWaterHeaterSetpointReport("test_mode_b", 70, "C", nil, true).
						MockWaterHeaterSetpointReport("test_mode_b", 71, "C", nil, false).
						MockWaterHeaterStateReport("idle", nil, true).
						MockWaterHeaterStateReport("", errors.New("test"), true).
						MockWaterHeaterStateReport("idle", nil, true).
						MockWaterHeaterStateReport("heat", nil, false),
					mockednumericsensor.NewReporter(t).
						MockNumericSensorReport("C", 60, nil, true).
						MockNumericSensorReport("C", 0, errors.New("test"), true).
						MockNumericSensorReport("C", 60, nil, true).
						MockNumericSensorReport("C", 60.5, nil, false),
					mockedmeterelec.NewReporter(t).
						MockElectricityMeterReport("W", 2, nil, true).
						MockElectricityMeterReport("W", 0, errors.New("test"), true).
						MockElectricityMeterReport("W", 2, nil, true).
						MockElectricityMeterReport("W", 1500, nil, false).
						MockElectricityMeterReport("kWh", 31.5, nil, true).
						MockElectricityMeterReport("kWh", 0, errors.New("test"), true).
						MockElectricityMeterReport("kWh", 31.5, nil, true).
						MockElectricityMeterReport("kWh", 31.6, nil, false),
					100*time.Millisecond,
				),
				Nodes: []*suite.Node{
					{
						Name: "One change and one error during three report cycles",
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "evt.mode.report", "water_heater", "test_mode_a").ExactlyOnce(),
							suite.ExpectString("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "evt.mode.report", "water_heater", "test_mode_b").ExactlyOnce(),
							suite.ExpectObject("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "evt.setpoint.report", "water_heater", waterheater.Setpoint{Type: "test_mode_a", Temperature: 60, Unit: "C"}).ExactlyOnce(),
							suite.ExpectObject("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "evt.setpoint.report", "water_heater", waterheater.Setpoint{Type: "test_mode_b", Temperature: 70, Unit: "C"}).ExactlyOnce(),
							suite.ExpectObject("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "evt.setpoint.report", "water_heater", waterheater.Setpoint{Type: "test_mode_b", Temperature: 71, Unit: "C"}).ExactlyOnce(),
							suite.ExpectString("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "evt.state.report", "water_heater", "idle").ExactlyOnce(),
							suite.ExpectString("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "evt.state.report", "water_heater", "heat").ExactlyOnce(),
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:sensor_wattemp/ad:2", "evt.sensor.report", "sensor_wattemp", 60).ExpectProperty("unit", "C").ExactlyOnce(),
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:sensor_wattemp/ad:2", "evt.sensor.report", "sensor_wattemp", 60.5).ExpectProperty("unit", "C").ExactlyOnce(),
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "evt.meter.report", "meter_elec", 2).ExpectProperty("unit", "W").ExactlyOnce(),
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "evt.meter.report", "meter_elec", 31.5).ExpectProperty("unit", "kWh").ExactlyOnce(),
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "evt.meter.report", "meter_elec", 1500).ExpectProperty("unit", "W").ExactlyOnce(),
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "evt.meter.report", "meter_elec", 31.6).ExpectProperty("unit", "kWh").ExactlyOnce(),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func routeBoiler(
	waterHeaterController *mockedwaterheater.Controller,
	sensorWatTempReporter *mockednumericsensor.Reporter,
	meterElecReporter *mockedmeterelec.Reporter,
) suite.BaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
		t.Helper()

		routing, _, mocks := setupBoiler(t, mqtt, waterHeaterController, sensorWatTempReporter, meterElecReporter, 0)

		return routing, nil, mocks
	}
}

func taskBoiler(
	waterHeaterController *mockedwaterheater.Controller,
	sensorWatTempReporter *mockednumericsensor.Reporter,
	meterElecReporter *mockedmeterelec.Reporter,
	interval time.Duration,
) suite.BaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
		t.Helper()

		_, tasks, mocks := setupBoiler(t, mqtt, waterHeaterController, sensorWatTempReporter, meterElecReporter, interval)

		return nil, tasks, mocks
	}
}

func setupBoiler(
	t *testing.T,
	mqtt *fimpgo.MqttTransport,
	waterHeaterController *mockedwaterheater.Controller,
	sensorWatTempReporter *mockednumericsensor.Reporter,
	meterElecReporter *mockedmeterelec.Reporter,
	duration time.Duration,
) ([]*router.Routing, []*task.Task, []suite.Mock) {
	t.Helper()

	mocks := []suite.Mock{waterHeaterController}

	cfg := &thing.BoilerConfig{
		ThingConfig: &adapter.ThingConfig{
			InclusionReport: &fimptype.ThingInclusionReport{
				Address: "2",
			},
			Connector: mockedadapter.NewConnector(t),
		},
		WaterHeaterConfig: &waterheater.Config{
			Specification: waterheater.Specification(
				"test_adapter",
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

	if sensorWatTempReporter != nil {
		cfg.SensorWatTempConfig = &numericsensor.Config{
			Specification: numericsensor.Specification(
				"test_adapter",
				"1",
				numericsensor.SensorWatTemp,
				"2",
				nil,
				[]string{"C"},
			),
			Reporter: sensorWatTempReporter,
		}

		mocks = append(mocks, sensorWatTempReporter)
	}

	if meterElecReporter != nil {
		cfg.MeterElecConfig = &meterelec.Config{
			Specification: meterelec.Specification(
				"test_adapter",
				"1",
				"2",
				nil,
				[]string{"W", "kWh"},
				nil,
			),
			Reporter: meterElecReporter,
		}

		mocks = append(mocks, meterElecReporter)
	}

	seed := &adapter.ThingSeed{ID: "B", CustomAddress: "2"}

	factory := adapterhelper.FactoryHelper(func(adapter adapter.Adapter, publisher adapter.Publisher, thingState adapter.ThingState) (adapter.Thing, error) {
		return thing.NewBoiler(publisher, thingState, cfg), nil
	})

	ad := adapterhelper.PrepareSeededAdapter(t, "../../testdata/adapter/test_adapter", mqtt, factory, adapter.ThingSeeds{seed})

	return thing.RouteBoiler(ad), thing.TaskBoiler(ad, duration), mocks
}
