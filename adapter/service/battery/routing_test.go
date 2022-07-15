package battery_test

import (
	"errors"
	"testing"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/battery"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
	mockedbattery "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/battery"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

func TestRouteBattery(t *testing.T) { // nolint:paralleltest
	fullValue := map[string]interface{}{
		"lvl":         90,
		"health":      70,
		"state":       "charging",
		"temp_sensor": 40,
	}

	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name: "successful get reports",
				Setup: routeBattery(
					mockedbattery.NewReporter(t).
						MockBatteryLevelReport(80, nil, true).
						MockBatteryHealthReport(70, nil, true).
						MockBatterySensorReport(20.1, nil, true).
						MockBatteryFullReport(fullValue, nil, true),
				),
				Nodes: []*suite.Node{
					{
						Name: "get level report",
						Command: suite.NewMessageBuilder().
							NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:battery/ad:2", "cmd.lvl.get_report", "battery").
							Build(),
						Expectations: []*suite.Expectation{
							suite.ExpectInt("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:battery/ad:2", "evt.lvl.report", "battery", 80),
						},
					},
					{
						Name: "get health report",
						Command: suite.NewMessageBuilder().
							NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:battery/ad:2", "cmd.health.get_report", "battery").
							Build(),
						Expectations: []*suite.Expectation{
							suite.ExpectInt("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:battery/ad:2", "evt.health.report", "battery", 70),
						},
					},
					{
						Name: "get sensor report",
						Command: suite.NewMessageBuilder().
							NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:battery/ad:2", "cmd.sensor.get_report", "battery").
							Build(),
						Expectations: []*suite.Expectation{
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:battery/ad:2", "evt.sensor.report", "battery", 20.1),
						},
					},
					{
						Name: "get battery full report",
						Command: suite.NewMessageBuilder().
							NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:battery/ad:2", "cmd.battery.get_report", "battery").
							Build(),
						Expectations: []*suite.Expectation{
							suite.ExpectObject("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:battery/ad:2", "evt.battery.report", "battery", fullValue),
						},
					},
				},
			},
			{
				Name: "failed get reports",
				Setup: routeBattery(
					mockedbattery.NewReporter(t).
						MockBatteryLevelReport(80, errors.New("fail level report"), true).
						MockBatteryHealthReport(70, errors.New("fail health report"), true).
						MockBatterySensorReport(20.1, errors.New("fail sensor report"), true).
						MockBatteryFullReport(fullValue, errors.New("fail full report"), true),
				),
				Nodes: []*suite.Node{
					{
						Name: "get level report",
						Command: suite.NewMessageBuilder().
							NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:battery/ad:2", "cmd.lvl.get_report", "battery").
							Build(),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:battery/ad:2", "battery"),
						},
					},
					{
						Name: "get health report",
						Command: suite.NewMessageBuilder().
							NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:battery/ad:2", "cmd.health.get_report", "battery").
							Build(),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:battery/ad:2", "battery"),
						},
					},
					{
						Name: "get sensor report",
						Command: suite.NewMessageBuilder().
							NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:battery/ad:2", "cmd.sensor.get_report", "battery").
							Build(),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:battery/ad:2", "battery"),
						},
					},
					{
						Name: "get battery full report",
						Command: suite.NewMessageBuilder().
							NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:battery/ad:2", "cmd.battery.get_report", "battery").
							Build(),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:battery/ad:2", "battery"),
						},
					},
					{
						Name: "wrong address get level report",
						Command: suite.NewMessageBuilder().
							NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:battery/ad:3", "cmd.lvl.get_report", "battery").
							Build(),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:battery/ad:3", "battery"),
						},
					},
					{
						Name: "wrong address get level report",
						Command: suite.NewMessageBuilder().
							NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:battery/ad:3", "cmd.lvl.get_report", "battery").
							Build(),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:battery/ad:3", "battery"),
						},
					},
					{
						Name: "wrong address get health report",
						Command: suite.NewMessageBuilder().
							NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:battery/ad:3", "cmd.health.get_report", "battery").
							Build(),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:battery/ad:3", "battery"),
						},
					},
					{
						Name: "wrong address get sensor report",
						Command: suite.NewMessageBuilder().
							NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:battery/ad:3", "cmd.sensor.get_report", "battery").
							Build(),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:battery/ad:3", "battery"),
						},
					},
					{
						Name: "wrong address get battery full report",
						Command: suite.NewMessageBuilder().
							NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:battery/ad:3", "cmd.battery.get_report", "battery").
							Build(),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:battery/ad:3", "battery"),
						},
					},
					{
						Name: "wrong service under provided address level report",
						Command: suite.NewMessageBuilder().
							NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:batteraay/ad:2", "cmd.lvl.get_report", "battery").
							Build(),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:batteraay/ad:2", "battery"),
						},
					},
					{
						Name: "wrong service under provided address health report",
						Command: suite.NewMessageBuilder().
							NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:batteraay/ad:2", "cmd.health.get_report", "battery").
							Build(),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:batteraay/ad:2", "battery"),
						},
					},
					{
						Name: "wrong service under provided address sensor report",
						Command: suite.NewMessageBuilder().
							NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:batteraay/ad:2", "cmd.sensor.get_report", "battery").
							Build(),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:batteraay/ad:2", "battery"),
						},
					},
					{
						Name: "wrong service under provided address battery full report",
						Command: suite.NewMessageBuilder().
							NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:batteraay/ad:2", "cmd.battery.get_report", "battery").
							Build(),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:batteraay/ad:2", "battery"),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func TestTaskBattery(t *testing.T) { // nolint:paralleltest
	fullValue1 := map[string]interface{}{
		"lvl":         90,
		"health":      60,
		"state":       "charging",
		"temp_sensor": 40.5,
	}

	fullValue2 := map[string]interface{}{
		"lvl":         80,
		"health":      50,
		"state":       "charging",
		"temp_sensor": 30.5,
	}

	alarmValue1 := map[string]string{
		"event":  "low_battery",
		"status": "activ",
	}

	alarmValue2 := map[string]string{
		"event":  "low_battery",
		"status": "deactiv",
	}

	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name: "Battery thing tasks",
				Setup: taskBattery(
					mockedbattery.NewReporter(t).
						MockBatteryLevelReport(80, nil, true).
						MockBatteryLevelReport(70, nil, true).
						MockBatteryLevelReport(0, errors.New("error"), false).
						MockBatteryAlarmReport(alarmValue1, nil, true).
						MockBatteryAlarmReport(alarmValue2, nil, true).
						MockBatteryAlarmReport(nil, errors.New("error"), false).
						MockBatteryHealthReport(70, nil, true).
						MockBatteryHealthReport(60, nil, true).
						MockBatteryHealthReport(0, errors.New("error"), false).
						MockBatterySensorReport(20.1, nil, true).
						MockBatterySensorReport(20.2, nil, true).
						MockBatterySensorReport(0, errors.New("error"), false).
						MockBatteryFullReport(fullValue1, nil, true).
						MockBatteryFullReport(fullValue2, nil, true).
						MockBatteryFullReport(nil, errors.New("error"), false),
					10*time.Millisecond,
				),
				Nodes: []*suite.Node{
					{
						Name: "level report task",
						Expectations: []*suite.Expectation{
							suite.ExpectInt("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:battery/ad:2", "evt.lvl.report", "battery", 80),
							suite.ExpectInt("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:battery/ad:2", "evt.lvl.report", "battery", 70),
							suite.ExpectStringMap("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:battery/ad:2", "evt.alarm.report", "battery", alarmValue1),
							suite.ExpectStringMap("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:battery/ad:2", "evt.alarm.report", "battery", alarmValue2),
							suite.ExpectInt("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:battery/ad:2", "evt.health.report", "battery", 70),
							suite.ExpectInt("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:battery/ad:2", "evt.health.report", "battery", 60),
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:battery/ad:2", "evt.sensor.report", "battery", 20.1),
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:battery/ad:2", "evt.sensor.report", "battery", 20.2),
							suite.ExpectObject("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:battery/ad:2", "evt.battery.report", "battery", fullValue1),
							suite.ExpectObject("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:battery/ad:2", "evt.battery.report", "battery", fullValue2),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func routeBattery(
	batteryReporter *mockedbattery.Reporter,
) suite.BaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
		t.Helper()

		routing, _, mocks := setupBattery(t, mqtt, batteryReporter, 0)

		return routing, nil, mocks
	}
}

func taskBattery(
	batteryReporter *mockedbattery.Reporter,
	interval time.Duration,
) suite.BaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
		t.Helper()

		routing, tasks, mocks := setupBattery(t, mqtt, batteryReporter, interval)

		return routing, tasks, mocks
	}
}

func setupBattery(
	t *testing.T,
	mqtt *fimpgo.MqttTransport,
	batteryReporter *mockedbattery.Reporter,
	interval time.Duration,
) ([]*router.Routing, []*task.Task, []suite.Mock) {
	t.Helper()

	mocks := []suite.Mock{batteryReporter}

	cfg := &BatteryThingConfig{
		InclusionReport: &fimptype.ThingInclusionReport{
			Address: "2",
		},
		BatteryConfig: &battery.Config{
			Specification: battery.Specification(
				"test_adapter",
				"1",
				"2",
				nil,
			),
			Reporter: batteryReporter,
		},
	}

	battery := newBatteryThing(mqtt, cfg)
	ad := adapter.NewAdapter(nil, "test_adapter", "1")
	ad.RegisterThing(battery)

	return routeBatteryThing(ad), taskBatteryThing(ad, interval), mocks
}

// BatteryThingConfig represents a config for testing battery service.
type BatteryThingConfig struct {
	InclusionReport *fimptype.ThingInclusionReport
	BatteryConfig   *battery.Config
}

// newBatteryThing creates a thinng that can be used for testing battery service.
func newBatteryThing(
	mqtt *fimpgo.MqttTransport,
	cfg *BatteryThingConfig,
) adapter.Thing {
	services := []adapter.Service{
		battery.NewService(mqtt, cfg.BatteryConfig),
	}

	return adapter.NewThing(cfg.InclusionReport, services...)
}

// routeBatteryThing creates a thing that can be used for testing battery service.
func routeBatteryThing(adapter adapter.Adapter) []*router.Routing {
	return router.Combine(
		battery.RouteService(adapter),
	)
}

// taskBatteryThing creates background tasks specific for battery service.
func taskBatteryThing(
	adapter adapter.Adapter,
	interval time.Duration,
	voter ...task.Voter,
) []*task.Task {
	return []*task.Task{
		battery.TaskReporting(adapter, interval, voter...),
	}
}
