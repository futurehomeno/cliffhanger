package app

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/lifecycle"
)

// TestSchedule_AfterCancelDoesNotResume pins the teardown fix. A Cancel landing after a failing
// probe cleared its staleness check but before it schedules the recheck finds no timer to stop,
// so schedule() must decline to arm one: otherwise the recheck resumes a checker that Logout or
// Reset tore down (edge-mill, edge-hoiax) and writes lifecycle state after MarkNotConfigured.
func TestSchedule_AfterCancelDoesNotResume(t *testing.T) {
	t.Parallel()

	// Atomic because the probe would run on the timer goroutine in the buggy case, racing
	// this goroutine's read below.
	var probes atomic.Int64

	c := NewConnectivityChecker(
		func() error { probes.Add(1); return nil },
		lifecycle.New(nil),
		nil,
		CheckerConfig{},
	)

	// The ordering the race produces: Cancel commits, then the in-flight failure path schedules.
	c.Cancel()
	c.schedule(time.Millisecond)

	time.Sleep(100 * time.Millisecond)

	assert.Zero(t, probes.Load(), "a recheck scheduled after Cancel must not probe")
	assert.True(t, c.stale(), "a recheck must not clear the cancelled flag")
}

// The recheck timer snapshots the cancelled flag, unlocks, then probes. A Cancel landing in
// that gap must still stop the probe: the result would be discarded either way, but the call
// itself must not reach the third-party API after a logout or reset.
func TestCheck_CancelBetweenSnapshotAndProbeDoesNotProbe(t *testing.T) {
	t.Parallel()

	var probes atomic.Int64

	c := NewConnectivityChecker(
		func() error { probes.Add(1); return nil },
		lifecycle.New(nil),
		nil,
		CheckerConfig{},
	)

	// Exactly the state the race leaves behind: the caller already read cancelled as false and
	// released the lock, and Cancel commits before check() runs.
	c.Cancel()
	c.check()

	assert.Zero(t, probes.Load(), "check must re-read the cancelled flag before probing")
}
