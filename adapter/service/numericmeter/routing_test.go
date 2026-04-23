package numericmeter_test

import (
	"errors"
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

const (
	meterEvtTopic = "pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2"
	meterCmdTopic = "pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2"
)

// resettableMeter combines Reporter and ResettableReporter for tests.
type resettableMeter struct {
	*mockednumericmeter.Reporter
	*mockednumericmeter.ResettableReporter
}

func TestRouteService(t *testing.T) { //nolint:paralleltest
	s := &cliffSuite.Suite{
		Cases: []*cliffSuite.Case{
			{
				Name:     "meter get report all units",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeMeter(mockednumericmeter.NewReporter(t).
					MockMeterReport(numericmeter.UnitW, 100.0, nil, true).
					MockMeterReport(numericmeter.UnitKWh, 5.5, nil, true),
				),
				Nodes: []*cliffSuite.Node{
					{
						Name: "null payload reports all units",
						Command: cliffSuite.NewMessageBuilder().
							NullMessage(meterCmdTopic, numericmeter.CmdMeterGetReport, numericmeter.MeterElec).
							Build(),
						Expectations: []*cliffSuite.Expectation{
							cliffSuite.ExpectFloat(meterEvtTopic, numericmeter.EvtMeterReport, numericmeter.MeterElec, 100.0),
							cliffSuite.ExpectFloat(meterEvtTopic, numericmeter.EvtMeterReport, numericmeter.MeterElec, 5.5),
						},
					},
				},
			},
			{
				Name:     "meter get report specific unit",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeMeter(mockednumericmeter.NewReporter(t).
					MockMeterReport(numericmeter.UnitW, 100.0, nil, true),
				),
				Nodes: []*cliffSuite.Node{
					{
						Name: "string payload reports requested unit",
						Command: cliffSuite.NewMessageBuilder().
							StringMessage(meterCmdTopic, numericmeter.CmdMeterGetReport, numericmeter.MeterElec, numericmeter.UnitW.String()).
							Build(),
						Expectations: []*cliffSuite.Expectation{
							cliffSuite.ExpectFloat(meterEvtTopic, numericmeter.EvtMeterReport, numericmeter.MeterElec, 100.0),
						},
					},
				},
			},
			{
				Name:     "meter get report controller error",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeMeter(mockednumericmeter.NewReporter(t).
					MockMeterReport(numericmeter.UnitW, 0, errors.New("controller error"), true),
				),
				Nodes: []*cliffSuite.Node{
					{
						Name: "controller error returns error event",
						Command: cliffSuite.NewMessageBuilder().
							StringMessage(meterCmdTopic, numericmeter.CmdMeterGetReport, numericmeter.MeterElec, numericmeter.UnitW.String()).
							Build(),
						Expectations: []*cliffSuite.Expectation{
							cliffSuite.ExpectError(meterEvtTopic, numericmeter.MeterElec),
						},
					},
				},
			},
			{
				Name:     "meter reset success",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeResettableMeter(
					mockednumericmeter.NewReporter(t),
					func() *mockednumericmeter.ResettableReporter {
						r := mockednumericmeter.NewResettableReporter(t)
						r.On("MeterReset").Return(nil).Once()
						return r
					}(),
				),
				Nodes: []*cliffSuite.Node{
					{
						Name: "cmd.meter.reset calls reset with no reply",
						Command: cliffSuite.NewMessageBuilder().
							NullMessage(meterCmdTopic, numericmeter.CmdMeterReset, numericmeter.MeterElec).
							Build(),
					},
					cliffSuite.SleepNode(50 * time.Millisecond),
				},
			},
			{
				Name:     "service not found",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup:    routeMeter(mockednumericmeter.NewReporter(t)),
				Nodes: []*cliffSuite.Node{
					{
						Name: "unknown address returns error",
						Command: cliffSuite.NewMessageBuilder().
							NullMessage(
								"pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:404",
								numericmeter.CmdMeterGetReport,
								numericmeter.MeterElec,
							).
							Build(),
						Expectations: []*cliffSuite.Expectation{
							cliffSuite.ExpectError(
								"pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:404",
								numericmeter.MeterElec,
							),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func routeMeter(reporter *mockednumericmeter.Reporter) cliffSuite.BaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []cliffSuite.Mock) {
		t.Helper()

		return setupMeterService(t, mqtt, reporter)
	}
}

func routeResettableMeter(reporter *mockednumericmeter.Reporter, resettable *mockednumericmeter.ResettableReporter) cliffSuite.BaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []cliffSuite.Mock) {
		t.Helper()

		return setupMeterService(t, mqtt, &resettableMeter{Reporter: reporter, ResettableReporter: resettable})
	}
}

func setupMeterService(t *testing.T, mqtt *fimpgo.MqttTransport, reporter numericmeter.Reporter) ([]*router.Routing, []*task.Task, []cliffSuite.Mock) {
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

	return numericmeter.RouteService(ad), nil, nil
}
