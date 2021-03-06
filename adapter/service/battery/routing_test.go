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
	report := battery.FullReport{
		Level:  90,
		Health: 70,
		State:  "charging",
		Temp:   40,
	}

	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name: "successful get reports",
				Setup: routeBattery(
					mockedbattery.NewReporter(t).
						MockBatteryLevelReport(80, "charging", nil, true).
						MockBatteryHealthReport(70, nil, true).
						MockBatterySensorReport(20.1, "c", nil, true).
						MockBatteryFullReport(report, nil, true),
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
							suite.ExpectObject("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:battery/ad:2", "evt.battery.report", "battery", report),
						},
					},
				},
			},
			{
				Name: "failed get reports",
				Setup: routeBattery(
					mockedbattery.NewReporter(t).
						MockBatteryLevelReport(80, "charging", errors.New("fail level report"), true).
						MockBatteryHealthReport(70, errors.New("fail health report"), true).
						MockBatterySensorReport(20.1, "c", errors.New("fail sensor report"), true).
						MockBatteryFullReport(report, errors.New("fail full report"), true),
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
	report1 := battery.FullReport{
		Level:  90,
		Health: 60,
		State:  "charging",
		Temp:   40.5,
	}

	report2 := battery.FullReport{
		Level:  80,
		Health: 50,
		State:  "charging",
		Temp:   30.5,
	}

	alarm1 := battery.AlarmReport{
		Event:  battery.AlarmLowBatteryEvent,
		Status: battery.AlarmStatusActivate,
	}

	alarm2 := battery.AlarmReport{
		Event:  battery.AlarmLowBatteryEvent,
		Status: battery.AlarmStatusDeactivate,
	}

	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name: "Battery thing tasks",
				Setup: taskBattery(
					mockedbattery.NewReporter(t).
						MockBatteryLevelReport(80, "charging", nil, true).
						MockBatteryLevelReport(70, "charging", nil, true).
						MockBatteryLevelReport(0, "charging", errors.New("error"), false).
						MockBatteryAlarmReport(alarm1, nil, true).
						MockBatteryAlarmReport(alarm2, nil, true).
						MockBatteryAlarmReport(battery.AlarmReport{}, errors.New("error"), false).
						MockBatteryHealthReport(70, nil, true).
						MockBatteryHealthReport(60, nil, true).
						MockBatteryHealthReport(0, errors.New("error"), false).
						MockBatterySensorReport(20.1, "c", nil, true).
						MockBatterySensorReport(20.2, "c", nil, true).
						MockBatterySensorReport(0, "c", errors.New("error"), false).
						MockBatteryFullReport(report1, nil, true).
						MockBatteryFullReport(report2, nil, true).
						MockBatteryFullReport(battery.FullReport{}, errors.New("error"), false),
					10*time.Millisecond,
				),
				Nodes: []*suite.Node{
					{
						Name: "level report task",
						Expectations: []*suite.Expectation{
							suite.ExpectInt("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:battery/ad:2", "evt.lvl.report", "battery", 80),
							suite.ExpectInt("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:battery/ad:2", "evt.lvl.report", "battery", 70),
							suite.ExpectStringMap("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:battery/ad:2", "evt.alarm.report", "battery", alarm1.ToStrMap()),
							suite.ExpectStringMap("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:battery/ad:2", "evt.alarm.report", "battery", alarm2.ToStrMap()),
							suite.ExpectInt("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:battery/ad:2", "evt.health.report", "battery", 70),
							suite.ExpectInt("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:battery/ad:2", "evt.health.report", "battery", 60),
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:battery/ad:2", "evt.sensor.report", "battery", 20.1),
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:battery/ad:2", "evt.sensor.report", "battery", 20.2),
							suite.ExpectObject("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:battery/ad:2", "evt.battery.report", "battery", report1),
							suite.ExpectObject("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:battery/ad:2", "evt.battery.report", "battery", report2),
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
