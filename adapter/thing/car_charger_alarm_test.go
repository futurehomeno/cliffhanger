package thing_test

import (
	"testing"

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
		},
	}

	s.Run(t)
}

func routeCarChargerAlarm(reporter *mockedalarm.Reporter) suite.BaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
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

		return thing.RouteCarCharger(ad), nil, mocks
	}
}
