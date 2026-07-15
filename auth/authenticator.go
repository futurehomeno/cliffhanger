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
	// OnAuthLoss is invoked when the refresh token is rejected and credentials are cleared,
	// e.g. to mark the lifecycle auth state as lost and publish a logout event.
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

	creds := a.store.Credentials()
	if creds.Empty() {
		return "", ErrNotLoggedIn
	}

	if !creds.expired(a.cfg.RefreshLead) {
		return creds.AccessToken, nil
	}

	if !creds.RefreshExpiresAt.IsZero() && time.Now().After(creds.RefreshExpiresAt) {
		a.authLost("refresh token expired")

		return "", errors.New("refresh token expired, re-login required")
	}

	if a.cfg.Backoff.Should() {
		if !creds.expired(0) {
			return creds.AccessToken, nil
		}

		return "", errors.New("token refresh suspended by backoff")
	}

	response, err := a.exchange.ExchangeRefreshToken(creds.RefreshToken)
	if err != nil {
		a.cfg.Backoff.Fail()

		if errors.Is(err, httpclient.ErrUnauthorized) && !a.withinUnauthorizedGrace() {
			a.authLost(fmt.Sprintf("refresh token rejected: %v", err))

			return "", err
		}

		// A transient refresh failure does not invalidate a token that is still valid.
		if !creds.expired(0) {
			return creds.AccessToken, nil
		}

		return "", fmt.Errorf("exchange refresh token: %w", err)
	}

	a.cfg.Backoff.Reset()
	a.unauthorizedSince = time.Time{}

	newCreds := response.Credentials()
	if newCreds.RefreshToken == "" || newCreds.RefreshToken == creds.RefreshToken {
		newCreds.RefreshToken = creds.RefreshToken
		newCreds.RefreshExpiresAt = creds.RefreshExpiresAt
	}

	if err := a.store.SetCredentials(newCreds); err != nil {
		return "", fmt.Errorf("store credentials: %w", err)
	}

	return newCreds.AccessToken, nil
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

	if err := a.store.ClearCredentials(); err != nil {
		log.Errorf("[auth] Clear credentials err: %v", err)
	}

	if a.cfg.OnAuthLoss != nil {
		a.cfg.OnAuthLoss(reason)
	}
}
