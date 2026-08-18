package thing_test

import (
	"testing"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/alarm"
	"github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"
	"github.com/futurehomeno/cliffhanger/adapter/thing"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
	adapterhelper "github.com/futurehomeno/cliffhanger/test/helper/adapter"
	mockedadapter "github.com/futurehomeno/cliffhanger/test/mocks/adapter"
	mockedalarm "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/alarm"
	mockedchargepoint "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/chargepoint"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

const alarmTopic = "pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:alarm_system/ad:2"

func TestRouteCarChargerAlarm(t *testing.T) { //nolint:paralleltest
	activated := &alarm.Report{Event: alarm.EventGroundingFault, Status: alarm.StatusActivate}
	deactivated := &alarm.Report{Event: alarm.EventGroundingFault, Status: alarm.StatusDeactivate}

	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "alarm report on demand",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeCarChargerAlarm(
					mockedalarm.NewReporter(t).
						MockAlarmReport(activated, alarm.EventGroundingFault, nil, true).
						MockAlarmReport(nil, alarm.EventOverTemp, nil, true),
				),
				Nodes: []*suite.Node{
					{
						Name:    "get report",
						Command: suite.StringMapMessage(alarmTopic, alarm.CmdAlarmGetReport, alarm.AlarmSystem, map[string]string{}),
						Expectations: []*suite.Expectation{
							suite.ExpectStringMap(
								"pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:alarm_system/ad:2",
								alarm.EvtAlarmReport,
								alarm.AlarmSystem,
								activated.ToStrMap(),
							),
						},
					},
				},
			},
			{
				Name:     "alarm cleared",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeCarChargerAlarm(
					mockedalarm.NewReporter(t).
						MockAlarmReport(deactivated, alarm.EventGroundingFault, nil, true).
						MockAlarmReport(nil, alarm.EventOverTemp, nil, true),
				),
				Nodes: []*suite.Node{
					{
						Name:    "get report",
						Command: suite.StringMapMessage(alarmTopic, alarm.CmdAlarmGetReport, alarm.AlarmSystem, map[string]string{}),
						Expectations: []*suite.Expectation{
							suite.ExpectStringMap(
								"pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:alarm_system/ad:2",
								alarm.EvtAlarmReport,
								alarm.AlarmSystem,
								deactivated.ToStrMap(),
							),
						},
					},
				},
			},
			{
				Name:     "no alarm service without configuration",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup:    routeCarChargerAlarm(nil),
				Nodes: []*suite.Node{
					{
						Name:    "get report",
						Command: suite.StringMapMessage(alarmTopic, alarm.CmdAlarmGetReport, alarm.AlarmSystem, map[string]string{}),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:alarm_system/ad:2", alarm.AlarmSystem),
						},
					},
				},
			},
			{
				Name:     "reporter supplied event is normalized to the advertised one",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeCarChargerAlarm(
					mockedalarm.NewReporter(t).
						MockAlarmReport(&alarm.Report{Event: "grounding_fault", Status: alarm.StatusActivate}, alarm.EventGroundingFault, nil, true).
						MockAlarmReport(nil, alarm.EventOverTemp, nil, true),
				),
				Nodes: []*suite.Node{
					{
						Name:    "get report",
						Command: suite.NullMessage(alarmTopic, alarm.CmdAlarmGetReport, alarm.AlarmSystem),
						Expectations: []*suite.Expectation{
							suite.ExpectStringMap(
								"pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:alarm_system/ad:2",
								alarm.EvtAlarmReport,
								alarm.AlarmSystem,
								activated.ToStrMap(),
							),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func TestTaskCarChargerAlarm(t *testing.T) { //nolint:paralleltest
	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "distinct events are not deduplicated when the reporter omits the event",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: taskCarChargerAlarm(
					mockedalarm.NewReporter(t).
						MockAlarmReport(&alarm.Report{Status: alarm.StatusActivate}, alarm.EventGroundingFault, nil, false).
						MockAlarmReport(&alarm.Report{Status: alarm.StatusActivate}, alarm.EventOverTemp, nil, false),
					100*time.Millisecond,
				),
				Nodes: []*suite.Node{
					{
						Name: "both alarms are reported under their own cache key",
						Expectations: []*suite.Expectation{
							suite.ExpectStringMap(
								"pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:alarm_system/ad:2",
								alarm.EvtAlarmReport,
								alarm.AlarmSystem,
								map[string]string{"event": alarm.EventGroundingFault, "status": alarm.StatusActivate},
							).ExactlyOnce(),
							suite.ExpectStringMap(
								"pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:alarm_system/ad:2",
								alarm.EvtAlarmReport,
								alarm.AlarmSystem,
								map[string]string{"event": alarm.EventOverTemp, "status": alarm.StatusActivate},
							).ExactlyOnce(),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func routeCarChargerAlarm(reporter *mockedalarm.Reporter) suite.BaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
		t.Helper()

		routing, _, mocks := setupCarChargerAlarm(t, mqtt, reporter, 0)

		return routing, nil, mocks
	}
}

func taskCarChargerAlarm(reporter *mockedalarm.Reporter, interval time.Duration) suite.BaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
		t.Helper()

		_, tasks, mocks := setupCarChargerAlarm(t, mqtt, reporter, interval)

		return nil, tasks, mocks
	}
}

func setupCarChargerAlarm(
	t *testing.T,
	mqtt *fimpgo.MqttTransport,
	reporter *mockedalarm.Reporter,
	interval time.Duration,
) ([]*router.Routing, []*task.Task, []suite.Mock) {
	t.Helper()

	controller := mockedchargepoint.NewController(t)
	mocks := []suite.Mock{controller}

	cfg := &thing.CarChargerConfig{
		ThingConfig: &adapter.ThingConfig{
			InclusionReport: &fimptype.ThingInclusionReport{Address: "2"},
			Connector:       mockedadapter.NewDefaultConnector(t),
		},
		ChargepointConfig: &chargepoint.Config{
			Specification: chargepoint.Specification(
				"test_adapter",
				"1",
				"2",
				nil,
				[]chargepoint.State{"ready_to_charge", "charging", "error"},
			),
			Controller: mockedchargepoint.NewMockedChargepoint(controller, nil, nil, nil, nil),
		},
	}

	if reporter != nil {
		cfg.AlarmConfig = &alarm.Config{
			Specification: alarm.Specification(
				"test_adapter",
				"1",
				"2",
				nil,
				[]string{alarm.EventGroundingFault, alarm.EventOverTemp},
			),
			Reporter: reporter,
		}

		mocks = append(mocks, reporter)
	}

	seed := &adapter.ThingSeed{ID: "B", CustomAddress: "2"}

	factory := adapterhelper.FactoryHelper(func(_ adapter.Adapter, publisher adapter.Publisher, thingState adapter.ThingState) (adapter.Thing, error) {
		return thing.NewCarCharger(publisher, thingState, cfg), nil
	})

	ad := adapterhelper.PrepareSeededAdapter(t, "../../testdata/adapter/test_adapter", mqtt, factory, adapter.ThingSeeds{seed})

	return thing.RouteCarCharger(ad), []*task.Task{alarm.TaskReporting(ad, interval)}, mocks
}
