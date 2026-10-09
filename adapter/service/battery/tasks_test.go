package battery_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/futurehomeno/fimpgo"
	log "github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/adapter"
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

func TestTaskReportingSkipsNotReported(t *testing.T) { //nolint:paralleltest
	old := log.StandardLogger().ReplaceHooks(make(log.LevelHooks))
	defer log.StandardLogger().ReplaceHooks(old)

	hook := logtest.NewGlobal()

	notReported := fmt.Errorf("no battery: %w", adapter.ErrNotReported)

	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "battery not reported yet",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: taskBattery(
					mockedbattery.NewReporter(t).
						MockBatteryLevelReport(0, errors.New("other"), true).
						MockBatteryLevelReport(0, notReported, false).
						MockBatteryAlarmReport(nil, battery.AlarmEventLowBattery, errors.New("other"), true).
						MockBatteryAlarmReport(nil, battery.AlarmEventLowBattery, notReported, false),
					50*time.Millisecond,
				),
				Nodes: []*suite.Node{
					{
						Name:    "no report is sent",
						Timeout: 300 * time.Millisecond,
						Expectations: []*suite.Expectation{
							suite.ExpectMessage("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:battery/ad:2", "evt.lvl.report", "battery").Never(),
							suite.ExpectMessage("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:battery/ad:2", "evt.alarm.report", "battery").Never(),
						},
					},
				},
			},
		},
	}

	s.Run(t)

	logged := 0

	for _, e := range hook.AllEntries() {
		if strings.Contains(e.Message, "[battery]") {
			logged++
		}
	}

	assert.Equal(t, 2, logged, "only the other errors are logged, never the battery not reported yet")
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
