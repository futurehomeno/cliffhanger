package alarm_test

import (
	"errors"
	"testing"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/alarm"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
	adapterhelper "github.com/futurehomeno/cliffhanger/test/helper/adapter"
	mockedadapter "github.com/futurehomeno/cliffhanger/test/mocks/adapter"
	mockedalarm "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/alarm"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

const (
	cmdTopic = "pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:alarm_system/ad:2"
	evtTopic = "pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:alarm_system/ad:2"

	nonExistentEvtTopic = "pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:alarm_system/ad:3"
)

var errTest = errors.New("test")

func TestRouteService(t *testing.T) { //nolint:paralleltest
	activated := &alarm.Report{Event: alarm.EventGroundingFault, Status: alarm.StatusActivate}
	deactivated := &alarm.Report{Event: alarm.EventOverTemp, Status: alarm.StatusDeactivate}

	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "successful get report for multiple events",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(
					mockedalarm.NewReporter(t).
						MockAlarmReport(activated, alarm.EventGroundingFault, nil, true).
						MockAlarmReport(deactivated, alarm.EventOverTemp, nil, true),
					[]string{alarm.EventGroundingFault, alarm.EventOverTemp},
				),
				Nodes: []*suite.Node{
					{
						Name:    "get report",
						Command: suite.NullMessage(cmdTopic, alarm.CmdAlarmGetReport, alarm.AlarmSystem),
						Expectations: []*suite.Expectation{
							suite.ExpectStringMap(evtTopic, alarm.EvtAlarmReport, alarm.AlarmSystem, activated.ToStrMap()),
							suite.ExpectStringMap(evtTopic, alarm.EvtAlarmReport, alarm.AlarmSystem, deactivated.ToStrMap()),
						},
					},
				},
			},
			{
				Name:     "reporter error on one of the events",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(
					mockedalarm.NewReporter(t).
						MockAlarmReport(activated, alarm.EventGroundingFault, nil, true).
						MockAlarmReport(nil, alarm.EventOverTemp, errTest, true),
					[]string{alarm.EventGroundingFault, alarm.EventOverTemp},
				),
				Nodes: []*suite.Node{
					{
						Name:    "get report",
						Command: suite.NullMessage(cmdTopic, alarm.CmdAlarmGetReport, alarm.AlarmSystem),
						Expectations: []*suite.Expectation{
							suite.ExpectStringMap(evtTopic, alarm.EvtAlarmReport, alarm.AlarmSystem, activated.ToStrMap()),
							suite.ExpectError(evtTopic, alarm.AlarmSystem),
						},
					},
				},
			},
			{
				Name:     "nil report is not sent and does not fail the command",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(
					mockedalarm.NewReporter(t).
						MockAlarmReport(nil, alarm.EventGroundingFault, nil, true).
						MockAlarmReport(deactivated, alarm.EventOverTemp, nil, true),
					[]string{alarm.EventGroundingFault, alarm.EventOverTemp},
				),
				Nodes: []*suite.Node{
					{
						Name:    "get report",
						Command: suite.NullMessage(cmdTopic, alarm.CmdAlarmGetReport, alarm.AlarmSystem),
						Timeout: 400 * time.Millisecond,
						Expectations: []*suite.Expectation{
							suite.ExpectStringMap(evtTopic, alarm.EvtAlarmReport, alarm.AlarmSystem, deactivated.ToStrMap()),
							suite.ExpectError(evtTopic, alarm.AlarmSystem).Never(),
							expectNoReportForEvent(alarm.EventGroundingFault),
						},
					},
				},
			},
			{
				Name:     "service not found",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup:    routeService(mockedalarm.NewReporter(t), []string{alarm.EventGroundingFault}),
				Nodes: []*suite.Node{
					{
						Name: "non-existent thing on get report",
						Command: suite.NullMessage(
							"pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:alarm_system/ad:3",
							alarm.CmdAlarmGetReport,
							alarm.AlarmSystem,
						),
						Expectations: []*suite.Expectation{
							suite.ExpectError(nonExistentEvtTopic, alarm.AlarmSystem),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

// expectNoReportForEvent asserts that no evt.alarm.report was ever published for the given event.
func expectNoReportForEvent(event string) *suite.Expectation {
	return suite.NewExpectation(
		router.ForTopic(evtTopic),
		router.ForType(alarm.EvtAlarmReport),
		router.ForService(alarm.AlarmSystem),
		router.MessageVoterFn(func(message *fimpgo.Message) bool {
			v, err := message.Payload.GetStrMapValue()
			if err != nil {
				return false
			}

			return v["event"] == event
		}),
	).Never()
}

func routeService(reporter *mockedalarm.Reporter, events []string) suite.BaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
		t.Helper()

		routing, _, mocks := setupService(t, mqtt, reporter, events, 0)

		return routing, nil, mocks
	}
}

func setupService(
	t *testing.T,
	mqtt *fimpgo.MqttTransport,
	reporter *mockedalarm.Reporter,
	events []string,
	duration time.Duration,
) ([]*router.Routing, []*task.Task, []suite.Mock) {
	t.Helper()

	mocks := []suite.Mock{reporter}

	thingCfg := &adapter.ThingConfig{
		InclusionReport: &fimptype.ThingInclusionReport{
			Address: "2",
		},
		Connector: mockedadapter.NewDefaultConnector(t),
	}

	alarmCfg := &alarm.Config{
		Specification: alarm.Specification(
			"test_adapter",
			"1",
			"2",
			nil,
			events,
		),
		Reporter: reporter,
	}

	seed := &adapter.ThingSeed{ID: "B", CustomAddress: "2"}

	factory := adapterhelper.FactoryHelper(func(a adapter.Adapter, publisher adapter.Publisher, thingState adapter.ThingState) (adapter.Thing, error) {
		return adapter.NewThing(publisher, thingState, thingCfg, alarm.NewService(publisher, alarmCfg)), nil
	})

	ad := adapterhelper.PrepareSeededAdapter(t, "../../testdata/adapter/test_adapter", mqtt, factory, adapter.ThingSeeds{seed})

	return alarm.RouteService(ad), task.Combine(alarm.TaskReporting(ad, duration)), mocks
}
