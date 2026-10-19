package router

import (
	"testing"

	"github.com/futurehomeno/fimpgo"
	"github.com/stretchr/testify/assert"
)

func TestRouter_WithOptions(t *testing.T) {
	t.Parallel()

	tcs := []struct {
		name   string
		option Option
		want   *config
	}{
		{
			name:   "With preserved global prefix",
			option: WithPreservedGlobalPrefix(),
			want: &config{
				buffer:               10,
				concurrency:          5,
				preserveGlobalPrefix: true,
			},
		},
		{
			name:   "Sync processing",
			option: WithSyncProcessing(),
			want: &config{
				buffer:      10,
				concurrency: 1,
			},
		},
		{
			name:   "Async processing",
			option: WithAsyncProcessing(3),
			want: &config{
				buffer:      10,
				concurrency: 3,
			},
		},
		{
			name:   "Async processing with incorrect value",
			option: WithAsyncProcessing(-3),
			want: &config{
				buffer:      10,
				concurrency: 5,
			},
		},
		{
			name:   "Message buffer",
			option: WithMessageBuffer(3),
			want: &config{
				buffer:      3,
				concurrency: 5,
			},
		},
		{
			name:   "Message buffer",
			option: WithMessageBuffer(-3),
			want: &config{
				buffer:      10,
				concurrency: 5,
			},
		},
	}

	for _, tc := range tcs {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			r, ok := NewRouter(nil, "").(*router)

			assert.True(t, ok)

			r.WithOptions(tc.option)

			assert.Equal(t, tc.want, r.cfg)
		})
	}
}

// valueHandler is a MessageHandler implemented on a value receiver, which the interface allows.
type valueHandler struct{}

func (valueHandler) Handle(*fimpgo.Message) *fimpgo.Message { return nil }

// TestIsNilHandler pins that the missing-handler guard tolerates handlers that are not pointers.
// reflect.Value.IsNil panics for a struct value, so such a handler used to panic on every message
// and have it dropped by the recover in processMessage.
func TestIsNilHandler(t *testing.T) {
	t.Parallel()

	var nilPointer *messageHandler

	assert.True(t, isNilHandler(nil))
	assert.True(t, isNilHandler(nilPointer))
	assert.False(t, isNilHandler(valueHandler{}))
	assert.False(t, isNilHandler(NewMessageHandler(MessageProcessorFn(
		func(*fimpgo.Message) (*fimpgo.FimpMessage, error) { return nil, nil },
	))))
}

func TestRouting_WrapPrependsWithoutMutatingReceiver(t *testing.T) {
	t.Parallel()

	var order []string

	voter := func(name string, result bool) MessageVoter {
		return MessageVoterFn(func(*fimpgo.Message) bool {
			order = append(order, name)

			return result
		})
	}

	original := NewRouting(nil, voter("inner", true))
	wrapped := original.Wrap(voter("outer", true))

	assert.True(t, wrapped.vote(&fimpgo.Message{}))
	assert.Equal(t, []string{"outer", "inner"}, order)

	order = nil

	assert.False(t, original.Wrap(voter("reject", false)).vote(&fimpgo.Message{}))
	assert.Equal(t, []string{"reject"}, order, "voting stops at the first rejecting voter")

	assert.Len(t, original.voters, 1)
	assert.True(t, NewRouting(nil).vote(&fimpgo.Message{}), "a routing without voters accepts every message")
}
