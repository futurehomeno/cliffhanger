package router_test

import (
	"testing"
	"time"

	"github.com/futurehomeno/fimpgo"

	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

func Test_Router(t *testing.T) {
	panicRouting := router.NewRouting(router.NewMessageHandler(
		router.MessageProcessorFn(
			func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
				panic("test panic")
			})),
		router.ForService("test_service"),
	)

	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:    "Test panic handling",
				Routing: []*router.Routing{panicRouting},
				Nodes: []*suite.Node{
					{
						Name:    "Send command raising panic",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.test.test_command", "test_service"),
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.test.test_event", "test_service", "test_value").Never(),
						},
						Timeout: 250 * time.Millisecond,
					},
				},
			},
		},
	}

	s.Run(t)
}
