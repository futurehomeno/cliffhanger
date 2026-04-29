package thermostat_test

import (
	"testing"
	"time"

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

func TestTaskReporting(t *testing.T) { //nolint:paralleltest
	s := &cliffSuite.Suite{
		Cases: []*cliffSuite.Case{
			{
				Name:     "thermostat periodic reporting",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: taskThermostat(
					mockedthermostat.NewController(t).
						MockThermostatModeReport("heat", nil, true).
						MockThermostatModeReport(thermostat.ModeOff, nil, true).
						MockThermostatSetpointReport("heat", 21.0, thermostat.UnitC, nil, true).
						MockThermostatSetpointReport("heat", 22.0, thermostat.UnitC, nil, true).
						MockThermostatStateReport(thermostat.StateHeat, nil, true).
						MockThermostatStateReport(thermostat.StateIdle, nil, true),
					100*time.Millisecond,
				),
				Nodes: []*cliffSuite.Node{
					{
						Name: "two cycles with changing values each emit one report",
						Expectations: []*cliffSuite.Expectation{
							cliffSuite.ExpectString(evtTopic, thermostat.EvtModeReport, thermostat.Thermostat, "heat").ExactlyOnce(),
							cliffSuite.ExpectString(evtTopic, thermostat.EvtModeReport, thermostat.Thermostat, thermostat.ModeOff).ExactlyOnce(),
							cliffSuite.ExpectStringMap(evtTopic, thermostat.EvtSetpointReport, thermostat.Thermostat,
								map[string]string{"type": "heat", "temp": "21.0", "unit": thermostat.UnitC}).ExactlyOnce(),
							cliffSuite.ExpectStringMap(evtTopic, thermostat.EvtSetpointReport, thermostat.Thermostat,
								map[string]string{"type": "heat", "temp": "22.0", "unit": thermostat.UnitC}).ExactlyOnce(),
							cliffSuite.ExpectString(evtTopic, thermostat.EvtStateReport, thermostat.Thermostat, thermostat.StateHeat).ExactlyOnce(),
							cliffSuite.ExpectString(evtTopic, thermostat.EvtStateReport, thermostat.Thermostat, thermostat.StateIdle).ExactlyOnce(),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func taskThermostat(controller *mockedthermostat.Controller, interval time.Duration) cliffSuite.BaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []cliffSuite.Mock) {
		t.Helper()

		thingCfg := &adapter.ThingConfig{
			InclusionReport: &fimptype.ThingInclusionReport{Address: "2"},
			Connector:       mockedadapter.NewDefaultConnector(t),
		}

		svcCfg := &thermostat.Config{
			Specification: thermostat.Specification(
				"test_adapter", "1", "2", nil,
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

		return nil, []*task.Task{thermostat.TaskReporting(ad, interval)}, nil
	}
}
