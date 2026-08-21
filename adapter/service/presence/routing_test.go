package presence_test

import (
	"errors"
	"testing"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/presence"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
	adapterhelper "github.com/futurehomeno/cliffhanger/test/helper/adapter"
	mockedadapter "github.com/futurehomeno/cliffhanger/test/mocks/adapter"
	mockedpresence "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/presence"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

var errTest = errors.New("test")

func TestRouteService(t *testing.T) { //nolint:paralleltest
	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "successful presence report routing",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(
					mockedpresence.NewController(t).
						MockSensorPresenceReport(true, nil, true).
						MockSensorPresenceReport(false, nil, true),
				),
				Nodes: []*suite.Node{
					{
						Name:    "get presence report - true",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:sensor_presence/ad:2", "cmd.presence.get_report", "sensor_presence"),
						Expectations: []*suite.Expectation{
							suite.ExpectBool("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:sensor_presence/ad:2", "evt.presence.report", "sensor_presence", true),
						},
					},
					{
						Name:    "get presence report - false",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:sensor_presence/ad:2", "cmd.presence.get_report", "sensor_presence"),
						Expectations: []*suite.Expectation{
							suite.ExpectBool("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:sensor_presence/ad:2", "evt.presence.report", "sensor_presence", false),
						},
					},
				},
			},
			{
				Name:     "failed presence report routing - controller error",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(
					mockedpresence.NewController(t).
						MockSensorPresenceReport(false, errTest, true),
				),
				Nodes: []*suite.Node{
					{
						Name:    "get presence report",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:sensor_presence/ad:2", "cmd.presence.get_report", "sensor_presence"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:sensor_presence/ad:2", "sensor_presence"),
						},
					},
				},
			},
			{
				Name:     "failed presence report routing - service not found",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup:    routeService(mockedpresence.NewController(t)),
				Nodes: []*suite.Node{
					{
						Name:    "get presence report - non-existent thing",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:sensor_presence/ad:3", "cmd.presence.get_report", "sensor_presence"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:sensor_presence/ad:3", "sensor_presence"),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func routeService(controller presence.Controller) suite.BaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
		t.Helper()

		routing, _, mocks := setupService(t, mqtt, controller, 0)

		return routing, nil, mocks
	}
}

func setupService(
	t *testing.T,
	mqtt *fimpgo.MqttTransport,
	controller presence.Controller,
	duration time.Duration,
) ([]*router.Routing, []*task.Task, []suite.Mock) {
	t.Helper()

	mockedController, ok := controller.(suite.Mock)
	if !ok {
		t.Fatal("controller is not a mock")
	}

	mocks := []suite.Mock{mockedController}

	thingCfg := &adapter.ThingConfig{
		InclusionReport: &fimptype.ThingInclusionReport{
			Address: "2",
		},
		Connector: mockedadapter.NewDefaultConnector(t),
	}

	presenceCfg := &presence.Config{
		Specification: presence.Specification(
			"test_adapter",
			"1",
			"2",
			nil,
		),
		Controller: controller,
	}

	seed := &adapter.ThingSeed{ID: "B", CustomAddress: "2"}

	factory := adapterhelper.FactoryHelper(func(a adapter.Adapter, publisher adapter.Publisher, thingState adapter.ThingState) (adapter.Thing, error) {
		return adapter.NewThing(publisher, thingState, thingCfg, presence.NewService(publisher, presenceCfg)), nil
	})

	ad := adapterhelper.PrepareSeededAdapter(t, "../../testdata/adapter/test_adapter", mqtt, factory, adapter.ThingSeeds{seed})

	return presence.RouteService(ad), task.Combine(presence.TaskReporting(ad, duration)), mocks
}
