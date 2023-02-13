package router_test

import (
	"sync"
	"testing"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

func Test_Router(t *testing.T) { //nolint:paralleltest
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

func Test_Router_Concurrency(t *testing.T) { //nolint:paralleltest
	var receivedCommands []string

	handlerLocker := router.NewMessageHandlerLocker()

	lock := &sync.Mutex{}

	routeMessage := func(command string, delay time.Duration, options ...router.MessageHandlerOption) *router.Routing {
		return router.NewRouting(router.NewMessageHandler(
			router.MessageProcessorFn(
				func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
					time.Sleep(delay)
					lock.Lock()
					defer lock.Unlock()

					receivedCommands = append(receivedCommands, command)

					return fimpgo.NewStringMessage("evt.test.test_event", "test_service", command, nil, nil, message.Payload), nil
				}), options...),
			router.ForService("test_service"),
			router.ForType(command),
		)
	}

	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name: "Test async processing",
				RouterOptions: []router.Option{
					router.WithAsyncProcessing(5),
				},
				Routing: []*router.Routing{
					routeMessage("cmd.test.test_command_1", 200*time.Millisecond),
					routeMessage("cmd.test.test_command_2", 50*time.Millisecond),
				},
				Nodes: []*suite.Node{
					{
						Name: "Initialize",
						InitCallbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()
								lock.Lock()
								defer lock.Unlock()

								receivedCommands = []string{}
							},
						},
						Timeout: 1 * time.Nanosecond,
					},
					{
						Name:    "Send command 1",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.test.test_command_1", "test_service"),
					},
					{
						Name:    "Send command 2",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.test.test_command_2", "test_service"),
					},
					{
						Name: "Check commands",
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.test.test_event", "test_service", "cmd.test.test_command_1"),
							suite.ExpectString("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.test.test_event", "test_service", "cmd.test.test_command_2"),
						},
					},
					{
						Name: "Check order",
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()
								lock.Lock()
								defer lock.Unlock()

								assert.Equal(t, []string{"cmd.test.test_command_2", "cmd.test.test_command_1"}, receivedCommands)
							},
						},
						Timeout: 1 * time.Nanosecond,
					},
				},
			},
			{
				Name: "Test sync processing",
				RouterOptions: []router.Option{
					router.WithSyncProcessing(),
				},
				Routing: []*router.Routing{
					routeMessage("cmd.test.test_command_1", 200*time.Millisecond),
					routeMessage("cmd.test.test_command_2", 50*time.Millisecond),
				},
				Nodes: []*suite.Node{
					{
						Name: "Initialize",
						InitCallbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()
								lock.Lock()
								defer lock.Unlock()

								receivedCommands = []string{}
							},
						},
						Timeout: 1 * time.Nanosecond,
					},
					{
						Name:    "Send command 1",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.test.test_command_1", "test_service"),
					},
					{
						Name:    "Send command 2",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.test.test_command_2", "test_service"),
					},
					{
						Name: "Check commands",
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.test.test_event", "test_service", "cmd.test.test_command_1"),
							suite.ExpectString("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.test.test_event", "test_service", "cmd.test.test_command_2"),
						},
					},
					{
						Name: "Check order",
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()
								lock.Lock()
								defer lock.Unlock()

								assert.Equal(t, []string{"cmd.test.test_command_1", "cmd.test.test_command_2"}, receivedCommands)
							},
						},
						Timeout: 1 * time.Nanosecond,
					},
				},
			},
			{
				Name: "Test async processing with concurrency lock",
				RouterOptions: []router.Option{
					router.WithAsyncProcessing(5),
				},
				Routing: []*router.Routing{
					routeMessage("cmd.test.test_command_1", 200*time.Millisecond, router.WithExternalLock(handlerLocker)),
					routeMessage("cmd.test.test_command_2", 50*time.Millisecond, router.WithExternalLock(handlerLocker)),
				},
				Nodes: []*suite.Node{
					{
						InitCallbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()
								lock.Lock()
								defer lock.Unlock()

								receivedCommands = []string{}
							},
						},
						Timeout: 1 * time.Nanosecond,
					},
					{
						Name:    "Send command 1",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.test.test_command_1", "test_service"),
					},
					{
						Name:    "Send command 2",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.test.test_command_2", "test_service"),
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.test.test_event", "test_service", "cmd.test.test_command_1"),
							suite.ExpectError("pt:j1/mt:evt/rt:app/rn:test/ad:1", "test_service"),
						},
					},
					{
						Name: "Check order",
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()
								lock.Lock()
								defer lock.Unlock()

								assert.Equal(t, []string{"cmd.test.test_command_1"}, receivedCommands)
							},
						},
						Timeout: 1 * time.Nanosecond,
					},
				},
			},
		},
	}

	s.Run(t)
}
