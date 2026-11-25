package router_test

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"testing"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
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
			assert.Equal(t, "internal", reply.Payload.Value)
			assert.Equal(t, fimpgo.Props{
				router.PropertyMsg:        "boom",
				router.PropertyCmdTopic:   tc.topic,
				router.PropertyCmdService: testServiceName.Str(),
				router.PropertyCmdType:    "cmd.test.do",
			}, reply.Payload.Properties)
			assert.Equal(t, tc.wantStorage, reply.Payload.Storage)
			assert.Equal(t, request.Payload.UID, reply.Payload.CorrelationID)
		})
	}
}

func TestMessageHandler_ErrorCodes(t *testing.T) {
	t.Parallel()

	refused := router.NewError(fimptype.ErrorCodeDeviceRefused, "The device refused the level.", errors.New("status 409"))
	urlErr := &url.Error{Op: "Get", URL: "https://10.0.0.5/api/KEY/lights/1", Err: context.DeadlineExceeded}

	tcs := []struct {
		name      string
		err       error
		wantValue string
		wantMsg   string
	}{
		{
			name:      "coded error",
			err:       router.NewError(fimptype.ErrorCodeOutOfRange, "Current must not exceed 16 A.", nil),
			wantValue: "out_of_range",
			wantMsg:   "Current must not exceed 16 A.",
		},
		{
			name:      "code found deep in the chain",
			err:       fmt.Errorf("controller: %w", fmt.Errorf("set level: %w", refused)),
			wantValue: "device_refused",
			wantMsg:   "The device refused the level.",
		},
		{
			name:      "adapter's own code",
			err:       router.NewError("eco_mode", "Eco mode is on.", nil),
			wantValue: "eco_mode",
			wantMsg:   "Eco mode is on.",
		},
		{
			name:      "URL removed from an uncoded chain",
			err:       fmt.Errorf("get light: %w", urlErr),
			wantValue: "internal",
			wantMsg:   "get light: Get: context deadline exceeded",
		},
		{
			name:      "URL error without a cause",
			err:       fmt.Errorf("get light: %w", &url.Error{Op: "Get", URL: "https://10.0.0.5/api/KEY/lights/1"}),
			wantValue: "internal",
			wantMsg:   "get light: Get: <nil>",
		},
		{
			name:      "error without a code answers internal",
			err:       &router.Error{Msg: "Eco mode is on.", NoLog: true},
			wantValue: "internal",
			wantMsg:   "Eco mode is on.",
		},
		{
			name:      "state not reported yet",
			err:       fmt.Errorf("sensor: %w", router.ErrNotReported),
			wantValue: "unavailable",
			wantMsg:   "state not reported yet",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			reply := router.NewMessageHandler(failingProcessor(tc.err)).Handle(newHandlerRequest(t, "pt:j1/mt:cmd/rt:app/rn:test/ad:1"))

			require.NotNil(t, reply)
			assert.Equal(t, tc.wantValue, reply.Payload.Value)
			assert.Equal(t, tc.wantMsg, reply.Payload.Properties[router.PropertyMsg])
		})
	}
}

func TestError(t *testing.T) {
	t.Parallel()

	cause := errors.New("status 409")
	err := router.NewError(fimptype.ErrorCodeDeviceRefused, "The device refused the level.", cause)

	assert.Equal(t, "The device refused the level.: status 409", err.Error())
	assert.ErrorIs(t, err, cause)
	assert.Equal(t, "Eco mode is on.", router.NewError("eco_mode", "Eco mode is on.", nil).Error())
}

func TestNewErrorReport(t *testing.T) {
	hook := logtest.NewGlobal()
	defer log.StandardLogger().ReplaceHooks(make(log.LevelHooks))

	tcs := []struct {
		name      string
		topic     string
		respTo    string
		wantTopic string
		wantSkip  bool
	}{
		{
			name:      "device resource",
			topic:     "pt:j1/mt:cmd/rt:dev/rn:test/ad:1/sv:test_service/ad:2",
			wantTopic: "pt:j1/mt:evt/rt:dev/rn:test/ad:1/sv:test_service/ad:2",
			wantSkip:  true,
		},
		{
			name:      "response topic wins",
			topic:     "pt:j1/mt:cmd/rt:dev/rn:test/ad:1/sv:test_service/ad:2",
			respTo:    "pt:j1/mt:rsp/rt:app/rn:caller/ad:1",
			wantTopic: "pt:j1/mt:rsp/rt:app/rn:caller/ad:1",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			hook.Reset()

			request := newHandlerRequest(t, tc.topic)
			request.Payload.ResponseToTopic = tc.respTo

			report := router.NewErrorReport(request, router.NewError(fimptype.ErrorCodeDeviceRefused, "The cloud refused the write.", nil))

			require.NotNil(t, report)
			assert.Equal(t, tc.wantTopic, report.Addr.Serialize())
			assert.Equal(t, router.EvtErrorReport, report.Payload.Interface)
			assert.Equal(t, "device_refused", report.Payload.Value)
			assert.Equal(t, request.Payload.UID, report.Payload.CorrelationID)
			assert.Equal(t, fimpgo.Props{
				router.PropertyMsg:        "The cloud refused the write.",
				router.PropertyCmdTopic:   tc.topic,
				router.PropertyCmdService: testServiceName.Str(),
				router.PropertyCmdType:    "cmd.test.do",
			}, report.Payload.Properties)
			assert.Equal(t, tc.wantSkip, report.Payload.Storage != nil && report.Payload.Storage.Strategy == fimpgo.StorageStrategySkip)
			require.Len(t, hook.AllEntries(), 1)
			assert.Equal(t, log.WarnLevel, hook.LastEntry().Level)
		})
	}
}

func TestNewErrorReport_InvalidResponseTopic(t *testing.T) {
	hook := logtest.NewGlobal()
	defer log.StandardLogger().ReplaceHooks(make(log.LevelHooks))

	request := newHandlerRequest(t, "pt:j1/mt:cmd/rt:dev/rn:test/ad:1/sv:test_service/ad:2")
	request.Payload.ResponseToTopic = "pt:j1/mt:rsp/garbage"

	assert.Nil(t, router.NewErrorReport(request, router.NewError(fimptype.ErrorCodeDeviceRefused, "The cloud refused the write.", nil)))
	require.Len(t, hook.AllEntries(), 1)
	assert.Equal(t, log.ErrorLevel, hook.LastEntry().Level)
}

func TestMessageHandler_ErrorLogLevel(t *testing.T) {
	hook := logtest.NewGlobal()
	defer log.StandardLogger().ReplaceHooks(make(log.LevelHooks))

	tcs := []struct {
		name string
		err  error
		want log.Level
	}{
		{name: "not reported yet is a warning", err: fmt.Errorf("sensor: %w", router.ErrNotReported), want: log.WarnLevel},
		{name: "unauthorized is a warning", err: router.NewError(fimptype.ErrorCodeUnauthorized, "The cloud refused the credentials.", errors.New("401")), want: log.WarnLevel},
		{name: "other failure is an error", err: errors.New("boom"), want: log.ErrorLevel},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			hook.Reset()

			request := newHandlerRequest(t, "pt:j1/mt:cmd/rt:app/rn:test/ad:1")
			reply := router.NewMessageHandler(failingProcessor(tc.err)).Handle(request)

			require.NotNil(t, reply)
			assert.Equal(t, router.EvtErrorReport, reply.Payload.Interface)
			require.Len(t, hook.AllEntries(), 1)
			assert.Equal(t, tc.want, hook.LastEntry().Level)
			assert.Contains(t, hook.LastEntry().Message, tc.err.Error())
		})
	}
}

func TestMessageHandler_RepeatedRefusalNotLogged(t *testing.T) {
	hook := logtest.NewGlobal()
	defer log.StandardLogger().ReplaceHooks(make(log.LevelHooks))

	err := router.NewError("eco_mode", "Eco mode is on.", nil)
	err.NoLog = true

	reply := router.NewMessageHandler(failingProcessor(err)).Handle(newHandlerRequest(t, "pt:j1/mt:cmd/rt:app/rn:test/ad:1"))

	require.NotNil(t, reply)
	assert.Equal(t, "eco_mode", reply.Payload.Value)
	assert.Empty(t, hook.AllEntries())
}

func TestMessageHandler_SilentErrors(t *testing.T) {
	t.Parallel()

	request := newHandlerRequest(t, "pt:j1/mt:cmd/rt:app/rn:test/ad:1")
	handler := router.NewMessageHandler(failingProcessor(errors.New("boom")), router.WithSilentErrors())

	assert.Nil(t, handler.Handle(request))

	request.Addr = nil

	assert.Nil(t, handler.Handle(request), "a silent failure needs no address to answer to")
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
				assert.Equal(t, "busy", rejected.Payload.Value)
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
