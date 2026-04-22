package numericmeter_test

import (
	"testing"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/numericmeter"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
	adapterhelper "github.com/futurehomeno/cliffhanger/test/helper/adapter"
	mockedadapter "github.com/futurehomeno/cliffhanger/test/mocks/adapter"
	mockednumericmeter "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/numericmeter"
	cliffSuite "github.com/futurehomeno/cliffhanger/test/suite"
)

func TestTaskReporting(t *testing.T) { //nolint:paralleltest
	s := &cliffSuite.Suite{
		Cases: []*cliffSuite.Case{
			{
				Name:     "meter periodic reporting",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: taskMeter(
					mockednumericmeter.NewReporter(t).
						MockMeterReport(numericmeter.UnitW, 100.0, nil, true).
						MockMeterReport(numericmeter.UnitW, 200.0, nil, true).
						MockMeterReport(numericmeter.UnitKWh, 1.0, nil, true).
						MockMeterReport(numericmeter.UnitKWh, 2.0, nil, true),
					100*time.Millisecond,
				),
				Nodes: []*cliffSuite.Node{
					{
						Name: "two cycles with changing values emit one report each",
						Expectations: []*cliffSuite.Expectation{
							cliffSuite.ExpectFloat(meterEvtTopic, numericmeter.EvtMeterReport, numericmeter.MeterElec, 100.0).ExactlyOnce(),
							cliffSuite.ExpectFloat(meterEvtTopic, numericmeter.EvtMeterReport, numericmeter.MeterElec, 200.0).ExactlyOnce(),
							cliffSuite.ExpectFloat(meterEvtTopic, numericmeter.EvtMeterReport, numericmeter.MeterElec, 1.0).ExactlyOnce(),
							cliffSuite.ExpectFloat(meterEvtTopic, numericmeter.EvtMeterReport, numericmeter.MeterElec, 2.0).ExactlyOnce(),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func taskMeter(reporter *mockednumericmeter.Reporter, interval time.Duration) cliffSuite.BaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []cliffSuite.Mock) {
		t.Helper()

		thingCfg := &adapter.ThingConfig{
			InclusionReport: &fimptype.ThingInclusionReport{Address: "2"},
			Connector:       mockedadapter.NewDefaultConnector(t),
		}

		svcCfg := &numericmeter.Config{
			Specification: numericmeter.Specification(
				numericmeter.MeterElec,
				"test_adapter",
				"1",
				"2",
				nil,
				numericmeter.Units{numericmeter.UnitW, numericmeter.UnitKWh},
			),
			Reporter: reporter,
		}

		seed := &adapter.ThingSeed{ID: "B", CustomAddress: "2"}

		factory := adapterhelper.FactoryHelper(func(a adapter.Adapter, p adapter.Publisher, ts adapter.ThingState) (adapter.Thing, error) {
			return adapter.NewThing(p, ts, thingCfg, numericmeter.NewService(p, svcCfg)), nil
		})

		ad := adapterhelper.PrepareSeededAdapter(t, "../../testdata/adapter/test_adapter", mqtt, factory, adapter.ThingSeeds{seed})

		return nil, []*task.Task{numericmeter.TaskReporting(ad, interval)}, nil
	}
}
