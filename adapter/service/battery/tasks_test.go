package battery_test

import (
	"testing"
	"time"

	"github.com/futurehomeno/fimpgo"

	"github.com/futurehomeno/cliffhanger/adapter/service/battery"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
	adapterhelper "github.com/futurehomeno/cliffhanger/test/helper/adapter"
	mockedbattery "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/battery"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

func TestTaskReporting(t *testing.T) { //nolint:paralleltest
	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "Battery tasks",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: taskBattery(
					mockedbattery.NewReporter(t).
						MockBatteryLevelReport(80, nil, true).
						MockBatteryLevelReport(0, errTest, true).
						MockBatteryLevelReport(80, nil, true).
						MockBatteryLevelReport(65, nil, true). // should be sent
						MockBatteryAlarmReport(&battery.AlarmReport{Event: battery.AlarmEventLowBattery, Status: battery.AlarmStatusDeactivate}, battery.AlarmEventLowBattery, nil, true).
						MockBatteryAlarmReport(nil, battery.AlarmEventLowBattery, errTest, true).
						MockBatteryAlarmReport(&battery.AlarmReport{Event: battery.AlarmEventLowBattery, Status: battery.AlarmStatusDeactivate}, battery.AlarmEventLowBattery, nil, true).
						MockBatteryAlarmReport(&battery.AlarmReport{Event: battery.AlarmEventLowBattery, Status: battery.AlarmStatusActivate}, battery.AlarmEventLowBattery, nil, true), // should be sent
					100*time.Millisecond,
				),
				Nodes: []*suite.Node{
					{
						Name: "One change and one error during four report cycles",
						Expectations: []*suite.Expectation{
							suite.ExpectInt("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:battery/ad:2", "evt.lvl.report", "battery", 80).ExactlyOnce(),
							suite.ExpectInt("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:battery/ad:2", "evt.lvl.report", "battery", 65).ExactlyOnce(),
							suite.ExpectStringMap("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:battery/ad:2", "evt.alarm.report", "battery",
								(&battery.AlarmReport{Event: battery.AlarmEventLowBattery, Status: battery.AlarmStatusDeactivate}).ToStrMap()).ExactlyOnce(),
							suite.ExpectStringMap("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:battery/ad:2", "evt.alarm.report", "battery",
								(&battery.AlarmReport{Event: battery.AlarmEventLowBattery, Status: battery.AlarmStatusActivate}).ToStrMap()).ExactlyOnce(),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func taskBattery(
	reporter battery.Reporter,
	interval time.Duration,
) suite.BaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
		t.Helper()

		_, tasks, mocks := setupService(t, mqtt, reporter, interval)

		return nil, tasks, mocks
	}
}
