package thing_test

import (
	"errors"
	"testing"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/battery"
	"github.com/futurehomeno/cliffhanger/adapter/thing"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
	adapterhelper "github.com/futurehomeno/cliffhanger/test/helper/adapter"
	mockedadapter "github.com/futurehomeno/cliffhanger/test/mocks/adapter"
	mockedbattery "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/battery"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

func TestRouteBattery(t *testing.T) { //nolint:paralleltest
	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "successful get reports",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeBattery(
					mockedbattery.NewReporter(t).
						MockBatteryLevelReport(80, nil, true),
				),
				Nodes: []*suite.Node{
					{
						Name: "successful get level report",
						Command: suite.NewMessageBuilder().
							NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:battery/ad:2", "cmd.lvl.get_report", "battery").
							Build(),
						Expectations: []*suite.Expectation{
							suite.ExpectInt("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:battery/ad:2", "evt.lvl.report", "battery", 80),
						},
					},
				},
			},

			{
				Name:     "failed get reports",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeBattery(
					mockedbattery.NewReporter(t).
						MockBatteryLevelReport(80, errors.New("fail level report"), true),
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
						Name: "wrong service under provided address level report",
						Command: suite.NewMessageBuilder().
							NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:batteraay/ad:2", "cmd.lvl.get_report", "battery").
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

func TestTaskBattery(t *testing.T) { //nolint:paralleltest
	alarm1 := &battery.AlarmReport{
		Event:  battery.AlarmEventLowBattery,
		Status: battery.AlarmStatusActivate,
	}

	alarm2 := &battery.AlarmReport{
		Event:  battery.AlarmEventLowBattery,
		Status: battery.AlarmStatusDeactivate,
	}

	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "Battery thing tasks",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: taskBattery(
					mockedbattery.NewReporter(t).
						MockBatteryLevelReport(80, nil, true).
						MockBatteryLevelReport(70, nil, true).
						MockBatteryLevelReport(0, errors.New("error"), false).
						MockBatteryAlarmReport(alarm1, battery.AlarmEventLowBattery, nil, true).
						MockBatteryAlarmReport(alarm2, battery.AlarmEventLowBattery, nil, true).
						MockBatteryAlarmReport(nil, battery.AlarmEventLowBattery, errors.New("error"), false),
					100*time.Millisecond,
				),
				Nodes: []*suite.Node{
					{
						Name: "level report task",
						Expectations: []*suite.Expectation{
							suite.ExpectInt("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:battery/ad:2", "evt.lvl.report", "battery", 80),
							suite.ExpectInt("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:battery/ad:2", "evt.lvl.report", "battery", 70),
							suite.ExpectStringMap("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:battery/ad:2", "evt.alarm.report", "battery", alarm1.ToStrMap()),
							suite.ExpectStringMap("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:battery/ad:2", "evt.alarm.report", "battery", alarm2.ToStrMap()),
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

	cfg := &thing.BatteryConfig{
		ThingConfig: &adapter.ThingConfig{
			InclusionReport: &fimptype.ThingInclusionReport{
				Address: "2",
			},
			Connector: mockedadapter.NewDefaultConnector(t),
		},
		BatteryConfig: &battery.Config{
			Specification: battery.Specification(
				"test_adapter",
				"1",
				"2",
				nil,
				[]string{battery.AlarmEventLowBattery},
			),
			Reporter: batteryReporter,
		},
	}

	seed := &adapter.ThingSeed{ID: "B", CustomAddress: "2"}

	factory := adapterhelper.FactoryHelper(func(adapter adapter.Adapter, publisher adapter.Publisher, thingState adapter.ThingState) (adapter.Thing, error) {
		return thing.NewBattery(publisher, thingState, cfg), nil
	})

	ad := adapterhelper.PrepareSeededAdapter(t, "../../testdata/adapter/test_adapter", mqtt, factory, adapter.ThingSeeds{seed})

	return thing.RouteBattery(ad), thing.TaskBattery(ad, interval), mocks
}
