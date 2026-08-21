package waterheater_test

import (
	"testing"
	"time"

	"github.com/futurehomeno/fimpgo"

	"github.com/futurehomeno/cliffhanger/adapter/service/waterheater"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
	adapterhelper "github.com/futurehomeno/cliffhanger/test/helper/adapter"
	mockedwaterheater "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/waterheater"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

func TestTaskReporting(t *testing.T) { //nolint:paralleltest
	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "Water heater tasks",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: taskWaterHeater(
					mockedwaterheater.NewController(t).
						MockWaterHeaterModeReport("heat", nil, true).
						MockWaterHeaterModeReport("", errTest, true).
						MockWaterHeaterModeReport("heat", nil, true).
						MockWaterHeaterModeReport("eco", nil, true). // should be sent twice
						MockWaterHeaterSetpointReport("heat", 65, "C", nil, true).
						MockWaterHeaterSetpointReport("heat", 0, "", errTest, true).
						MockWaterHeaterSetpointReport("heat", 65, "C", nil, true).
						MockWaterHeaterSetpointReport("heat", 70, "C", nil, true). // should be sent twice
						MockWaterHeaterSetpointReport("eco", 45, "C", nil, true).
						MockWaterHeaterSetpointReport("eco", 0, "", errTest, true).
						MockWaterHeaterSetpointReport("eco", 45, "C", nil, true).
						MockWaterHeaterSetpointReport("eco", 50, "C", nil, true). // should be sent twice
						MockWaterHeaterStateReport("idle", nil, true).
						MockWaterHeaterStateReport("", errTest, true).
						MockWaterHeaterStateReport("idle", nil, true).
						MockWaterHeaterStateReport("heat", nil, true), // should be sent twice
					testModes, testSetpoints, testStates, testRange, nil, testStep,
					100*time.Millisecond,
				),
				Nodes: []*suite.Node{
					{
						Name: "One change and one error during four report cycles",
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "evt.mode.report", "water_heater", "heat").ExactlyOnce(),
							suite.ExpectString("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "evt.mode.report", "water_heater", "eco").ExactlyOnce(),
							suite.ExpectObject("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "evt.setpoint.report", "water_heater", waterheater.NewSetpoint("heat", 65, "C")).ExactlyOnce(),
							suite.ExpectObject("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "evt.setpoint.report", "water_heater", waterheater.NewSetpoint("heat", 70, "C")).ExactlyOnce(),
							suite.ExpectObject("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "evt.setpoint.report", "water_heater", waterheater.NewSetpoint("eco", 45, "C")).ExactlyOnce(),
							suite.ExpectObject("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "evt.setpoint.report", "water_heater", waterheater.NewSetpoint("eco", 50, "C")).ExactlyOnce(),
							suite.ExpectString("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "evt.state.report", "water_heater", "idle").ExactlyOnce(),
							suite.ExpectString("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:water_heater/ad:2", "evt.state.report", "water_heater", "heat").ExactlyOnce(),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func taskWaterHeater(
	controller waterheater.Controller,
	modes, setpoints, states []string,
	supportedRange *waterheater.Range,
	supportedRanges map[string]waterheater.Range,
	step float64,
	interval time.Duration,
) suite.BaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
		t.Helper()

		_, tasks, mocks := setupService(t, mqtt, controller, modes, setpoints, states, supportedRange, supportedRanges, step, interval)

		return nil, tasks, mocks
	}
}
