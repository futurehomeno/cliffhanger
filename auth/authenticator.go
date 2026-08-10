package auth

import (
	"errors"
	"fmt"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/cliffhanger/backoff"
	"github.com/futurehomeno/cliffhanger/httpclient"
)

// ErrNotLoggedIn is returned when no credentials are available.
var ErrNotLoggedIn = errors.New("not logged in")

// ErrRefreshDeferred is returned by AccessToken for a rejected refresh still inside
// UnauthorizedGrace. It deliberately does not wrap httpclient.ErrUnauthorized: a caller
// keyed on errors.Is(err, httpclient.ErrUnauthorized) (e.g. a ConnectivityChecker probe
// wrapping this transport) would otherwise conclude authorization loss on the very first
// rejection, defeating the grace window this error exists to honor.
var ErrRefreshDeferred = errors.New("token refresh deferred within unauthorized grace")

// ErrRefreshSuspended is returned by AccessToken while backoff suppresses a refresh
// attempt. It is transient and fires on every call for the whole backoff window, so a
// caller that cannot tell it apart logs the expected wait as a failure on every request.
var ErrRefreshSuspended = errors.New("token refresh suspended by backoff")

// ErrReloginRequired is returned by AccessToken once the refresh token can no longer
// produce an access token, whether it expired or was rejected. It is terminal: only a new
// login clears it, and the wrapping message names the cause.
var ErrReloginRequired = errors.New("re-login required")

// Credentials is a snapshot of persisted OAuth credentials.
// RefreshExpiresAt is optional; when set, a refresh past it is skipped as doomed.
type Credentials struct {
	AccessToken      string
	RefreshToken     string
	ExpiresAt        time.Time
	RefreshExpiresAt time.Time
}

func (c Credentials) Empty() bool {
	return c.AccessToken == "" && c.RefreshToken == ""
}

func (c Credentials) expired(lead time.Duration) bool {
	return time.Now().After(c.ExpiresAt.Add(-lead))
}

// Credentials converts the token response into credentials with an absolute expiry time.
func (r *OAuth2TokenResponse) Credentials() Credentials {
	return Credentials{
		AccessToken:  r.AccessToken,
		RefreshToken: r.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(r.ExpiresIn) * time.Second),
	}
}

// CredentialsStore persists OAuth credentials, typically in the application configuration.
type CredentialsStore interface {
	Credentials() Credentials
	SetCredentials(Credentials) error
	ClearCredentials() error
}

// TokenExchanger exchanges a refresh token for fresh credentials. It is satisfied by ProxyClient.
type TokenExchanger interface {
	ExchangeRefreshToken(refreshToken string) (*OAuth2TokenResponse, error)
}

type AuthenticatorConfig struct {
	// RefreshLead is how long before expiry the access token is refreshed proactively.
	RefreshLead time.Duration
	// Backoff spaces retries of failed refresh attempts.
	Backoff backoff.Stateful
	// OnAuthLoss is invoked when authorization is lost — the refresh token is rejected or
	// has locally expired — and credentials are cleared, e.g. to mark the lifecycle auth
	// state as lost and publish a logout event.
	// It runs with the internal lock held, so it must not call back into the Authenticator.
	OnAuthLoss func(reason string)
	// UnauthorizedGrace tolerates rejected refresh attempts for this long before concluding
	// authorization loss, for APIs known to return spurious rejections on valid tokens.
	// 0 concludes authorization loss on the first rejection.
	UnauthorizedGrace time.Duration
}

func (c *AuthenticatorConfig) setDefaults() {
	if c.RefreshLead == 0 {
		c.RefreshLead = 5 * time.Minute
	}

	if c.Backoff == nil {
		c.Backoff = backoff.NewTolerantFixed(2, 5*time.Minute)
	}
}

// Authenticator provides valid access tokens with single-flight proactive refresh and backoff.
type Authenticator struct {
	store    CredentialsStore
	exchange TokenExchanger
	cfg      AuthenticatorConfig

	mu                sync.Mutex
	unauthorizedSince time.Time
	unauthorizedToken string
	unsaved           Credentials
	unsavedFrom       string
}

func NewAuthenticator(store CredentialsStore, exchanger TokenExchanger, cfg AuthenticatorConfig) *Authenticator {
	cfg.setDefaults()

	return &Authenticator{store: store, exchange: exchanger, cfg: cfg}
}

// AccessToken returns a valid access token, refreshing it proactively RefreshLead before
// expiry. A rejected refresh token clears the credentials and reports authorization loss.
func (a *Authenticator) AccessToken() (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	creds := a.credentials()
	if creds.Empty() {
		return "", ErrNotLoggedIn
	}

	if !creds.expired(a.cfg.RefreshLead) {
		return creds.AccessToken, nil
	}

	if !creds.RefreshExpiresAt.IsZero() && time.Now().After(creds.RefreshExpiresAt) {
		// A doomed refresh does not invalidate an access token that is still valid, matching
		// the backoff and transient-failure branches below.
		if !creds.expired(0) {
			return creds.AccessToken, nil
		}

		a.authLost("refresh token expired")

		return "", fmt.Errorf("refresh token expired, %w", ErrReloginRequired)
	}

	if a.cfg.Backoff.Should() {
		if !creds.expired(0) {
			return creds.AccessToken, nil
		}

		// A rejection streak that outlived the grace period concludes auth loss even while
		// backoff suppresses new exchanges. unauthorizedSince is only set by a real 401/403,
		// so a transient (non-auth) backoff is left alone to keep retrying. The streak is tied
		// to the rejected refresh token, so credentials replaced out-of-band are not concluded
		// lost on a stale timestamp they never triggered.
		if !a.unauthorizedSince.IsZero() && a.unauthorizedToken == creds.RefreshToken && !a.withinUnauthorizedGrace() {
			a.authLost("refresh token rejected, grace elapsed during backoff")

			return "", fmt.Errorf("refresh token rejected, %w", ErrReloginRequired)
		}

		return "", ErrRefreshSuspended
	}

	a.persistUnsaved()

	response, err := a.exchange.ExchangeRefreshToken(creds.RefreshToken)
	if err != nil {
		a.cfg.Backoff.Fail()

		if errors.Is(err, httpclient.ErrUnauthorized) {
			// A refresh token different from the one that started the streak gets its own grace
			// window, so credentials replaced out-of-band are not concluded lost on the elapsed
			// grace of a token they never shared.
			if a.unauthorizedSince.IsZero() || a.unauthorizedToken != creds.RefreshToken {
				a.unauthorizedSince = time.Time{}
				a.unauthorizedToken = creds.RefreshToken
			}

			if !a.withinUnauthorizedGrace() {
				a.authLost(fmt.Sprintf("refresh token rejected: %v", err))

				// Both chains matter: ErrReloginRequired marks the loss terminal, while
				// httpclient.ErrUnauthorized is what a ConnectivityChecker probe wrapping this
				// transport keys on to map the outcome onto the auth-loss lifecycle state.
				return "", fmt.Errorf("refresh token rejected (%w), %w", err, ErrReloginRequired)
			}
		}

		// A transient refresh failure does not invalidate a token that is still valid.
		if !creds.expired(0) {
			return creds.AccessToken, nil
		}

		if errors.Is(err, httpclient.ErrUnauthorized) {
			// err.Error(), not %w err: wrapping it would let it still match
			// errors.Is(returnedErr, httpclient.ErrUnauthorized), defeating the deferral above.
			return "", fmt.Errorf("exchange refresh token deferred within grace (%s): %w", err.Error(), ErrRefreshDeferred)
		}

		return "", fmt.Errorf("exchange refresh token: %w", err)
	}

	// The server accepted the refresh token, refuting any rejection streak regardless
	// of whether persistence below succeeds.
	a.unauthorizedSince = time.Time{}
	a.unauthorizedToken = ""

	newCreds := response.Credentials()
	if newCreds.RefreshToken == "" || newCreds.RefreshToken == creds.RefreshToken {
		newCreds.RefreshToken = creds.RefreshToken
		newCreds.RefreshExpiresAt = creds.RefreshExpiresAt
	}

	if err := a.store.SetCredentials(newCreds); err != nil {
		// The exchange already succeeded, so the stored refresh token may have been invalidated
		// by a rotating provider: keeping the new credentials in memory avoids retrying with a
		// token the server has thrown away, and the access token stays usable meanwhile.
		log.Errorf("[auth] Store credentials err, keeping them in memory: %v", err)

		// unsavedFrom already holds what the store had when credentials() read it, which is
		// what the memory copy supersedes even across a second failed rotation.
		a.unsaved = newCreds
	} else {
		// A provider that keeps the refresh token leaves unsavedFrom matching the store, so an
		// earlier memory copy would shadow what was just persisted and later be written over it.
		a.unsaved = Credentials{}
	}

	a.cfg.Backoff.Reset()

	return newCreds.AccessToken, nil
}

// credentials returns the stored credentials, preferring a rotation that failed to persist for
// as long as the store still holds the token it replaced, so credentials changed out-of-band
// (a fresh login, a logout) still win.
func (a *Authenticator) credentials() Credentials {
	creds := a.store.Credentials()

	if !a.unsaved.Empty() && a.unsavedFrom == creds.RefreshToken {
		return a.unsaved
	}

	// Either nothing is held, or the store no longer has the token the memory copy replaced
	// and the credentials changed out-of-band. Either way the store wins and becomes the
	// token a later rotation would supersede.
	a.unsaved, a.unsavedFrom = Credentials{}, creds.RefreshToken

	return creds
}

// persistUnsaved retries a rotation the store refused earlier. It is called only on the refresh
// path: AccessToken runs per outbound request, so retrying there would mean a failing write per
// request, while the refresh path is already spaced by RefreshLead and the backoff.
func (a *Authenticator) persistUnsaved() {
	if a.unsaved.Empty() {
		return
	}

	if err := a.store.SetCredentials(a.unsaved); err != nil {
		return
	}

	// The store now holds what was persisted, so it is what a rotation later in this same call
	// supersedes. Clearing the anchor instead would make the next read judge that rotation stale
	// and fall back to the token the provider had just replaced.
	a.unsavedFrom = a.unsaved.RefreshToken
	a.unsaved = Credentials{}
}

func (a *Authenticator) withinUnauthorizedGrace() bool {
	if a.cfg.UnauthorizedGrace == 0 {
		return false
	}

	if a.unauthorizedSince.IsZero() {
		a.unauthorizedSince = time.Now()
	}

	return time.Since(a.unauthorizedSince) < a.cfg.UnauthorizedGrace
}

func (a *Authenticator) authLost(reason string) {
	log.Warnf("[auth] Authorization lost: %s", reason)

	a.cfg.Backoff.Reset()
	a.unauthorizedSince = time.Time{}
	a.unauthorizedToken = ""
	a.unsaved, a.unsavedFrom = Credentials{}, ""

	if err := a.store.ClearCredentials(); err != nil {
		log.Errorf("[auth] Clear credentials err: %v", err)
	}

	if a.cfg.OnAuthLoss != nil {
		a.cfg.OnAuthLoss(reason)
	}
}
