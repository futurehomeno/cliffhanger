package numericsensor_test

import (
	"testing"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/numericsensor"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
	adapterhelper "github.com/futurehomeno/cliffhanger/test/helper/adapter"
	mockedadapter "github.com/futurehomeno/cliffhanger/test/mocks/adapter"
	mockednumericsensor "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/numericsensor"
	cliffSuite "github.com/futurehomeno/cliffhanger/test/suite"
)

func TestTaskReporting(t *testing.T) { //nolint:paralleltest
	s := &cliffSuite.Suite{
		Cases: []*cliffSuite.Case{
			{
				Name:     "sensor periodic reporting",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: taskSensor(
					mockednumericsensor.NewReporter(t).
						MockNumericSensorReport(numericsensor.UnitC, 21.5, nil, true).
						MockNumericSensorReport(numericsensor.UnitF, 70.7, nil, true).
						MockNumericSensorReport(numericsensor.UnitC, 22.0, nil, true).
						MockNumericSensorReport(numericsensor.UnitF, 71.6, nil, true),
					100*time.Millisecond,
				),
				Nodes: []*cliffSuite.Node{
					{
						Name: "two cycles with changing values emit one report each",
						Expectations: []*cliffSuite.Expectation{
							cliffSuite.ExpectFloat(sensorEvtTopic, numericsensor.EvtSensorReport, sensorService, 21.5).ExactlyOnce(),
							cliffSuite.ExpectFloat(sensorEvtTopic, numericsensor.EvtSensorReport, sensorService, 70.7).ExactlyOnce(),
							cliffSuite.ExpectFloat(sensorEvtTopic, numericsensor.EvtSensorReport, sensorService, 22.0).ExactlyOnce(),
							cliffSuite.ExpectFloat(sensorEvtTopic, numericsensor.EvtSensorReport, sensorService, 71.6).ExactlyOnce(),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func taskSensor(reporter *mockednumericsensor.Reporter, interval time.Duration) cliffSuite.BaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []cliffSuite.Mock) {
		t.Helper()

		thingCfg := &adapter.ThingConfig{
			InclusionReport: &fimptype.ThingInclusionReport{Address: "2"},
			Connector:       mockedadapter.NewDefaultConnector(t),
		}

		svcCfg := &numericsensor.Config{
			Specification: numericsensor.Specification(
				"test_adapter",
				"1",
				sensorService,
				"2",
				nil,
				[]string{numericsensor.UnitC, numericsensor.UnitF},
			),
			Reporter: reporter,
		}

		seed := &adapter.ThingSeed{ID: "B", CustomAddress: "2"}

		factory := adapterhelper.FactoryHelper(func(a adapter.Adapter, p adapter.Publisher, ts adapter.ThingState) (adapter.Thing, error) {
			return adapter.NewThing(p, ts, thingCfg, numericsensor.NewService(p, svcCfg)), nil
		})

		ad := adapterhelper.PrepareSeededAdapter(t, "../../testdata/adapter/test_adapter", mqtt, factory, adapter.ThingSeeds{seed})

		return nil, []*task.Task{numericsensor.TaskReporting(ad, interval)}, nil
	}
}
