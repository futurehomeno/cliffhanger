package thing_test

import (
	"errors"
	"testing"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/meterelec"
	"github.com/futurehomeno/cliffhanger/adapter/thing"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
	mockedmeterelec "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/meterelec"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

func TestRouteMainElec(t *testing.T) { //nolint:paralleltest
	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name: "Successful main elec reporting",
				Setup: routeMainElec(
					mockedmeterelec.MockReporter().
						MockElectricityMeterReport("W", 1500, nil, false).
						MockElectricityMeterReport("kWh", 165.78, nil, false),
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
				Name: "Failed main elec reporting",
				Setup: routeMainElec(
					mockedmeterelec.MockReporter().
						MockElectricityMeterReport("W", 0, errors.New("test"), true),
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
				Name: "Successful extended main elec reporting",
				Setup: routeMainElec(
					mockedmeterelec.MockExtendedReporter().
						MockElectricityMeterExtendedReport(map[string]float64{"p_import": 1500, "e_import": 165.78}, nil, true),
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
				Name: "Failed extended main elec reporting",
				Setup: routeMainElec(
					mockedmeterelec.MockExtendedReporter().
						MockElectricityMeterExtendedReport(nil, errors.New("test"), true),
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

func TestTaskMainElec(t *testing.T) {
	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name: "Main elec tasks",
				Setup: taskMainElec(
					mockedmeterelec.MockReporter().
						MockElectricityMeterReport("W", 1500, nil, true).
						MockElectricityMeterReport("W", 1500, nil, true).
						MockElectricityMeterReport("W", 0, errors.New("test"), true).
						MockElectricityMeterReport("W", 750, nil, false).
						MockElectricityMeterReport("kWh", 167.89, nil, true).
						MockElectricityMeterReport("kWh", 167.89, nil, true).
						MockElectricityMeterReport("kWh", 0, errors.New("test"), true).
						MockElectricityMeterReport("kWh", 167.99, nil, false),
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
				Name: "Extended main elec tasks",
				Setup: taskMainElec(
					mockedmeterelec.MockExtendedReporter().
						MockElectricityMeterExtendedReport(map[string]float64{"p_import": 1500, "e_import": 165.78}, nil, true).
						MockElectricityMeterExtendedReport(map[string]float64{"p_import": 1500, "e_import": 165.78}, nil, true).
						MockElectricityMeterExtendedReport(nil, errors.New("test"), true).
						MockElectricityMeterExtendedReport(map[string]float64{"p_import": 750, "e_import": 165.99}, nil, false),
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

type mockedMeterElec interface {
	*mockedmeterelec.Reporter | *mockedmeterelec.ExtendedReporter
	suite.Mock
	meterelec.Reporter
}

func routeMainElec[T mockedMeterElec](
	meterElecReporter T,
) suite.CaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
		t.Helper()

		routing, _, mocks := setupMainElec(t, mqtt, meterElecReporter, 0)

		return routing, nil, mocks
	}
}

func taskMainElec[T mockedMeterElec](
	meterElecReporter T,
	interval time.Duration,
) suite.CaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
		t.Helper()

		_, tasks, mocks := setupMainElec(t, mqtt, meterElecReporter, interval)

		return nil, tasks, mocks
	}
}

func setupMainElec[T mockedMeterElec](
	t *testing.T,
	mqtt *fimpgo.MqttTransport,
	meterElecReporter T,
	duration time.Duration,
) ([]*router.Routing, []*task.Task, []suite.Mock) {
	t.Helper()

	mocks := []suite.Mock{meterElecReporter}

	cfg := &thing.MainElecConfig{
		InclusionReport: &fimptype.ThingInclusionReport{
			Address: "2",
		},
		MeterElecConfig: &meterelec.Config{
			Specification: meterelec.Specification(
				"test_adapter",
				"1",
				"2",
				nil,
				[]string{"W", "kWh"},
				[]string{"p_import", "e_import"},
			),
			Reporter: meterElecReporter,
		},
	}

	b := thing.NewMainElec(
		mqtt,
		cfg,
	)

	ad := adapter.NewAdapter(nil, "test_adapter", "1")
	ad.RegisterThing(b)

	return thing.RouteMainElec(ad), thing.TaskMainElec(ad, duration), mocks
}
