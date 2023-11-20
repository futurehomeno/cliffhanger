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

var ()

func Test_Router_PanicCallback(t *testing.T) { //nolint:paralleltest
	var panicCallbackCalled bool

	tearDownFn := func(t *testing.T) {
		t.Helper()

		panicCallbackCalled = false
	}

	panicCallback := func(msg *fimpgo.Message, err interface{}) {
		panicCallbackCalled = true
	}

	panicRouting := router.NewRouting(router.NewMessageHandler(
		router.MessageProcessorFn(
			func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
				panic("oops")
			})),
		router.ForService("test_service"),
	)
	noPanicRouting := router.NewRouting(router.NewMessageHandler(
		router.MessageProcessorFn(
			func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
				return nil, nil
			})),
		router.ForService("test_service"),
	)

	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "panic callback",
				TearDown: []suite.Callback{tearDownFn},
				Routing:  []*router.Routing{panicRouting},
				RouterOptions: []router.Option{
					router.WithPanicCallback(panicCallback),
				},
				Nodes: []*suite.Node{
					{
						Name:    "send a command raising panic",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.test.test_command", "test_service"),
						Timeout: -1,
					},
					suite.SleepNode(10 * time.Millisecond),
					{
						Name:    "verify panic callback was called",
						Timeout: -1,
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()

								assert.True(t, panicCallbackCalled)
							},
						},
					},
				},
			},
			{
				Name:     "no panic callback",
				TearDown: []suite.Callback{tearDownFn},
				Routing:  []*router.Routing{noPanicRouting},
				RouterOptions: []router.Option{
					router.WithPanicCallback(panicCallback),
				},
				Nodes: []*suite.Node{
					{
						Name:    "send a command not raising panic",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.test.test_command", "test_service"),
						Timeout: -1,
					},
					suite.SleepNode(10 * time.Millisecond),
					{
						Name:    "panic callback cannot be called",
						Timeout: -1,
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()

								assert.False(t, panicCallbackCalled)
							},
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func Test_Router_ProcessingCallback(t *testing.T) { //nolint:paralleltest
	var callbackCalled bool

	tearDownFn := func(t *testing.T) {
		t.Helper()

		callbackCalled = false
	}

	callbackFn := func(msg *fimpgo.Message) {
		callbackCalled = true
	}

	routing := router.NewRouting(router.NewMessageHandler(
		router.MessageProcessorFn(
			func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
				return nil, nil
			})),
		router.ForService("test_service"),
	)

	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "processing callback",
				TearDown: []suite.Callback{tearDownFn},
				Routing:  []*router.Routing{routing},
				RouterOptions: []router.Option{
					router.WithMessageProcessingCallback(callbackFn),
				},
				Nodes: []*suite.Node{
					{
						Name:    "send a command that should be processed",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.test.test_command", "test_service"),
						Timeout: -1,
					},
					suite.SleepNode(10 * time.Millisecond),
					{
						Name:    "verify processing callback was called",
						Timeout: -1,
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()

								assert.True(t, callbackCalled)
							},
						},
					},
				},
			},
			{
				Name:     "no processing callback",
				TearDown: []suite.Callback{tearDownFn},
				Routing:  []*router.Routing{routing},
				RouterOptions: []router.Option{
					router.WithMessageProcessingCallback(callbackFn),
				},
				Nodes: []*suite.Node{
					{
						Name:    "send a command that should not be processed",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:2", "cmd.test.do_not_process", "non_test_service"),
						Timeout: -1,
					},
					suite.SleepNode(10 * time.Millisecond),
					{
						Name:    "processing callback cannot be called",
						Timeout: -1,
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()

								assert.False(t, callbackCalled)
							},
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func Test_Router_ResponseCallback(t *testing.T) { //nolint:paralleltest
	var callbackCalled bool

	tearDownFn := func(t *testing.T) {
		t.Helper()

		callbackCalled = false
	}

	callbackFn := func(in, out *fimpgo.Message) {
		callbackCalled = true
	}

	responseRouting := router.NewRouting(router.NewMessageHandler(
		router.MessageProcessorFn(
			func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
				return fimpgo.NewStringMessage("evt.test.test_event", "test_service", "test_value", nil, nil, message.Payload), nil
			})),
		router.ForService("test_service"),
		router.ForType("cmd.test.test_command"),
	)
	noResponseRouting := router.NewRouting(router.NewMessageHandler(
		router.MessageProcessorFn(
			func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
				return nil, nil
			})),
		router.ForService("test_service"),
		router.ForType("cmd.test.test_command"),
	)

	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "processing callback",
				TearDown: []suite.Callback{tearDownFn},
				Routing:  []*router.Routing{responseRouting},
				RouterOptions: []router.Option{
					router.WithResponseCallback(callbackFn),
				},
				Nodes: []*suite.Node{
					{
						Name:    "send a command that should result with a response",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.test.test_command", "test_service"),
						Timeout: -1,
					},
					suite.SleepNode(10 * time.Millisecond),
					{
						Name:    "verify response callback was called",
						Timeout: -1,
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()

								assert.True(t, callbackCalled)
							},
						},
					},
				},
			},
			{
				Name:     "no processing callback",
				TearDown: []suite.Callback{tearDownFn},
				Routing:  []*router.Routing{noResponseRouting},
				RouterOptions: []router.Option{
					router.WithResponseCallback(callbackFn),
				},
				Nodes: []*suite.Node{
					{
						Name:    "send a command that should not result with a response",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:2", "cmd.test.test_command", "test_service"),
						Timeout: -1,
					},
					suite.SleepNode(10 * time.Millisecond),
					{
						Name:    "response callback cannot be called",
						Timeout: -1,
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()

								assert.False(t, callbackCalled)
							},
						},
					},
				},
			},
		},
	}

	s.Run(t)
}
