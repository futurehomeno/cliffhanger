package presence_test

import (
	"testing"
	"time"

	"github.com/futurehomeno/fimpgo"

	"github.com/futurehomeno/cliffhanger/adapter/service/presence"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
	adapterhelper "github.com/futurehomeno/cliffhanger/test/helper/adapter"
	mockedpresence "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/presence"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

func TestTaskReporting(t *testing.T) { //nolint:paralleltest
	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "Presence reporting task",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: taskPresence(
					mockedpresence.NewController(t).
						MockSensorPresenceReport(true, nil, true).
						MockSensorPresenceReport(false, nil, true).
						MockSensorPresenceReport(false, errTest, true).
						MockSensorPresenceReport(false, nil, false), // repeatable: dedup ensures only one evt.presence.report fires even if this fires on every remaining tick
					100*time.Millisecond,
				),
				Nodes: []*suite.Node{
					{
						Name:    "One change and one error during four report cycles",
						Timeout: 500 * time.Millisecond,
						Expectations: []*suite.Expectation{
							suite.ExpectBool("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:sensor_presence/ad:2", "evt.presence.report", "sensor_presence", true).ExactlyOnce(),
							suite.ExpectBool("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:sensor_presence/ad:2", "evt.presence.report", "sensor_presence", false).ExactlyOnce(),
							// forces the node to wait the full timeout so all four controller calls have a chance to fire.
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:sensor_presence/ad:2", "sensor_presence").Never(),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func taskPresence(controller presence.Controller, interval time.Duration) suite.BaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
		t.Helper()

		_, tasks, mocks := setupService(t, mqtt, controller, interval)

		return nil, tasks, mocks
	}
}
