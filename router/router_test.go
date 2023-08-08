package router_test

import (
	"errors"
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

func Test_Router_OptionalSuccessConfirmation(t *testing.T) { //nolint:paralleltest
	successConfirmationRouting := func(messageType string, message *fimpgo.FimpMessage, err error) *router.Routing {
		return router.NewRouting(router.NewMessageHandler(router.MessageProcessorFn(
			func(*fimpgo.Message) (*fimpgo.FimpMessage, error) {
				return message, err
			}), router.WithSuccessConfirmation()),
			router.ForService("test_service"),
			router.ForType(messageType),
		)
	}

	noConfirmationRouting := func(messageType string, message *fimpgo.FimpMessage, err error) *router.Routing {
		return router.NewRouting(router.NewMessageHandler(router.MessageProcessorFn(
			func(*fimpgo.Message) (*fimpgo.FimpMessage, error) {
				return message, err
			})),
			router.ForService("test_service"),
			router.ForType(messageType),
		)
	}

	timeout := 100 * time.Millisecond

	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name: "Test optional success confirmation on routing",
				Routing: []*router.Routing{
					successConfirmationRouting("cmd.test.confirm1", nil, nil),
					successConfirmationRouting("cmd.test.confirm2", fimpgo.NewStringMessage("evt.test.test_event", "test_service", "test", nil, nil, nil), nil),
					successConfirmationRouting("cmd.test.confirm3", nil, errors.New("oops")),
				},
				Nodes: []*suite.Node{
					{
						Name:    "Message processor returns nil - send success confirmation",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.test.confirm1", "test_service"),
						Expectations: []*suite.Expectation{
							suite.ExpectNull("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.success.report", "test_service").ExactlyOnce(),
						},
						Timeout: timeout,
					},
					{
						Name:    "Message processor returns a message - do not send success confirmation",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.test.confirm2", "test_service"),
						Expectations: []*suite.Expectation{
							suite.ExpectNull("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.success.report", "test_service").Never(),
							suite.ExpectString("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.test.test_event", "test_service", "test").ExactlyOnce(),
						},
						Timeout: timeout,
					},
					{
						Name:    "Error returned by processor cannot trigger success confirmation",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.test.confirm3", "test_service"),
						Expectations: []*suite.Expectation{
							suite.ExpectNull("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.success.report", "test_service").Never(),
							suite.ExpectError("pt:j1/mt:evt/rt:app/rn:test/ad:1", "test_service").ExactlyOnce(),
						},
						Timeout: timeout,
					},
				},
			},
			{
				Name: "Test cases success confirmation on routing is not enabled",
				Routing: []*router.Routing{
					noConfirmationRouting("cmd.test.do_not_confirm1", nil, nil),
					noConfirmationRouting("cmd.test.do_not_confirm2", fimpgo.NewStringMessage("evt.test.test_event", "test_service", "test", nil, nil, nil), nil),
					noConfirmationRouting("cmd.test.do_not_confirm3", nil, errors.New("oops")),
				},
				Nodes: []*suite.Node{
					{
						Name:    "Message processor returns nil - do not send success confirmation",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.test.do_not_confirm1", "test_service"),
						Expectations: []*suite.Expectation{
							suite.ExpectNull("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.success.report", "test_service").Never(),
						},
						Timeout: timeout,
					},
					{
						Name:    "Message processor returns a message - do not send success confirmation",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.test.do_not_confirm2", "test_service"),
						Expectations: []*suite.Expectation{
							suite.ExpectNull("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.success.report", "test_service").Never(),
							suite.ExpectString("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.test.test_event", "test_service", "test").ExactlyOnce(),
						},
						Timeout: timeout,
					},
					{
						Name:    "Error returned by processor cannot trigger success confirmation",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.test.do_not_confirm3", "test_service"),
						Expectations: []*suite.Expectation{
							suite.ExpectNull("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.success.report", "test_service").Never(),
							suite.ExpectError("pt:j1/mt:evt/rt:app/rn:test/ad:1", "test_service").ExactlyOnce(),
						},
						Timeout: timeout,
					},
				},
			},
		},
	}

	s.Run(t)
}
