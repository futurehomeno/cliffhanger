package router_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/futurehomeno/fimpgo"
	log "github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/cliffhanger/router"
)

func newHandlerRequest(t *testing.T, topic string) *fimpgo.Message {
	t.Helper()

	addr, err := fimpgo.NewAddressFromString(topic)
	require.NoError(t, err)

	return &fimpgo.Message{
		Topic:   topic,
		Addr:    addr,
		Payload: fimpgo.NewNullMessage("cmd.test.do", testServiceName, nil, nil, nil),
	}
}

func failingProcessor(err error) router.MessageProcessor {
	return router.MessageProcessorFn(func(*fimpgo.Message) (*fimpgo.FimpMessage, error) {
		return nil, err
	})
}

func TestMessageHandler_ErrorReportContract(t *testing.T) {
	t.Parallel()

	tcs := []struct {
		name        string
		topic       string
		wantTopic   string
		wantStorage *fimpgo.Storage
	}{
		{
			name:        "device error is not stored",
			topic:       "pt:j1/mt:cmd/rt:dev/rn:test/ad:1/sv:test_service/ad:2",
			wantTopic:   "pt:j1/mt:evt/rt:dev/rn:test/ad:1/sv:test_service/ad:2",
			wantStorage: &fimpgo.Storage{Strategy: fimpgo.StorageStrategySkip},
		},
		{
			name:      "app error keeps the default storage",
			topic:     "pt:j1/mt:cmd/rt:app/rn:test/ad:1",
			wantTopic: "pt:j1/mt:evt/rt:app/rn:test/ad:1",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			request := newHandlerRequest(t, tc.topic)
			reply := router.NewMessageHandler(failingProcessor(errors.New("boom"))).Handle(request)

			require.NotNil(t, reply)
			assert.Equal(t, tc.wantTopic, reply.Addr.Serialize())
			assert.Equal(t, router.EvtErrorReport, reply.Payload.Interface)
			assert.Equal(t, testServiceName, reply.Payload.Service)
			assert.Equal(t, "failed to process incoming message", reply.Payload.Value)
			assert.Equal(t, fimpgo.Props{
				router.PropertyMsg:        "boom",
				router.PropertyCmdTopic:   tc.topic,
				router.PropertyCmdService: testServiceName.Str(),
				router.PropertyCmdType:    "cmd.test.do",
			}, reply.Payload.Properties)
			assert.Equal(t, tc.wantStorage, reply.Payload.Storage)
		})
	}
}

func TestMessageHandler_ErrorLogLevel(t *testing.T) {
	hook := logtest.NewGlobal()
	defer hook.Reset()

	tcs := []struct {
		name string
		err  error
		want log.Level
	}{
		{name: "not reported yet is a warning", err: fmt.Errorf("sensor: %w", router.ErrNotReported), want: log.WarnLevel},
		{name: "other failure is an error", err: errors.New("boom"), want: log.ErrorLevel},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			hook.Reset()

			request := newHandlerRequest(t, "pt:j1/mt:cmd/rt:app/rn:test/ad:1")
			reply := router.NewMessageHandler(failingProcessor(tc.err)).Handle(request)

			require.NotNil(t, reply)
			assert.Equal(t, router.EvtErrorReport, reply.Payload.Interface)
			require.NotNil(t, hook.LastEntry())
			assert.Equal(t, tc.want, hook.LastEntry().Level)
		})
	}
}

func TestMessageHandler_SilentErrors(t *testing.T) {
	t.Parallel()

	request := newHandlerRequest(t, "pt:j1/mt:cmd/rt:app/rn:test/ad:1")
	handler := router.NewMessageHandler(failingProcessor(errors.New("boom")), router.WithSilentErrors())

	assert.Nil(t, handler.Handle(request))
}

func TestMessageHandler_DefaultAddress(t *testing.T) {
	t.Parallel()

	defaultAddr, err := fimpgo.NewAddressFromString("pt:j1/mt:cmd/rt:app/rn:other/ad:1")
	require.NoError(t, err)

	handler := router.NewMessageHandler(router.MessageProcessorFn(func(*fimpgo.Message) (*fimpgo.FimpMessage, error) {
		return fimpgo.NewNullMessage("evt.test.report", testServiceName, nil, nil, nil), nil
	}), router.WithDefaultAddress(defaultAddr))

	reply := handler.Handle(newHandlerRequest(t, "pt:j1/mt:cmd/rt:dev/rn:test/ad:1/sv:test_service/ad:2"))

	require.NotNil(t, reply)
	assert.Equal(t, "pt:j1/mt:evt/rt:app/rn:other/ad:1", reply.Addr.Serialize())
}

func TestMessageHandler_LockRejectsConcurrentMessage(t *testing.T) {
	t.Parallel()

	tcs := []struct {
		name   string
		silent bool
		build  func(router.MessageProcessor) (first, second router.MessageHandler)
	}{
		{
			name: "private lock",
			build: func(p router.MessageProcessor) (router.MessageHandler, router.MessageHandler) {
				h := router.NewMessageHandler(p, router.WithLock())

				return h, h
			},
		},
		{
			name:   "shared external lock",
			silent: true,
			build: func(p router.MessageProcessor) (router.MessageHandler, router.MessageHandler) {
				locker := router.NewMessageHandlerLocker()

				return router.NewMessageHandler(p, router.WithExternalLock(locker)),
					router.NewMessageHandler(p, router.WithExternalLock(locker), router.WithSilentErrors())
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			entered := make(chan struct{})
			release := make(chan struct{})
			calls := 0

			processor := router.MessageProcessorFn(func(*fimpgo.Message) (*fimpgo.FimpMessage, error) {
				calls++
				if calls == 1 {
					close(entered)
					<-release

					return nil, errors.New("first fails")
				}

				return fimpgo.NewNullMessage("evt.test.done", testServiceName, nil, nil, nil), nil
			})

			first, second := tc.build(processor)
			request := newHandlerRequest(t, "pt:j1/mt:cmd/rt:app/rn:test/ad:1")

			done := make(chan *fimpgo.Message)

			go func() { done <- first.Handle(request) }()

			<-entered

			rejected := second.Handle(request)
			if !tc.silent {
				require.NotNil(t, rejected)
				assert.Equal(t, router.EvtErrorReport, rejected.Payload.Interface)
				assert.Equal(t, "another operation is already running, skipping message", rejected.Payload.Properties[router.PropertyMsg])
			} else {
				assert.Nil(t, rejected)
			}

			close(release)
			<-done

			assert.Equal(t, 1, calls, "a rejected message must not reach the processor")

			reply := second.Handle(request)
			require.NotNil(t, reply, "the lock must be released after a failed run")
			assert.Equal(t, "evt.test.done", reply.Payload.Interface)
			assert.Equal(t, 2, calls)
		})
	}
}

func TestMessageHandlerLocker(t *testing.T) {
	t.Parallel()

	locker := router.NewMessageHandlerLocker()

	assert.True(t, locker.Lock())
	assert.False(t, locker.Lock())

	locker.Unlock()

	assert.True(t, locker.Lock())
}
