package thing_test

import (
	"errors"
	"testing"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/presence"
	"github.com/futurehomeno/cliffhanger/adapter/thing"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
	adapterhelper "github.com/futurehomeno/cliffhanger/test/helper/adapter"
	mockedadapter "github.com/futurehomeno/cliffhanger/test/mocks/adapter"
	mockednumericsensor "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/numericsensor"
	mockedpresence "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/presence"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

func TestRouteSensor(t *testing.T) { //nolint:paralleltest
	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "successful presence report",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routePresence(
					mockedpresence.NewController(t).
						MockSensorPresenceReport(true, nil, true),
					nil,
				),
				Nodes: []*suite.Node{
					{
						Name: "get report",
						Command: suite.NewMessageBuilder().
							NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:sensor_presence/ad:2", "cmd.presence.get_report", "sensor_presence").
							Build(),
						Expectations: []*suite.Expectation{
							suite.ExpectBool("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:sensor_presence/ad:2", "evt.presence.report", "sensor_presence", true),
						},
					},
				},
			},
			{
				Name:     "failed get report",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routePresence(
					mockedpresence.NewController(t).
						MockSensorPresenceReport(false, errors.New("error"), true),
					nil,
				),
				Nodes: []*suite.Node{
					{
						Name: "get report error",
						Command: suite.NewMessageBuilder().
							NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:sensor_presence/ad:2", "cmd.presence.get_report", "sensor_presence").
							Build(),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:sensor_presence/ad:2", "sensor_presence"),
						},
					},
					{
						Name: "wrong address",
						Command: suite.NewMessageBuilder().
							NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:sensor_presence/ad:3", "cmd.presence.get_report", "sensor_presence").
							Build(),
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

func TestTaskSensor(t *testing.T) { //nolint:paralleltest
	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "Presence thing tasks",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: taskPresence(
					mockedpresence.NewController(t).
						MockSensorPresenceReport(true, nil, true).
						MockSensorPresenceReport(true, errors.New("task error"), true).
						MockSensorPresenceReport(false, nil, true).
						MockSensorPresenceReport(false, nil, false),
					nil,
					100*time.Millisecond,
				),
				Nodes: []*suite.Node{
					{
						Name: "Two reports and one skip",
						Expectations: []*suite.Expectation{
							suite.ExpectBool("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:sensor_presence/ad:2", "evt.presence.report", "sensor_presence", true),
							suite.ExpectBool("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:sensor_presence/ad:2", "evt.presence.report", "sensor_presence", false),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func routePresence(
	presenceController *mockedpresence.Controller,
	numericSensorReporters []*mockednumericsensor.Reporter,
) suite.BaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
		t.Helper()

		routing, _, mocks := setupPresence(t, mqtt, presenceController, numericSensorReporters, 0)

		return routing, nil, mocks
	}
}

func taskPresence(
	presenceController *mockedpresence.Controller,
	numericSensorReporters []*mockednumericsensor.Reporter,
	interval time.Duration,
) suite.BaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
		t.Helper()

		_, tasks, mocks := setupPresence(t, mqtt, presenceController, numericSensorReporters, interval)

		return nil, tasks, mocks
	}
}

func setupPresence(
	t *testing.T,
	mqtt *fimpgo.MqttTransport,
	presenceController *mockedpresence.Controller,
	_ []*mockednumericsensor.Reporter,
	duration time.Duration,
) ([]*router.Routing, []*task.Task, []suite.Mock) {
	t.Helper()

	mocks := []suite.Mock{presenceController}

	cfg := &thing.SensorConfig{
		ThingConfig: &adapter.ThingConfig{
			InclusionReport: &fimptype.ThingInclusionReport{
				Address: "2",
			},
			Connector: mockedadapter.NewConnector(t),
		},
		PresenceConfig: &presence.Config{
			Specification: presence.Specification(
				"test_adapter",
				"1",
				"2",
				nil,
			),
			Controller: presenceController,
		},
	}

	seed := &adapter.ThingSeed{ID: "B", CustomAddress: "2"}

	factory := adapterhelper.FactoryHelper(func(adapter adapter.Adapter, publisher adapter.Publisher, thingState adapter.ThingState) (adapter.Thing, error) {
		return thing.NewSensor(publisher, thingState, cfg), nil
	})

	ad := adapterhelper.PrepareSeededAdapter(t, "../../testdata/adapter/test_adapter", mqtt, factory, adapter.ThingSeeds{seed})

	return thing.RouteSensor(ad), thing.TaskSensor(ad, duration), mocks
}
