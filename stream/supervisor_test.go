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

func TestSupervisor_StartDuringStop(t *testing.T) {
	t.Parallel()

	release := make(chan struct{})

	var calls atomic.Int32

	connect := func(ctx context.Context, connected func()) error {
		calls.Add(1)
		connected()
		<-ctx.Done()
		<-release

		return nil
	}

	s := stream.NewSupervisor(connect, backoff.NewStateful(time.Hour, time.Hour, time.Hour, 1, 1))

	assert.NoError(t, s.Start())
	assert.Eventually(t, func() bool { return calls.Load() == 1 }, time.Second, time.Millisecond)

	stopped := make(chan struct{})

	go func() {
		assert.NoError(t, s.Stop())
		close(stopped)
	}()

	time.Sleep(20 * time.Millisecond)
	assert.Error(t, s.Start(), "start must fail until the previous connection loop has fully exited")
	assert.Equal(t, int32(1), calls.Load(), "no overlapping connection may be started")

	close(release)
	<-stopped

	assert.NoError(t, s.Start(), "start should succeed after stop has completed")
	assert.Eventually(t, func() bool { return calls.Load() == 2 }, time.Second, time.Millisecond)
	assert.NoError(t, s.Stop())
}

func TestSupervisor_StopDuringBackoffWait(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32

	connect := func(context.Context, func()) error {
		calls.Add(1)

		return errors.New("connection err")
	}

	s := stream.NewSupervisor(connect, backoff.NewStateful(time.Hour, time.Hour, time.Hour, 1, 1))

	assert.NoError(t, s.Start())
	assert.Eventually(t, func() bool { return calls.Load() == 1 }, time.Second, time.Millisecond)

	assert.NoError(t, s.Stop())
	assert.Equal(t, int32(1), calls.Load(), "stop during the backoff wait must not dial again")
}

// TestSupervisor_TriggerDoesNotAdvanceBackoff pins that a triggered reconnect is not counted as
// a failure: an attempt failing right after the trigger must start at the initial delay, not the
// second tier.
func TestSupervisor_TriggerDoesNotAdvanceBackoff(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32

	connect := func(ctx context.Context, connected func()) error {
		if calls.Add(1) == 1 {
			connected()
			<-ctx.Done()

			return nil
		}

		return errors.New("connection err")
	}

	s := stream.NewSupervisor(connect, backoff.NewStateful(time.Millisecond, time.Hour, time.Hour, 1, 1))

	assert.NoError(t, s.Start())
	assert.Eventually(t, func() bool { return calls.Load() == 1 }, time.Second, time.Millisecond)

	s.TriggerReconnect()
	assert.Eventually(t, func() bool { return calls.Load() >= 3 }, time.Second, time.Millisecond,
		"a failure right after a triggered reconnect must retry at the initial delay")
	assert.NoError(t, s.Stop())
}

// resetAwareBackoff waits an hour until Reset and a millisecond after, making a test hang unless
// the supervisor resets the backoff at the point under test.
type resetAwareBackoff struct {
	wasReset atomic.Bool
}

func (b *resetAwareBackoff) Next() time.Duration {
	if b.wasReset.Load() {
		return time.Millisecond
	}

	return time.Hour
}

func (b *resetAwareBackoff) Fail()        {}
func (b *resetAwareBackoff) Should() bool { return false }
func (b *resetAwareBackoff) Reset()       { b.wasReset.Store(true) }

// TestSupervisor_TriggerDuringWaitResetsBackoff pins that a trigger landing during the backoff
// wait also grants the next attempt a fresh backoff, no matter how the wait was cut short.
func TestSupervisor_TriggerDuringWaitResetsBackoff(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32

	connect := func(context.Context, func()) error {
		calls.Add(1)

		return errors.New("connection err")
	}

	s := stream.NewSupervisor(connect, &resetAwareBackoff{})

	assert.NoError(t, s.Start())
	assert.Eventually(t, func() bool { return calls.Load() == 1 }, time.Second, time.Millisecond)

	time.Sleep(10 * time.Millisecond)
	s.TriggerReconnect()
	assert.Eventually(t, func() bool { return calls.Load() >= 3 }, time.Second, time.Millisecond,
		"a failure after a trigger during the wait must retry at the initial delay")
	assert.NoError(t, s.Stop())
}

// TestSupervisor_CleanEndDoesNotAdvanceBackoff pins that a cleanly ended connection is paced but
// not counted as a failure: a failing attempt after it must start at the initial delay.
func TestSupervisor_CleanEndDoesNotAdvanceBackoff(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32

	connect := func(_ context.Context, connected func()) error {
		if calls.Add(1) == 1 {
			connected()

			return nil
		}

		return errors.New("connection err")
	}

	s := stream.NewSupervisor(connect, backoff.NewStateful(time.Millisecond, time.Hour, time.Hour, 1, 1))

	assert.NoError(t, s.Start())
	assert.Eventually(t, func() bool { return calls.Load() >= 3 }, time.Second, time.Millisecond,
		"a failure after a clean ending must retry at the initial delay")
	assert.NoError(t, s.Stop())
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
