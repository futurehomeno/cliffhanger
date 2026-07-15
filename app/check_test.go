package app_test

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/app"
	"github.com/futurehomeno/cliffhanger/backoff"
	"github.com/futurehomeno/cliffhanger/httpclient"
	"github.com/futurehomeno/cliffhanger/lifecycle"
)

type fakeReporter struct {
	reports atomic.Int32
}

func (r *fakeReporter) SendConnectivityReport() error {
	r.reports.Add(1)

	return nil
}

func TestConnectivityChecker_Check(t *testing.T) { //nolint:funlen
	t.Parallel()

	probeErr := errors.New("probe err")

	testCases := []struct {
		name        string
		probeErrs   []error
		wantAuth    lifecycle.State
		wantConn    lifecycle.State
		wantReports int32
	}{
		{
			name:        "success marks authenticated and connected",
			probeErrs:   []error{nil},
			wantAuth:    lifecycle.AuthStateAuthenticated,
			wantConn:    lifecycle.ConnStateConnected,
			wantReports: 1,
		},
		{
			name:        "unauthorized marks auth lost and disconnected",
			probeErrs:   []error{fmt.Errorf("wrap: %w", httpclient.ErrUnauthorized)},
			wantAuth:    lifecycle.AuthStateLost,
			wantConn:    lifecycle.ConnStateDisconnected,
			wantReports: 1,
		},
		{
			name:        "rate limit leaves states untouched",
			probeErrs:   []error{nil, httpclient.ErrTooManyRequests},
			wantAuth:    lifecycle.AuthStateAuthenticated,
			wantConn:    lifecycle.ConnStateConnected,
			wantReports: 1,
		},
		{
			name:        "rate limit with retry-after leaves states untouched",
			probeErrs:   []error{nil, &httpclient.TooManyRequestsError{RetryAfter: time.Hour}},
			wantAuth:    lifecycle.AuthStateAuthenticated,
			wantConn:    lifecycle.ConnStateConnected,
			wantReports: 1,
		},
		{
			name:        "single transient failure does not disconnect",
			probeErrs:   []error{nil, probeErr},
			wantAuth:    lifecycle.AuthStateAuthenticated,
			wantConn:    lifecycle.ConnStateConnected,
			wantReports: 1,
		},
		{
			name:        "second consecutive failure disconnects",
			probeErrs:   []error{nil, probeErr, probeErr},
			wantAuth:    lifecycle.AuthStateAuthenticated,
			wantConn:    lifecycle.ConnStateDisconnected,
			wantReports: 2,
		},
		{
			name:        "recovery after failure reconnects",
			probeErrs:   []error{nil, probeErr, probeErr, nil},
			wantAuth:    lifecycle.AuthStateAuthenticated,
			wantConn:    lifecycle.ConnStateConnected,
			wantReports: 3,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			errs := tc.probeErrs
			call := 0
			probe := func() error {
				err := errs[call]
				call++

				return err
			}

			lc := lifecycle.New(nil)
			reporter := &fakeReporter{}

			checker := app.NewConnectivityChecker(probe, lc, reporter, app.CheckerConfig{
				// An hour-long recheck delay ensures scheduled rechecks never fire within the test.
				RecheckBackoff: backoff.New(time.Hour, time.Hour, time.Hour, 1, 1),
				RateLimitDelay: time.Hour,
			})

			for range errs {
				assert.NoError(t, checker.CheckNow())
			}

			checker.Cancel()

			assert.Equal(t, tc.wantAuth, lc.AuthState())
			assert.Equal(t, tc.wantConn, lc.ConnectionState())
			assert.Equal(t, tc.wantReports, reporter.reports.Load())
		})
	}
}

func TestConnectivityChecker_RepairsStrandedAppHealth(t *testing.T) {
	t.Parallel()

	lc := lifecycle.New(nil)
	// A transient failure during authorization strands the app in NOT_CONFIGURED;
	// the periodic check task does not run in this state, so the successful
	// recheck must repair the app health, not just connectivity.
	lc.SetAppHealth(lifecycle.AppHealthNotConfigured, nil)
	lc.SetConfigState(lifecycle.ConfigStateNotConfigured)

	checker := app.NewConnectivityChecker(func() error { return nil }, lc, nil, app.CheckerConfig{})

	assert.NoError(t, checker.Check())

	assert.Equal(t, lifecycle.AppHealthRunning, lc.AppHealth())
	assert.Equal(t, lifecycle.ConfigStateConfigured, lc.ConfigState())
	assert.Equal(t, lifecycle.ConnStateConnected, lc.ConnectionState())
	assert.Equal(t, lifecycle.AuthStateAuthenticated, lc.AuthState())
}

func TestConnectivityChecker_CancelDiscardsInFlightProbe(t *testing.T) {
	t.Parallel()

	entered := make(chan struct{})
	release := make(chan struct{})

	probe := func() error {
		close(entered)
		<-release

		return nil
	}

	lc := lifecycle.New(nil)
	reporter := &fakeReporter{}
	checker := app.NewConnectivityChecker(probe, lc, reporter, app.CheckerConfig{})

	wantAuth, wantConn := lc.AuthState(), lc.ConnectionState()

	done := make(chan struct{})

	go func() {
		defer close(done)

		assert.NoError(t, checker.Check())
	}()

	<-entered
	checker.Cancel()
	close(release)
	<-done

	assert.Equal(t, wantAuth, lc.AuthState(), "stale probe result should not change the auth state")
	assert.Equal(t, wantConn, lc.ConnectionState(), "stale probe result should not change the connectivity state")
	assert.Equal(t, int32(0), reporter.reports.Load())
}

func TestConnectivityChecker_ShortRetryAfterHonored(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32

	probe := func() error {
		if calls.Add(1) == 1 {
			return &httpclient.TooManyRequestsError{RetryAfter: time.Millisecond}
		}

		return nil
	}

	checker := app.NewConnectivityChecker(probe, lifecycle.New(nil), nil, app.CheckerConfig{
		RateLimitDelay: time.Hour,
	})

	assert.NoError(t, checker.Check())

	assert.Eventually(t, func() bool { return calls.Load() >= 2 }, time.Second, 5*time.Millisecond,
		"recheck should honor a server Retry-After shorter than the configured fallback delay")

	checker.Cancel()
}

func TestConnectivityChecker_PendingRecheckSkipsPeriodicProbe(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32

	probe := func() error {
		calls.Add(1)

		return errors.New("probe err")
	}

	checker := app.NewConnectivityChecker(probe, lifecycle.New(nil), nil, app.CheckerConfig{
		RecheckBackoff: backoff.New(time.Hour, time.Hour, time.Hour, 1, 1),
	})

	assert.NoError(t, checker.Check())
	assert.NoError(t, checker.Check())
	assert.Equal(t, int32(1), calls.Load(), "periodic probe should be skipped while a recheck is pending")

	checker.Cancel()

	assert.NoError(t, checker.Check())
	assert.Equal(t, int32(2), calls.Load(), "probe should resume once the pending recheck is canceled")
}

func TestConnectivityChecker_ConcurrentChecks(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32

	probe := func() error {
		if calls.Add(1)%2 == 0 {
			return errors.New("probe err")
		}

		return nil
	}

	checker := app.NewConnectivityChecker(probe, lifecycle.New(nil), &fakeReporter{}, app.CheckerConfig{
		RecheckBackoff: backoff.New(time.Hour, time.Hour, time.Hour, 1, 1),
	})

	var wg sync.WaitGroup

	for range 8 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			assert.NoError(t, checker.Check())
		}()
	}

	wg.Wait()

	assert.Positive(t, calls.Load())

	checker.Cancel()
}

func TestConnectivityChecker_Recheck(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32

	probe := func() error {
		calls.Add(1)

		return errors.New("probe err")
	}

	checker := app.NewConnectivityChecker(probe, lifecycle.New(nil), nil, app.CheckerConfig{
		RecheckBackoff: backoff.New(time.Millisecond, time.Millisecond, time.Millisecond, 1, 1),
	})

	assert.NoError(t, checker.Check())

	assert.Eventually(t, func() bool { return calls.Load() >= 2 }, time.Second, 5*time.Millisecond,
		"scheduled recheck should re-run the probe")

	checker.Cancel()
}
