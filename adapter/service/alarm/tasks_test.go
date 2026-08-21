package alarm_test

import (
	"testing"
	"time"

	"github.com/futurehomeno/fimpgo"

	"github.com/futurehomeno/cliffhanger/adapter/service/alarm"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
	adapterhelper "github.com/futurehomeno/cliffhanger/test/helper/adapter"
	mockedalarm "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/alarm"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

func TestTaskReporting(t *testing.T) { //nolint:paralleltest
	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "one change and one error per event during report cycles",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: taskReporting(
					mockedalarm.NewReporter(t).
						MockAlarmReport(&alarm.Report{Status: alarm.StatusActivate}, alarm.EventGroundingFault, nil, true).
						MockAlarmReport(nil, alarm.EventGroundingFault, errTest, true).
						MockAlarmReport(&alarm.Report{Status: alarm.StatusActivate}, alarm.EventGroundingFault, nil, true). // repeat, should not be sent again
						MockAlarmReport(&alarm.Report{Status: alarm.StatusDeactivate}, alarm.EventGroundingFault, nil, true).
						MockAlarmReport(&alarm.Report{Status: alarm.StatusDeactivate}, alarm.EventOverTemp, nil, true).
						MockAlarmReport(nil, alarm.EventOverTemp, errTest, true).
						MockAlarmReport(&alarm.Report{Status: alarm.StatusDeactivate}, alarm.EventOverTemp, nil, true). // repeat, should not be sent again
						MockAlarmReport(&alarm.Report{Status: alarm.StatusActivate}, alarm.EventOverTemp, nil, true),
					[]string{alarm.EventGroundingFault, alarm.EventOverTemp},
					100*time.Millisecond,
				),
				Nodes: []*suite.Node{
					{
						Name: "baseline, error and change are reported, repeats are deduplicated",
						Expectations: []*suite.Expectation{
							suite.ExpectStringMap(evtTopic, alarm.EvtAlarmReport, alarm.AlarmSystem,
								map[string]string{"event": alarm.EventGroundingFault, "status": alarm.StatusActivate}).ExactlyOnce(),
							suite.ExpectStringMap(evtTopic, alarm.EvtAlarmReport, alarm.AlarmSystem,
								map[string]string{"event": alarm.EventGroundingFault, "status": alarm.StatusDeactivate}).ExactlyOnce(),
							suite.ExpectStringMap(evtTopic, alarm.EvtAlarmReport, alarm.AlarmSystem,
								map[string]string{"event": alarm.EventOverTemp, "status": alarm.StatusDeactivate}).ExactlyOnce(),
							suite.ExpectStringMap(evtTopic, alarm.EvtAlarmReport, alarm.AlarmSystem,
								map[string]string{"event": alarm.EventOverTemp, "status": alarm.StatusActivate}).ExactlyOnce(),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func taskReporting(reporter *mockedalarm.Reporter, events []string, interval time.Duration) suite.BaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
		t.Helper()

		_, tasks, mocks := setupService(t, mqtt, reporter, events, interval)

		return nil, tasks, mocks
	}
}
