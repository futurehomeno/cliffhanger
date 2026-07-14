package app

import (
	"errors"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/cliffhanger/backoff"
	"github.com/futurehomeno/cliffhanger/httpclient"
	"github.com/futurehomeno/cliffhanger/lifecycle"
	"github.com/futurehomeno/cliffhanger/utils"
)

// ConnectivityReporter broadcasts availability of all things, e.g. evt.network.all_nodes_report.
// It is satisfied by adapter.Adapter. Things reachable over a local transport keep reporting UP
// through their own Connector.Connectivity() even when the third party API is down.
type ConnectivityReporter interface {
	SendConnectivityReport() error
}

// CheckerConfig configures the ConnectivityChecker recheck behavior.
type CheckerConfig struct {
	// Interval between periodic checks. 0 uses DefaultCheckInterval.
	Interval time.Duration
	// RecheckBackoff provides delays between rechecks following failed probes.
	RecheckBackoff backoff.Backoff
	// MaxRechecks is the number of consecutive probe failures tolerated before reporting disconnection.
	MaxRechecks uint32
	// RateLimitDelay is the recheck delay after a rate-limited probe without a Retry-After indication.
	RateLimitDelay time.Duration
}

func (c *CheckerConfig) setDefaults() {
	if c.RecheckBackoff == nil {
		c.RecheckBackoff = backoff.New(time.Minute, 5*time.Minute, 15*time.Minute, 1, 2)
	}

	if c.MaxRechecks == 0 {
		c.MaxRechecks = 1
	}

	if c.RateLimitDelay == 0 {
		c.RateLimitDelay = 5 * time.Minute
	}
}

// ConnectivityChecker implements CheckableApp by probing the third party API and mapping the
// outcome onto lifecycle states. The probe should wrap httpclient.ErrUnauthorized on authorization
// loss and httpclient.ErrTooManyRequests on rate limiting. Other errors are retried with backoff
// and reported as disconnection only after more than MaxRechecks consecutive failures. On every
// connectivity or authorization transition all nodes are reported via the ConnectivityReporter.
type ConnectivityChecker struct {
	cfg      CheckerConfig
	probe    func() error
	lc       *lifecycle.Lifecycle
	reporter ConnectivityReporter
	warnLog  utils.Throttle

	mu       sync.Mutex
	timer    *time.Timer
	failures uint32
}

func NewConnectivityChecker(
	probe func() error,
	appLifecycle *lifecycle.Lifecycle,
	reporter ConnectivityReporter,
	cfg CheckerConfig,
) *ConnectivityChecker {
	cfg.setDefaults()

	return &ConnectivityChecker{
		cfg:      cfg,
		probe:    probe,
		lc:       appLifecycle,
		reporter: reporter,
	}
}

func (c *ConnectivityChecker) CheckInterval() time.Duration {
	return c.cfg.Interval
}

func (c *ConnectivityChecker) Check() error {
	err := c.probe()
	if err == nil {
		c.Cancel()
		c.apply(lifecycle.AuthStateAuthenticated, lifecycle.ConnStateConnected)

		return nil
	}

	if errors.Is(err, httpclient.ErrUnauthorized) {
		c.Cancel()
		c.apply(lifecycle.AuthStateLost, lifecycle.ConnStateDisconnected)

		return nil
	}

	// A rate limit does not mean connectivity is lost, so the connectivity state is left untouched.
	if errors.Is(err, httpclient.ErrTooManyRequests) {
		c.Cancel()
		c.schedule(c.rateLimitDelay(err))

		return nil
	}

	c.warnLog.Do(err.Error(), func() { log.Warnf("[app] Check probe err: %v", err) })

	failures := c.fail()
	if failures > c.cfg.MaxRechecks {
		c.apply("", lifecycle.ConnStateDisconnected)
	}

	c.schedule(c.cfg.RecheckBackoff.Delay(failures))

	return nil
}

// Cancel stops a pending recheck and clears the failure counter; call it on logout and reset.
func (c *ConnectivityChecker) Cancel() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.failures = 0
	c.warnLog.Reset()

	if c.timer != nil {
		c.timer.Stop()
		c.timer = nil
	}
}

// rateLimitDelay honors a server-requested Retry-After delay if it exceeds the configured one.
func (c *ConnectivityChecker) rateLimitDelay(err error) time.Duration {
	delay := c.cfg.RateLimitDelay

	var rateLimitErr *httpclient.TooManyRequestsError
	if errors.As(err, &rateLimitErr) && rateLimitErr.RetryAfter > delay {
		return rateLimitErr.RetryAfter
	}

	return delay
}

// apply sets the provided lifecycle states (empty auth leaves it untouched) and
// broadcasts node availability if any of them changed.
func (c *ConnectivityChecker) apply(auth, conn lifecycle.State) {
	changed := false

	if auth != "" && c.lc.AuthState() != auth {
		c.lc.SetAuthState(auth)

		changed = true
	}

	if c.lc.ConnectionState() != conn {
		c.lc.SetConnState(conn)

		changed = true
	}

	if !changed || c.reporter == nil {
		return
	}

	if err := c.reporter.SendConnectivityReport(); err != nil {
		log.Errorf("[app] Send connectivity report err: %v", err)
	}
}

// schedule sets a one-shot recheck, skipped if one is already pending.
func (c *ConnectivityChecker) schedule(delay time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.timer != nil {
		return
	}

	log.Infof("[app] Check failed, retrying in %s", delay)

	// The callback compares timer identity to detect a Cancel that raced its firing.
	var timer *time.Timer

	timer = time.AfterFunc(delay, func() {
		c.mu.Lock()
		if c.timer != timer {
			c.mu.Unlock()

			return
		}
		c.timer = nil
		c.mu.Unlock()

		_ = c.Check()
	})
	c.timer = timer
}

func (c *ConnectivityChecker) fail() uint32 {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.failures++

	return c.failures
}
