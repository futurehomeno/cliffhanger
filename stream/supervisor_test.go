package stream_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/backoff"
	"github.com/futurehomeno/cliffhanger/stream"
)

func TestSupervisor(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32

	connect := func(ctx context.Context, connected func()) error {
		calls.Add(1)
		connected()
		<-ctx.Done()

		return nil
	}

	s := stream.NewSupervisor(connect, backoff.NewStateful(time.Hour, time.Hour, time.Hour, 1, 1))

	assert.NoError(t, s.Start())
	assert.Error(t, s.Start(), "second start should fail")

	assert.Eventually(t, func() bool { return calls.Load() == 1 }, time.Second, time.Millisecond)

	s.TriggerReconnect()
	assert.Eventually(t, func() bool { return calls.Load() == 2 }, time.Second, time.Millisecond,
		"trigger should drop the connection and skip the backoff wait")

	assert.NoError(t, s.Stop())
	assert.NoError(t, s.Stop(), "stop should be idempotent")

	total := calls.Load()
	time.Sleep(20 * time.Millisecond)
	assert.Equal(t, total, calls.Load(), "no reconnects after stop")
}

func TestSupervisor_ReconnectsAfterFailure(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32

	connect := func(context.Context, func()) error {
		calls.Add(1)

		return errors.New("connection err")
	}

	s := stream.NewSupervisor(connect, backoff.NewStateful(time.Millisecond, time.Millisecond, time.Millisecond, 1, 1))

	assert.NoError(t, s.Start())
	assert.Eventually(t, func() bool { return calls.Load() >= 3 }, time.Second, time.Millisecond,
		"failed connections should be retried with backoff")
	assert.NoError(t, s.Stop())
}
