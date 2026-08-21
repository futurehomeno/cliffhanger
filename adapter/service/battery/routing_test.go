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
	adapterhelper "github.com/futurehomeno/cliffhanger/test/helper/adapter"
	mockedadapter "github.com/futurehomeno/cliffhanger/test/mocks/adapter"
	mockedbattery "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/battery"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

var errTest = errors.New("test")

func TestRouteService(t *testing.T) { //nolint:paralleltest
	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "successful level report routing",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(
					mockedbattery.NewReporter(t).
						MockBatteryLevelReport(80, nil, true),
				),
				Nodes: []*suite.Node{
					{
						Name:    "level report",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:battery/ad:2", "cmd.lvl.get_report", "battery"),
						Expectations: []*suite.Expectation{
							suite.ExpectInt("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:battery/ad:2", "evt.lvl.report", "battery", 80),
						},
					},
				},
			},
			{
				Name:     "failed level report routing - reporter error",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(
					mockedbattery.NewReporter(t).
						MockBatteryLevelReport(0, errTest, true),
				),
				Nodes: []*suite.Node{
					{
						Name:    "level report",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:battery/ad:2", "cmd.lvl.get_report", "battery"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:battery/ad:2", "battery"),
						},
					},
				},
			},
			{
				Name:     "failed level report routing - service not found",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup:    routeService(mockedbattery.NewReporter(t)),
				Nodes: []*suite.Node{
					{
						Name:    "non-existent thing on level report",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:battery/ad:3", "cmd.lvl.get_report", "battery"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:battery/ad:3", "battery"),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func routeService(reporter battery.Reporter) suite.BaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
		t.Helper()

		routing, _, mocks := setupService(t, mqtt, reporter, 0)

		return routing, nil, mocks
	}
}

func setupService(
	t *testing.T,
	mqtt *fimpgo.MqttTransport,
	reporter battery.Reporter,
	duration time.Duration,
) ([]*router.Routing, []*task.Task, []suite.Mock) {
	t.Helper()

	mockedReporter, ok := reporter.(suite.Mock)
	if !ok {
		t.Fatal("reporter is not a mock")
	}

	mocks := []suite.Mock{mockedReporter}

	thingCfg := &adapter.ThingConfig{
		InclusionReport: &fimptype.ThingInclusionReport{
			Address: "2",
		},
		Connector: mockedadapter.NewDefaultConnector(t),
	}

	batteryCfg := &battery.Config{
		Specification: battery.Specification(
			"test_adapter",
			"1",
			"2",
			nil,
			[]string{battery.AlarmEventLowBattery},
		),
		Reporter: reporter,
	}

	seed := &adapter.ThingSeed{ID: "B", CustomAddress: "2"}

	factory := adapterhelper.FactoryHelper(func(a adapter.Adapter, publisher adapter.Publisher, thingState adapter.ThingState) (adapter.Thing, error) {
		return adapter.NewThing(publisher, thingState, thingCfg, battery.NewService(publisher, batteryCfg)), nil
	})

	ad := adapterhelper.PrepareSeededAdapter(t, "../../testdata/adapter/test_adapter", mqtt, factory, adapter.ThingSeeds{seed})

	return battery.RouteService(ad), task.Combine(battery.TaskReporting(ad, duration)), mocks
}
