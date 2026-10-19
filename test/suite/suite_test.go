package suite_test

import (
	"testing"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
	cliffSuite "github.com/futurehomeno/cliffhanger/test/suite"
)

type fakeService struct {
	start func() error
	stop  func() error
}

func (s *fakeService) Start() error { return s.start() }
func (s *fakeService) Stop() error  { return s.stop() }

func TestSuite_CaseLifecycle(t *testing.T) { //nolint:paralleltest
	topic := "pt:j1/mt:evt/rt:dev/rn:suite_test/ad:1/sv:out_bin_switch/ad:1"

	var (
		mqtt   *fimpgo.MqttTransport
		events []string
	)

	service := &fakeService{
		start: func() error {
			events = append(events, "service start")

			return mqtt.PublishToTopic(topic, fimpgo.NewBoolMessage("evt.binary.report", "out_bin_switch", true, nil, nil, nil))
		},
		stop: func() error {
			events = append(events, "service stop")

			return nil
		},
	}

	c := cliffSuite.NewCase("lifecycle").
		WithSetup(cliffSuite.BaseSetup(func(_ *testing.T, m *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []cliffSuite.Mock) {
			mqtt = m

			return nil, nil, nil
		})).
		WithService(service).
		WithNodes(
			cliffSuite.NewNode("report emitted on start").
				WithInitCallbacks(func(*testing.T) { events = append(events, "node init") }).
				WithExpectations(cliffSuite.ExpectBool(topic, "evt.binary.report", "out_bin_switch", true).ExactlyOnce()),
		).
		WithTearDown(func(*testing.T) { events = append(events, "teardown") })

	cliffSuite.NewSuite().WithCases(c).Run(t)

	assert.Equal(t, []string{"service start", "node init", "service stop", "teardown"}, events)
}

func TestSuite_ExpectationMatching(t *testing.T) { //nolint:paralleltest
	cmdTopic := "pt:j1/mt:cmd/rt:dev/rn:suite_test/ad:1/sv:out_bin_switch/ad:2"
	evtTopic := "pt:j1/mt:evt/rt:dev/rn:suite_test/ad:1/sv:out_bin_switch/ad:2"

	node := cliffSuite.NewNode("command answered").
		WithTimeout(300*time.Millisecond).
		WithCommand(cliffSuite.BoolMessage(cmdTopic, "cmd.binary.set", "out_bin_switch", false)).
		WithCommandFn(func(*testing.T) *fimpgo.Message {
			return cliffSuite.BoolMessage(cmdTopic, "cmd.binary.set", "out_bin_switch", true)
		}).
		WithExpectations(
			cliffSuite.ExpectBool(cmdTopic, "cmd.binary.set", "out_bin_switch", true).
				ExactlyOnce().
				PublishInResponse(cliffSuite.BoolMessage(evtTopic, "evt.binary.report", "out_bin_switch", true)),
			cliffSuite.ExpectBool(evtTopic, "evt.binary.report", "out_bin_switch", true).ExactlyOnce(),
			cliffSuite.NewExpectation(router.ForTopic(cmdTopic)).Never(),
		)

	start := time.Now()

	cliffSuite.NewSuite().WithCases(cliffSuite.NewCase("matching").WithNodes(node)).Run(t)

	assert.GreaterOrEqual(t, time.Since(start), 300*time.Millisecond, "a Never expectation waits out the whole timeout")
}
