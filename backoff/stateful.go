package backoff

import (
	"sync"
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
	backoff     Backoff
	lastFailure time.Time
	failures    uint32

	m sync.Mutex
}

// Next returns the backoff delay based on the number of consecutive failures. It also increments the failure counter.
func (e *stateful) Next() time.Duration {
	e.m.Lock()
	defer e.m.Unlock()

	e.lastFailure = time.Now()
	e.failures++

	return e.backoff.Delay(e.failures)
}

// Fail increments the failure counter.
func (e *stateful) Fail() {
	e.m.Lock()
	defer e.m.Unlock()

	e.lastFailure = time.Now()
	e.failures++
}

// Should returns true if the backoff should be applied or not based on the time of last failure and the number of consecutive failures.
func (e *stateful) Should() bool {
	e.m.Lock()
	defer e.m.Unlock()

	return e.backoff.Should(e.lastFailure, e.failures)
}

// Reset resets backoff failure counter.
func (e *stateful) Reset() {
	e.m.Lock()
	defer e.m.Unlock()

	e.lastFailure = time.Time{}
	e.failures = 0
}
