package backoff

import (
	"time"
)

// Backoff is a simple backoff strategy based on the number of failures and three thresholds.
type Backoff interface {
	// Should returns true if the backoff should be applied or not based on the time of last failure and the number of consecutive failures.
	Should(lastFailure time.Time, failureCount uint32) bool
	// Delay returns the backoff delay based on the number of consecutive failures.
	Delay(failureCount uint32) time.Duration
}

// New creates a new backoff strategy.
func New(
	initialBackoff, repeatedBackoff, finalBackoff time.Duration,
	initialFailureCount, repeatedFailureCount uint32,
) Backoff {
	return &backoff{
		initialBackoff:       initialBackoff,
		repeatedBackoff:      repeatedBackoff,
		finalBackoff:         finalBackoff,
		initialFailureCount:  initialFailureCount,
		repeatedFailureCount: repeatedFailureCount,
	}
}

// backoff is a simple backoff strategy based on the number of failures and three thresholds.
type backoff struct {
	initialBackoff       time.Duration
	repeatedBackoff      time.Duration
	finalBackoff         time.Duration
	initialFailureCount  uint32
	repeatedFailureCount uint32
}

// Should returns true if the backoff should be applied or not based on the time of last failure and the number of consecutive failures.
func (b *backoff) Should(lastFailure time.Time, failureCount uint32) bool {
	if lastFailure.IsZero() {
		return false
	}

	backoffDelay := b.Delay(failureCount)

	return lastFailure.Add(backoffDelay).After(time.Now())
}

// Delay returns the backoff delay based on the number of consecutive failures.
func (b *backoff) Delay(failureCount uint32) time.Duration {
	if failureCount <= b.initialFailureCount {
		return b.initialBackoff
	}

	if failureCount-b.initialFailureCount <= b.repeatedFailureCount {
		return b.repeatedBackoff
	}

	return b.finalBackoff
}
