package numericsensor_test

import (
	"errors"
	"testing"

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

const (
	sensorEvtTopic = "pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:sensor_temp/ad:2"
	sensorCmdTopic = "pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:sensor_temp/ad:2"
	sensorService  = "sensor_temp"
)

func TestRouteService(t *testing.T) { //nolint:paralleltest
	s := &cliffSuite.Suite{
		Cases: []*cliffSuite.Case{
			{
				Name:     "sensor get report all units",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeSensor(mockednumericsensor.NewReporter(t).
					MockNumericSensorReport(numericsensor.UnitC, 21.5, nil, true).
					MockNumericSensorReport(numericsensor.UnitF, 70.7, nil, true),
				),
				Nodes: []*cliffSuite.Node{
					{
						Name: "null payload reports all units",
						Command: cliffSuite.NewMessageBuilder().
							NullMessage(sensorCmdTopic, numericsensor.CmdSensorGetReport, sensorService).
							Build(),
						Expectations: []*cliffSuite.Expectation{
							cliffSuite.ExpectFloat(sensorEvtTopic, numericsensor.EvtSensorReport, sensorService, 21.5),
							cliffSuite.ExpectFloat(sensorEvtTopic, numericsensor.EvtSensorReport, sensorService, 70.7),
						},
					},
				},
			},
			{
				Name:     "sensor get report specific unit",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeSensor(mockednumericsensor.NewReporter(t).
					MockNumericSensorReport(numericsensor.UnitC, 21.5, nil, true),
				),
				Nodes: []*cliffSuite.Node{
					{
						Name: "string payload reports requested unit",
						Command: cliffSuite.NewMessageBuilder().
							StringMessage(sensorCmdTopic, numericsensor.CmdSensorGetReport, sensorService, numericsensor.UnitC).
							Build(),
						Expectations: []*cliffSuite.Expectation{
							cliffSuite.ExpectFloat(sensorEvtTopic, numericsensor.EvtSensorReport, sensorService, 21.5),
						},
					},
				},
			},
			{
				Name:     "sensor get report controller error",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeSensor(mockednumericsensor.NewReporter(t).
					MockNumericSensorReport(numericsensor.UnitC, 0, errors.New("controller error"), true),
				),
				Nodes: []*cliffSuite.Node{
					{
						Name: "controller error returns error event",
						Command: cliffSuite.NewMessageBuilder().
							StringMessage(sensorCmdTopic, numericsensor.CmdSensorGetReport, sensorService, numericsensor.UnitC).
							Build(),
						Expectations: []*cliffSuite.Expectation{
							cliffSuite.ExpectError(sensorEvtTopic, sensorService),
						},
					},
				},
			},
			{
				Name:     "service not found",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup:    routeSensor(mockednumericsensor.NewReporter(t)),
				Nodes: []*cliffSuite.Node{
					{
						Name: "unknown address returns error",
						Command: cliffSuite.NewMessageBuilder().
							NullMessage(
								"pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:sensor_temp/ad:404",
								numericsensor.CmdSensorGetReport,
								sensorService,
							).
							Build(),
						Expectations: []*cliffSuite.Expectation{
							cliffSuite.ExpectError(
								"pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:sensor_temp/ad:404",
								sensorService,
							),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func routeSensor(reporter *mockednumericsensor.Reporter) cliffSuite.BaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []cliffSuite.Mock) {
		t.Helper()

		return setupSensorService(t, mqtt, reporter)
	}
}

func setupSensorService(t *testing.T, mqtt *fimpgo.MqttTransport, reporter *mockednumericsensor.Reporter) ([]*router.Routing, []*task.Task, []cliffSuite.Mock) {
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

	return numericsensor.RouteService(ad), nil, nil
}
