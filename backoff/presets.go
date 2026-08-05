package backoff

import "time"

// NewTolerantFixed returns a stateful backoff applying no delay for the first tolerance
// consecutive failures (absorbing transient blips) and a fixed delay afterwards.
func NewTolerantFixed(tolerance uint32, delay time.Duration) Stateful {
	return NewStateful(0, delay, delay, tolerance, tolerance)
}
