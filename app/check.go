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
	// AuthLossState maps an unauthorized probe to an auth state, given the current one. nil
	// keeps the default of always AuthStateLost; return AuthStateNotAuthenticated to stay
	// silent when the app was never authenticated.
	AuthLossState func(current lifecycle.State) lifecycle.State
	// Authorized gates the success path; returning false skips restoring the running state,
	// e.g. when a Logout/Reset cleared credentials mid-probe. nil keeps the default of always
	// restoring. It re-reads state without a lock, so a teardown landing between this check
	// and the restore is possible; the cancelled/stale guard still covers a teardown that
	// overlaps the probe itself.
	Authorized func() bool
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

	// checkMu serializes Check so overlapping periodic and recheck probes cannot interleave state changes.
	checkMu sync.Mutex

	mu        sync.Mutex
	timer     *time.Timer
	failures  uint32
	cancelled bool
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
	c.checkMu.Lock()
	defer c.checkMu.Unlock()

	c.mu.Lock()
	// Skip the periodic probe when a recheck is already pending (it honors its backoff /
	// Retry-After delay) or when a Cancel (logout/reset) has torn the checker down and no
	// re-authorizing CheckNow has resumed it. Reading both under one lock — rather than
	// clearing cancelled unconditionally — closes the window where a Cancel racing the
	// periodic Check would be silently overwritten and the stale probe applied anyway.
	if c.timer != nil || c.cancelled {
		c.mu.Unlock()

		return nil
	}
	c.mu.Unlock()

	c.check()

	return nil
}

// CheckNow cancels any pending recheck delay and probes immediately. Use it when fresh
// credentials must be validated right away, e.g. as the check callback of Authorize. The
// failure counter is reset so a prior streak cannot trip MaxRechecks on the first probe.
func (c *ConnectivityChecker) CheckNow() error {
	c.checkMu.Lock()
	defer c.checkMu.Unlock()

	// Clear any pending recheck within checkMu and probe unconditionally: delegating to
	// Check would re-read pending() in a separate section, letting a concurrent probe that
	// just failed schedule a new timer in the gap and turn this into a silent no-op.
	c.mu.Lock()
	if c.timer != nil {
		c.timer.Stop()
		c.timer = nil
	}
	c.cancelled = false
	c.failures = 0
	c.mu.Unlock()

	c.check()

	return nil
}

// authLossState maps an unauthorized probe to an auth state, defaulting to AuthStateLost.
func (c *ConnectivityChecker) authLossState() lifecycle.State {
	if c.cfg.AuthLossState != nil {
		return c.cfg.AuthLossState(c.lc.AuthState())
	}

	return lifecycle.AuthStateLost
}

// authorized reports whether a successful probe may restore the running state, defaulting to true.
func (c *ConnectivityChecker) authorized() bool {
	return c.cfg.Authorized == nil || c.cfg.Authorized()
}

// check performs the probe and applies its outcome; the caller must hold checkMu
// and have cleared the cancelled flag while committing to this probe.
func (c *ConnectivityChecker) check() {
	err := c.probe()

	// A Cancel during the probe (logout or reset) makes its result stale, so it is discarded.
	if c.stale() {
		return
	}

	if err == nil {
		c.settle()

		if c.authorized() {
			c.apply(lifecycle.AuthStateAuthenticated, lifecycle.ConnStateConnected)
			c.repairAppHealth()
		}

		return
	}

	if errors.Is(err, httpclient.ErrUnauthorized) {
		c.settle()
		c.apply(c.authLossState(), lifecycle.ConnStateDisconnected)

		return
	}

	// A rate limit does not mean connectivity is lost, so the connectivity state is left untouched.
	if errors.Is(err, httpclient.ErrTooManyRequests) {
		c.settle()
		c.schedule(c.rateLimitDelay(err))

		return
	}

	c.warnLog.Do(err.Error(), func() { log.Warnf("[app] Check probe err: %v", err) })

	failures := c.fail()
	if failures > c.cfg.MaxRechecks {
		c.apply("", lifecycle.ConnStateDisconnected)
	}

	c.schedule(c.cfg.RecheckBackoff.Delay(failures))
}

// Cancel tears the checker down on logout or reset: it marks the checker cancelled so an
// in-flight probe's result is discarded (via stale) and subsequent periodic Checks are skipped,
// and it settles any pending recheck. A re-authorizing CheckNow clears the flag and resumes.
func (c *ConnectivityChecker) Cancel() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cancelled = true
	c.settleLocked()
}

// settle clears a pending recheck and the failure streak after a definitive probe result,
// WITHOUT marking the checker cancelled — only an external Cancel does that. Using it here (not
// Cancel) keeps the cancelled flag meaning "torn down", so a Cancel racing a Check is not masked
// by the checker's own post-probe cleanup.
func (c *ConnectivityChecker) settle() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.settleLocked()
}

func (c *ConnectivityChecker) settleLocked() {
	c.failures = 0
	c.warnLog.Reset()

	if c.timer != nil {
		c.timer.Stop()
		c.timer = nil
	}
}

// repairAppHealth restores the running and configured states after a successful probe.
// A transient failure during authorization may have left the app NOT_CONFIGURED, and the
// periodic check task does not run while the app is not RUNNING (task.WhenAppIsRunning),
// so a successful recheck is the only opportunity to repair that state. A successful
// probe proves valid credentials, which is what configured means for these adapters.
func (c *ConnectivityChecker) repairAppHealth() {
	if c.lc.AppHealth() == lifecycle.AppHealthRunning {
		return
	}

	c.lc.SetAppHealth(lifecycle.AppHealthRunning, nil)
	c.lc.SetConfigState(lifecycle.ConfigStateConfigured)
}

func (c *ConnectivityChecker) stale() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.cancelled
}

// rateLimitDelay honors a server-requested Retry-After delay, falling back to the configured one.
func (c *ConnectivityChecker) rateLimitDelay(err error) time.Duration {
	var rateLimitErr *httpclient.TooManyRequestsError
	if errors.As(err, &rateLimitErr) && rateLimitErr.RetryAfter > 0 {
		return rateLimitErr.RetryAfter
	}

	return c.cfg.RateLimitDelay
}

// apply sets the provided lifecycle states (empty auth leaves it untouched) and
// broadcasts node availability if any of them changed.
func (c *ConnectivityChecker) apply(auth, conn lifecycle.State) {
	changed := false

	// AuthStateLost triggers the auth-loss watcher, which reports the whole state bundle.
	// Set both states atomically and emit once so it observes a consistent bundle and the
	// trigger is not evicted from its buffer by a separate connection event.
	if auth == lifecycle.AuthStateLost {
		if c.lc.ConnectionState() != conn || c.lc.AuthState() != auth {
			c.lc.SetConnAndAuthState(conn, auth)

			changed = true
		}

		c.report(changed)

		return
	}

	if auth != "" && c.lc.AuthState() != auth {
		c.lc.SetAuthState(auth)

		changed = true
	}

	if c.lc.ConnectionState() != conn {
		c.lc.SetConnState(conn)

		changed = true
	}

	c.report(changed)
}

// report broadcasts node availability when a state change occurred.
func (c *ConnectivityChecker) report(changed bool) {
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
		// checkMu is taken before clearing the timer so a periodic Check cannot
		// slip into the gap and probe back-to-back with this recheck.
		c.checkMu.Lock()
		defer c.checkMu.Unlock()

		c.mu.Lock()
		if c.timer != timer {
			c.mu.Unlock()

			return
		}
		c.timer = nil
		// Cleared in the same critical section that commits this recheck, so a Cancel
		// racing the firing timer is either seen here or discarded later via stale().
		c.cancelled = false
		c.mu.Unlock()

		c.check()
	})
	c.timer = timer
}

func (c *ConnectivityChecker) fail() uint32 {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.failures++

	return c.failures
}
