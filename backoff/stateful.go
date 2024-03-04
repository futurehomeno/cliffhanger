package backoff

import (
	"sync/atomic"
	"time"
)

// Stateful is a backoff strategy that keeps track of the number of consecutive failures and the time of the last of them.
type Stateful interface {
	// Next returns the backoff delay based on the number of consecutive failures. It also increments the failure counter.
	Next() time.Duration
	// Reset resets backoff failure counter.
	Reset()
	// Fail increments the failure counter.
	Fail()
	// Should returns true if the backoff should be applied or not based on the time of last failure and the number of consecutive failures.
	Should() bool
}

// NewStateful creates a new stateful backoff strategy.
func NewStateful(
	initialBackoff, repeatedBackoff, finalBackoff time.Duration,
	initialFailureCount, repeatedFailureCount uint32,
) Stateful {
	return &stateful{
		backoff: New(initialBackoff, repeatedBackoff, finalBackoff, initialFailureCount, repeatedFailureCount),
	}
}

// stateful is a backoff strategy that keeps track of the number of consecutive failures and the time of the last of them.
type stateful struct {
	backoff  Backoff
	failures atomic.Uint32
}

// Next returns the backoff delay based on the number of consecutive failures. It also increments the failure counter.
func (e *stateful) Next() time.Duration {
	failures := e.failures.Add(1)

	return e.backoff.Delay(failures)
}

// Fail increments the failure counter.
func (e *stateful) Fail() {
	e.failures.Add(1)
}

// Should returns true if the backoff should be applied or not based on the time of last failure and the number of consecutive failures.
func (e *stateful) Should() bool {
	failures := e.failures.Load()

	return e.backoff.Should(time.Now(), failures)
}

// Reset resets backoff failure counter.
func (e *stateful) Reset() {
	e.failures.Swap(0)
}
