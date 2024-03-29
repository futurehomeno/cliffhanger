package cache

import (
	"fmt"
	"sync"
	"time"
)

// constants defining default option values for the refresher.
const (
	defaultFailureThreshold = 2
	defaultBackoffThreshold = 3
	defaultOffset           = 0.05
	defaultInitialBackoff   = 15 * time.Second
	defaultRepeatedBackoff  = 1 * time.Minute
	defaultFinalBackoff     = 5 * time.Minute
)

type Refresh[T any] func() (T, error)

// Refresher is a helper service that performs refreshing if the a configured interval has passed.
type Refresher[T any] interface {
	// Refresh refreshes data if required and returns it.
	Refresh() (T, error)
	// Reset cache so next invocation will result in execution of provided refresh function.
	Reset()
	// IsFailing returns true if the refreshers failure count exceeded configured threshold and false otherwise.
	IsFailing() bool
}

// RefresherOption is an option for refresher service.
type RefresherOption interface {
	// apply applies refresher option to the provided refresher.
	apply(*refresherOptions)
}

// refresherOptionFn is a function adapter that allows to use anonymous functions as refresher option.
type refresherOptionFn func(*refresherOptions)

// apply applies refresher option to the provided refresher.
func (f refresherOptionFn) apply(r *refresherOptions) {
	f(r)
}

// WithDefaultBackoff sets default backoff parameters for the refresher.
func WithDefaultBackoff() RefresherOption {
	return WithBackoff(defaultInitialBackoff, defaultRepeatedBackoff, defaultFinalBackoff, defaultBackoffThreshold)
}

// WithBackoff sets backoff parameters for the refresher.
func WithBackoff(initialBackoff, repeatedBackoff, finalBackoff time.Duration, backoffThreshold int) RefresherOption {
	return refresherOptionFn(func(r *refresherOptions) {
		r.initialBackoff = initialBackoff
		r.repeatedBackoff = repeatedBackoff
		r.finalBackoff = finalBackoff
		r.backoffThreshold = backoffThreshold
	})
}

// WithDefaultFailureThreshold sets failure threshold for the refresher.
func WithDefaultFailureThreshold() RefresherOption {
	return WithFailureThreshold(defaultFailureThreshold)
}

// WithFailureThreshold sets failure threshold for the refresher.
func WithFailureThreshold(threshold int) RefresherOption {
	return refresherOptionFn(func(r *refresherOptions) {
		r.failureThreshold = threshold
	})
}

// WithDefaultIntervalOffset sets default interval offset for the refresher.
func WithDefaultIntervalOffset() RefresherOption {
	return WithIntervalOffset(defaultOffset)
}

// WithIntervalOffset sets interval offset for the refresher.
func WithIntervalOffset(offset float64) RefresherOption {
	return refresherOptionFn(func(r *refresherOptions) {
		r.interval = time.Duration((1 - offset) * float64(r.interval))
	})
}

// WithDefaultOptions sets default options for the refresher.
func WithDefaultOptions() RefresherOption {
	return refresherOptionFn(func(r *refresherOptions) {
		WithDefaultBackoff().apply(r)
		WithDefaultFailureThreshold().apply(r)
		WithDefaultIntervalOffset().apply(r)
	})
}

// NewRefresher creates new instance of a refresher service.
func NewRefresher[T any](refresh Refresh[T], interval time.Duration, options ...RefresherOption) Refresher[T] {
	r := &refresher[T]{
		refresherOptions: refresherOptions{
			interval: interval,
		},
		refresh: refresh,
	}

	for _, option := range options {
		option.apply(&r.refresherOptions)
	}

	return r
}

// refresher is a private implementation of the refresher service.
type refresher[T any] struct {
	lock sync.Mutex
	refresherOptions

	value        T
	lastRefresh  time.Time
	lastFailure  time.Time
	failureCount int

	refresh Refresh[T]
}

type refresherOptions struct {
	interval         time.Duration
	failureThreshold int
	backoffThreshold int
	initialBackoff   time.Duration
	repeatedBackoff  time.Duration
	finalBackoff     time.Duration
}

// Refresh refreshes data if required and returns it.
func (r *refresher[T]) Refresh() (T, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	if !r.lastRefresh.IsZero() && time.Since(r.lastRefresh) < r.interval {
		return r.value, nil
	}

	if r.shouldBackoff() {
		var def T

		return def, fmt.Errorf("refresher: backoff is in effect")
	}

	val, err := r.refresh()
	if err != nil {
		r.lastFailure = time.Now()
		r.failureCount++

		var def T

		return def, fmt.Errorf("refresher: failed to refresh data: %w", err)
	}

	r.value = val
	r.lastRefresh = time.Now()
	r.failureCount = 0
	r.lastFailure = time.Time{}

	return r.value, nil
}

// Reset cache so next invocation will result in execution of provided refresh function.
func (r *refresher[T]) Reset() {
	r.lock.Lock()
	defer r.lock.Unlock()

	var def T
	r.value = def
	r.lastRefresh = time.Time{}
}

// IsFailing returns true if the refreshers failure count exceeded configured threshold and false otherwise.
func (r *refresher[T]) IsFailing() bool {
	r.lock.Lock()
	defer r.lock.Unlock()

	return r.failureCount > r.failureThreshold
}

// shouldBackoff returns true if the backoff is in effect and false otherwise.
func (r *refresher[T]) shouldBackoff() bool {
	if r.backoffThreshold == 0 {
		return false
	}

	if r.lastFailure.IsZero() {
		return false
	}

	return time.Since(r.lastFailure) < r.getBackoff(r.failureCount)
}

// getBackoff returns backoff duration based on the provided failure count.
func (r *refresher[T]) getBackoff(failureCount int) time.Duration {
	if failureCount <= r.backoffThreshold {
		return r.initialBackoff
	}

	if failureCount <= 2*r.backoffThreshold {
		return r.repeatedBackoff
	}

	return r.finalBackoff
}
