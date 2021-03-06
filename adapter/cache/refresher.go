package cache

import (
	"fmt"
	"sync"
	"time"
)

// DefaultOffset is the default percentage offset of the refresh interval.
const defaultOffset = 0.05

// Refresher is a helper service that performs refreshing if the a configured interval has passed.
type Refresher interface {
	// Refresh refreshes data if required and returns it.
	Refresh() (interface{}, error)
	// Reset cache so next invocation will result in execution of provided refresh function.
	Reset()
}

// OffsetInterval is a helper to offset interval with a safety margin.
func OffsetInterval(interval time.Duration) time.Duration {
	return time.Duration((1 - defaultOffset) * float64(interval))
}

// NewRefresher creates new instance of a refresher service.
func NewRefresher(refresh func() (interface{}, error), interval time.Duration) Refresher {
	return &refresher{
		lock:     &sync.Mutex{},
		interval: interval,
		refresh:  refresh,
	}
}

// refresher is a private implementation of the refresher service.
type refresher struct {
	lock        *sync.Mutex
	interval    time.Duration
	value       interface{}
	lastRefresh time.Time
	refresh     func() (interface{}, error)
}

// Refresh refreshes data if required and returns it.
func (r *refresher) Refresh() (interface{}, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	if !r.lastRefresh.IsZero() && time.Since(r.lastRefresh) < r.interval {
		return r.value, nil
	}

	val, err := r.refresh()
	if err != nil {
		return nil, fmt.Errorf("refresher: failed to refresh data: %w", err)
	}

	r.value = val
	r.lastRefresh = time.Now()

	return r.value, nil
}

// Reset cache so next invocation will result in execution of provided refresh function.
func (r *refresher) Reset() {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.value = nil
	r.lastRefresh = time.Time{}
}
