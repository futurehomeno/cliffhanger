package router_test

import (
	"testing"

	"github.com/futurehomeno/fimpgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/cliffhanger/router"
)

func TestVoters(t *testing.T) {
	t.Parallel()

	const topic = "pt:j1/mt:cmd/rt:dev/rn:zigbee/ad:1/sv:out_bin_switch/ad:2"

	addr, err := fimpgo.NewAddressFromString(topic)
	require.NoError(t, err)

	payload := fimpgo.NewNullMessage("cmd.binary.set", "out_bin_switch", fimpgo.Props{"mode": "eco"}, nil, nil)
	payload.Source = "vinculum"
	msg := &fimpgo.Message{Topic: topic, Addr: addr, Payload: payload}

	accept := router.MessageVoterFn(func(*fimpgo.Message) bool { return true })
	reject := router.MessageVoterFn(func(*fimpgo.Message) bool { return false })

	tcs := []struct {
		name  string
		voter router.MessageVoter
		want  bool
	}{
		{"topic match", router.ForTopic(topic), true},
		{"topic mismatch", router.ForTopic("pt:j1/mt:cmd/rt:dev/rn:zigbee/ad:1"), false},
		{"message type", router.ForMessageType("cmd"), true},
		{"message type mismatch", router.ForMessageType("evt"), false},
		{"resource type", router.ForResourceType("dev"), true},
		{"resource name", router.ForResourceName("zigbee"), true},
		{"resource name mismatch", router.ForResourceName("zwave"), false},
		{"service", router.ForService("out_bin_switch"), true},
		{"service prefix", router.ForServicePrefix("out_"), true},
		{"service prefix mismatch", router.ForServicePrefix("bin"), false},
		{"type", router.ForType("cmd.binary.set"), true},
		{"source", router.ForSource("vinculum"), true},
		{"source mismatch", router.ForSource("kind-owl"), false},
		{"service and type", router.ForServiceAndType("out_bin_switch", "cmd.binary.set"), true},
		{"service and wrong type", router.ForServiceAndType("out_bin_switch", "cmd.binary.get_report"), false},
		{"property", router.ForProperty("mode", "eco"), true},
		{"property other value", router.ForProperty("mode", "comfort"), false},
		{"property absent", router.ForProperty("duration", ""), false},
		{"or with one accepting", router.Or(reject, accept), true},
		{"or with none accepting", router.Or(reject, reject), false},
		{"empty or", router.Or(), false},
		{"not", router.Not(reject), true},
		{"not accepting", router.Not(accept), false},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, tc.voter.Vote(msg))
		})
	}
}

func TestCombine(t *testing.T) {
	t.Parallel()

	a := router.NewRouting(nil)
	b := router.NewRouting(nil)
	c := router.NewRouting(nil)

	assert.Equal(t, []*router.Routing{a, b, c}, router.Combine([]*router.Routing{a}, []*router.Routing{b, c}))
	assert.Equal(t, []*router.Routing{c, a}, router.Combine(c, a))
	assert.Equal(t, []string{"a", "b", "c"}, router.CombineTopicPatterns([]string{"a"}, nil, []string{"b", "c"}))
}
