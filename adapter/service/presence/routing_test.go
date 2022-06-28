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
	mockedpresence "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/presence"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

func TestRoutePresence(t *testing.T) { //nolint:paralleltest
	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name: "successful get report",
				Setup: routePresence(
					mockedpresence.NewController(t).
						MockPresencePresenceReport(true, nil, true),
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
				Name: "failed get report",
				Setup: routePresence(
					mockedpresence.NewController(t).
						MockPresencePresenceReport(false, errors.New("error"), true),
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
						Name: "wrond address",
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

func TestTaskPresence(t *testing.T) { //nolint:paralleltest
	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name: "Presence thing tasks",
				Setup: taskPresence(
					mockedpresence.NewController(t).
						MockPresencePresenceReport(true, nil, true).
						MockPresencePresenceReport(true, errors.New("task error"), true).
						MockPresencePresenceReport(false, nil, true).
						MockPresencePresenceReport(false, nil, false),
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
) suite.BaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
		t.Helper()

		routing, _, mocks := setupPresence(t, mqtt, presenceController, 0)

		return routing, nil, mocks
	}
}

func taskPresence(
	presenceController *mockedpresence.Controller,
	interval time.Duration,
) suite.BaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
		t.Helper()

		_, tasks, mocks := setupPresence(t, mqtt, presenceController, interval)

		return nil, tasks, mocks
	}
}

func setupPresence(
	t *testing.T,
	mqtt *fimpgo.MqttTransport,
	presenceController *mockedpresence.Controller,
	duration time.Duration,
) ([]*router.Routing, []*task.Task, []suite.Mock) {
	t.Helper()

	mocks := []suite.Mock{presenceController}

	cfg := &PresenceThingConfig{
		InclusionReport: &fimptype.ThingInclusionReport{
			Address: "2",
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

	motionSensor := newPresenceThing(mqtt, cfg)
	ad := adapter.NewAdapter(nil, "test_adapter", "1")
	ad.RegisterThing(motionSensor)

	return routePresenceThing(ad), taskPresenceThing(ad, duration), mocks
}

// ThingConfig represents a config for testing precence service.
type PresenceThingConfig struct {
	InclusionReport *fimptype.ThingInclusionReport
	PresenceConfig  *presence.Config
}

// newPresenceThing creates a thing that can be used for testing presence service.
func newPresenceThing(
	mqtt *fimpgo.MqttTransport,
	cfg *PresenceThingConfig,
) adapter.Thing {
	services := []adapter.Service{
		presence.NewService(mqtt, cfg.PresenceConfig),
	}

	return adapter.NewThing(cfg.InclusionReport, services...)
}

// routePresenceThing creates a thing that can be used for testing presence service.
func routePresenceThing(adapter adapter.Adapter) []*router.Routing {
	return router.Combine(
		presence.RouteService(adapter),
	)
}

// taskPresenceThing creates background tasks specific for presence service.
func taskPresenceThing(
	adapter adapter.Adapter,
	reportingInterval time.Duration,
	reportingVoter ...task.Voter,
) []*task.Task {
	return []*task.Task{
		presence.TaskReporting(adapter, reportingInterval, reportingVoter...),
	}
}
