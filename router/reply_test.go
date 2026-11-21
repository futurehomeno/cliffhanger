package router_test

import (
	"testing"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

const (
	replyCmdTopic      = "pt:j1/mt:cmd/rt:app/rn:test/ad:1"
	replyInferredTopic = "pt:j1/mt:evt/rt:app/rn:test/ad:1"
	replyRespToTopic   = "pt:j1/mt:rsp/rt:app/rn:requester/ad:1"
)

func Test_Router_ReplyAddressAndCorrelation(t *testing.T) { //nolint:paralleltest
	routeReply := func(iface string, reply func(*fimpgo.Message) *fimpgo.Message) *router.Routing {
		return router.NewRouting(router.MessageHandlerFn(reply), router.ForService(testServiceName), router.ForType(iface))
	}

	ownAddress := func(m *fimpgo.Message) *fimpgo.Message {
		addr, _ := fimpgo.NewAddressFromString(replyInferredTopic)

		return &fimpgo.Message{Addr: addr, Payload: fimpgo.NewStringMessage("evt.test.reply", testServiceName, m.Payload.Interface, nil, nil, nil)}
	}

	presetCorrelation := func(m *fimpgo.Message) *fimpgo.Message {
		reply := ownAddress(m)
		reply.Payload.CorrelationID = "preset"

		return reply
	}

	noAddress := func(m *fimpgo.Message) *fimpgo.Message {
		return &fimpgo.Message{Payload: fimpgo.NewStringMessage("evt.test.reply", testServiceName, m.Payload.Interface, nil, nil, nil)}
	}

	withRespTo := func(iface, respTo string) *fimpgo.Message {
		m := suite.NullMessage(replyCmdTopic, iface, testServiceName)
		m.Payload.ResponseToTopic = respTo

		return m
	}

	correlated := suite.NullMessage(replyCmdTopic, "cmd.test.correlated", testServiceName)
	respTo := withRespTo("cmd.test.resp_to", replyRespToTopic)

	reply := func(topic, value string) *suite.Expectation {
		return suite.ExpectString(topic, "evt.test.reply", testServiceName, value)
	}

	corID := func(want func() string) router.MessageVoter {
		return router.MessageVoterFn(func(m *fimpgo.Message) bool { return m.Payload.CorrelationID == want() })
	}

	timeout := 200 * time.Millisecond

	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name: "reply address resolution",
				Routing: []*router.Routing{
					routeReply("cmd.test.correlated", ownAddress),
					routeReply("cmd.test.preset", presetCorrelation),
					routeReply("cmd.test.resp_to", ownAddress),
					routeReply("cmd.test.no_address", noAddress),
					routeReply("cmd.test.bad_resp_to", ownAddress),
				},
				Nodes: []*suite.Node{
					{
						Name:    "reply goes to its own address correlated with the request UID",
						Command: correlated,
						Expectations: []*suite.Expectation{
							reply(replyInferredTopic, "cmd.test.correlated").Expect(corID(func() string { return correlated.Payload.UID })).ExactlyOnce(),
						},
						Timeout: timeout,
					},
					{
						Name:    "correlation set by the handler is preserved",
						Command: suite.NullMessage(replyCmdTopic, "cmd.test.preset", testServiceName),
						Expectations: []*suite.Expectation{
							reply(replyInferredTopic, "cmd.test.preset").Expect(corID(func() string { return "preset" })).ExactlyOnce(),
						},
						Timeout: timeout,
					},
					{
						Name:    "response topic wins over the reply address",
						Command: respTo,
						Expectations: []*suite.Expectation{
							reply(replyRespToTopic, "cmd.test.resp_to").Expect(corID(func() string { return respTo.Payload.UID })).ExactlyOnce(),
							reply(replyInferredTopic, "cmd.test.resp_to").Never(),
						},
						Timeout: timeout,
					},
					{
						Name:    "reply without any address is dropped",
						Command: suite.NullMessage(replyCmdTopic, "cmd.test.no_address", testServiceName),
						Expectations: []*suite.Expectation{
							suite.NewExpectation(router.ForType("evt.test.reply")).Never(),
						},
						Timeout: timeout,
					},
					{
						Name:    "unparsable response topic publishes nothing",
						Command: withRespTo("cmd.test.bad_resp_to", "pt:j1/not-a-fimp-segment"),
						Expectations: []*suite.Expectation{
							suite.NewExpectation(router.ForType("evt.test.reply")).Never(),
						},
						Timeout: timeout,
					},
					{
						Name:    "router keeps processing after a dropped reply",
						Command: suite.NullMessage(replyCmdTopic, "cmd.test.correlated", testServiceName),
						Expectations: []*suite.Expectation{
							reply(replyInferredTopic, "cmd.test.correlated").ExactlyOnce(),
						},
						Timeout: timeout,
					},
				},
			},
			{
				Name:          "preserved global prefix",
				RouterOptions: []router.Option{router.WithPreservedGlobalPrefix()},
				Routing:       []*router.Routing{routeReply("cmd.test.prefixed", ownAddress)},
				Nodes: []*suite.Node{
					{
						Name:    "reply carries the request's global prefix",
						Command: suite.NullMessage("gp1/"+replyCmdTopic, "cmd.test.prefixed", testServiceName),
						Expectations: []*suite.Expectation{
							reply("gp1/"+replyInferredTopic, "cmd.test.prefixed").ExactlyOnce(),
							reply(replyInferredTopic, "cmd.test.prefixed").Never(),
						},
						Timeout: timeout,
					},
				},
			},
		},
	}

	s.Run(t)
}

func Test_Router_StartStopTwice(t *testing.T) { //nolint:paralleltest
	r := router.NewRouter(suite.DefaultMQTT("router_start_stop_twice", "", "", ""), "router_start_stop_twice")

	assert.Error(t, r.Stop())
	assert.NoError(t, r.Start())
	assert.Error(t, r.Start())
	assert.NoError(t, r.Stop())
	assert.Error(t, r.Stop())
	assert.NoError(t, r.Start(), "a stopped router can be started again")
	assert.NoError(t, r.Stop())
}
