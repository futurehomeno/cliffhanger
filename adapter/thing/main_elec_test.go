package thing_test

import (
	"errors"
	"testing"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
	"github.com/stretchr/testify/mock"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/numericmeter"
	"github.com/futurehomeno/cliffhanger/adapter/thing"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
	adapterhelper "github.com/futurehomeno/cliffhanger/test/helper/adapter"
	mockedadapter "github.com/futurehomeno/cliffhanger/test/mocks/adapter"
	mockednumericmeter "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/numericmeter"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

func TestRouteMainElec(t *testing.T) { //nolint:paralleltest
	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "Successful main elec reporting",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeMainElec(
					mockednumericmeter.NewMockedMeter(
						mockednumericmeter.NewReporter(t).
							MockMeterReport("W", 1500, nil, false).
							MockMeterReport("kWh", 165.78, nil, false),
						mockednumericmeter.NewExtendedReporter(t),
					),
				),
				Nodes: []*suite.Node{
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
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "evt.meter.report", "meter_elec", 165.78).
								ExpectProperty("unit", "kWh"),
						},
					},
					{
						Name:    "All electricity units",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "cmd.meter.get_report", "meter_elec", ""),
						Expectations: []*suite.Expectation{
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "evt.meter.report", "meter_elec", 1500).
								ExpectProperty("unit", "W"),
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "evt.meter.report", "meter_elec", 165.78).
								ExpectProperty("unit", "kWh"),
						},
					},
					{
						Name:    "All electricity units with null",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "cmd.meter.get_report", "meter_elec"),
						Expectations: []*suite.Expectation{
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "evt.meter.report", "meter_elec", 1500).
								ExpectProperty("unit", "W"),
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "evt.meter.report", "meter_elec", 165.78).
								ExpectProperty("unit", "kWh"),
						},
					},
				},
			},
			{
				Name:     "Failed main elec reporting",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeMainElec(
					mockednumericmeter.NewReporter(t).
						MockMeterReport("W", 0, errors.New("test"), true),
				),
				Nodes: []*suite.Node{
					{
						Name:    "Reporter error on power report",
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
					{
						Name:    "Unsupported extended report",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "cmd.meter_ext.get_report", "meter_elec"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "meter_elec"),
						},
					},
				},
			},
			{
				Name:     "Successful extended main elec reporting",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeMainElec(
					mockednumericmeter.NewMockedMeter(
						mockednumericmeter.NewReporter(t),
						mockednumericmeter.NewExtendedReporter(t).
							MockMeterExtendedReport(mock.Anything, map[string]float64{"p_import": 1500, "e_import": 165.78}, nil, true),
					),
				),
				Nodes: []*suite.Node{
					{
						Name:    "Extended report",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "cmd.meter_ext.get_report", "meter_elec"),
						Expectations: []*suite.Expectation{
							suite.ExpectFloatMap("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "evt.meter_ext.report", "meter_elec", map[string]float64{"p_import": 1500, "e_import": 165.78}),
						},
					},
				},
			},
			{
				Name:     "Failed extended main elec reporting",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeMainElec(
					mockednumericmeter.NewMockedMeter(
						mockednumericmeter.NewReporter(t),
						mockednumericmeter.NewExtendedReporter(t).
							MockMeterExtendedReport(mock.Anything, nil, errors.New("test"), true),
					),
				),
				Nodes: []*suite.Node{
					{
						Name:    "Reporter error on extended report",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "cmd.meter_ext.get_report", "meter_elec"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "meter_elec"),
						},
					},
					{
						Name:    "Non existent thing on on extended report",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:3", "cmd.meter_ext.get_report", "meter_elec"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:3", "meter_elec"),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func TestTaskMainElec(t *testing.T) { //nolint:paralleltest
	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "Main elec tasks",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: taskMainElec(
					mockednumericmeter.NewReporter(t).
						MockMeterReport("W", 1500, nil, true).
						MockMeterReport("W", 1500, nil, true).
						MockMeterReport("W", 0, errors.New("test"), true).
						MockMeterReport("W", 750, nil, false).
						MockMeterReport("kWh", 167.89, nil, true).
						MockMeterReport("kWh", 167.89, nil, true).
						MockMeterReport("kWh", 0, errors.New("test"), true).
						MockMeterReport("kWh", 167.99, nil, false),
					100*time.Millisecond,
				),
				Nodes: []*suite.Node{
					{
						Name: "One change and one error during four report cycles",
						Expectations: []*suite.Expectation{
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "evt.meter.report", "meter_elec", 1500).ExpectProperty("unit", "W").ExactlyOnce(),
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "evt.meter.report", "meter_elec", 750).ExpectProperty("unit", "W").ExactlyOnce(),
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "evt.meter.report", "meter_elec", 167.89).ExpectProperty("unit", "kWh").ExactlyOnce(),
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "evt.meter.report", "meter_elec", 167.99).ExpectProperty("unit", "kWh").ExactlyOnce(),
						},
					},
				},
			},
			{
				Name:     "Extended main elec tasks",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: taskMainElec(
					mockednumericmeter.NewMockedMeter(
						mockednumericmeter.NewReporter(t),
						mockednumericmeter.NewExtendedReporter(t).
							MockMeterExtendedReport(mock.Anything, map[string]float64{"p_import": 1500, "e_import": 165.78}, nil, true).
							MockMeterExtendedReport(mock.Anything, map[string]float64{"p_import": 1500, "e_import": 165.78}, nil, true).
							MockMeterExtendedReport(mock.Anything, nil, errors.New("test"), true).
							MockMeterExtendedReport(mock.Anything, map[string]float64{"p_import": 750, "e_import": 165.99}, nil, false),
					),
					100*time.Millisecond,
				),
				Nodes: []*suite.Node{
					{
						Name: "One change and one error during four report cycles",
						Expectations: []*suite.Expectation{
							suite.ExpectFloatMap("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "evt.meter_ext.report", "meter_elec", map[string]float64{"p_import": 1500, "e_import": 165.78}),
							suite.ExpectFloatMap("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "evt.meter_ext.report", "meter_elec", map[string]float64{"p_import": 750, "e_import": 165.99}),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func routeMainElec(
	meter interface{},
) suite.BaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
		t.Helper()

		routing, _, mocks := setupMainElec(t, mqtt, meter, 0)

		return routing, nil, mocks
	}
}

func taskMainElec(
	meterElecReporter interface{},
	interval time.Duration,
) suite.BaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
		t.Helper()

		_, tasks, mocks := setupMainElec(t, mqtt, meterElecReporter, interval)

		return nil, tasks, mocks
	}
}

func setupMainElec(
	t *testing.T,
	mqtt *fimpgo.MqttTransport,
	mockedMeter interface{},
	duration time.Duration,
) ([]*router.Routing, []*task.Task, []suite.Mock) {
	t.Helper()

	reporterMock := mockedMeter.(suite.Mock)        //nolint:forcetypeassert
	reporter := mockedMeter.(numericmeter.Reporter) //nolint:forcetypeassert

	mocks := []suite.Mock{reporterMock}

	cfg := &thing.MainElecConfig{
		ThingConfig: &adapter.ThingConfig{
			InclusionReport: &fimptype.ThingInclusionReport{
				Address: "2",
			},
			Connector: mockedadapter.NewConnector(t),
		},
		MeterElecConfig: &numericmeter.Config{
			Specification: numericmeter.Specification(
				numericmeter.MeterElec,
				"test_adapter",
				"1",
				"2",
				nil,
				[]numericmeter.Unit{"W", "kWh"},
				numericmeter.WithExtendedValues("p_import", "e_import"),
			),
			Reporter: reporter,
		},
	}

	seed := &adapter.ThingSeed{ID: "B", CustomAddress: "2"}

	factory := adapterhelper.FactoryHelper(func(adapter adapter.Adapter, publisher adapter.Publisher, thingState adapter.ThingState) (adapter.Thing, error) {
		return thing.NewMainElec(publisher, thingState, cfg), nil
	})

	ad := adapterhelper.PrepareSeededAdapter(t, "../../testdata/adapter/test_adapter", mqtt, factory, adapter.ThingSeeds{seed})

	return thing.RouteMainElec(ad), thing.TaskMainElec(ad, duration), mocks
}
