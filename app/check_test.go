package app_test

import (
	"errors"
	"fmt"
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
				assert.NoError(t, checker.Check())
			}

			checker.Cancel()

			assert.Equal(t, tc.wantAuth, lc.AuthState())
			assert.Equal(t, tc.wantConn, lc.ConnectionState())
			assert.Equal(t, tc.wantReports, reporter.reports.Load())
		})
	}
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
