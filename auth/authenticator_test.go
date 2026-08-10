package auth_test

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/auth"
	"github.com/futurehomeno/cliffhanger/backoff"
	"github.com/futurehomeno/cliffhanger/httpclient"
)

type fakeStore struct {
	creds  auth.Credentials
	setErr error
	// failOn limits setErr to the listed write ordinals, counted from 1; nil fails every write.
	failOn []int
	writes int
}

func (s *fakeStore) Credentials() auth.Credentials { return s.creds }

func (s *fakeStore) SetCredentials(c auth.Credentials) error {
	s.writes++

	if s.setErr != nil && (s.failOn == nil || slices.Contains(s.failOn, s.writes)) {
		return s.setErr
	}

	s.creds = c

	return nil
}

func (s *fakeStore) ClearCredentials() error {
	s.creds = auth.Credentials{}

	return nil
}

type fakeExchanger struct {
	response  *auth.OAuth2TokenResponse
	err       error
	calls     int
	lastToken string
}

func (e *fakeExchanger) ExchangeRefreshToken(refreshToken string) (*auth.OAuth2TokenResponse, error) {
	e.calls++
	e.lastToken = refreshToken

	return e.response, e.err
}

type fakeBackoff struct {
	resets int
	should bool
}

func (b *fakeBackoff) Next() time.Duration { return 0 }
func (b *fakeBackoff) Fail()               {}
func (b *fakeBackoff) Reset()              { b.resets++ }
func (b *fakeBackoff) Should() bool        { return b.should }

func validCreds() auth.Credentials {
	return auth.Credentials{AccessToken: "access", RefreshToken: "refresh", ExpiresAt: time.Now().Add(time.Hour)}
}

func expiredCreds() auth.Credentials {
	return auth.Credentials{AccessToken: "stale", RefreshToken: "refresh", ExpiresAt: time.Now().Add(-time.Minute)}
}

func TestAuthenticator_AccessToken(t *testing.T) {
	t.Parallel()

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()

		a := auth.NewAuthenticator(&fakeStore{}, &fakeExchanger{}, auth.AuthenticatorConfig{})

		_, err := a.AccessToken()
		assert.ErrorIs(t, err, auth.ErrNotLoggedIn)
	})

	t.Run("valid token returned without refresh", func(t *testing.T) {
		t.Parallel()

		exchanger := &fakeExchanger{}
		a := auth.NewAuthenticator(&fakeStore{creds: validCreds()}, exchanger, auth.AuthenticatorConfig{})

		token, err := a.AccessToken()
		assert.NoError(t, err)
		assert.Equal(t, "access", token)
		assert.Zero(t, exchanger.calls)
	})

	t.Run("expired token is refreshed and persisted", func(t *testing.T) {
		t.Parallel()

		store := &fakeStore{creds: expiredCreds()}
		exchanger := &fakeExchanger{response: &auth.OAuth2TokenResponse{AccessToken: "fresh", ExpiresIn: 3600}}
		a := auth.NewAuthenticator(store, exchanger, auth.AuthenticatorConfig{})

		token, err := a.AccessToken()
		assert.NoError(t, err)
		assert.Equal(t, "fresh", token)
		assert.Equal(t, "refresh", store.creds.RefreshToken, "missing refresh token in response should keep the old one")
		assert.True(t, store.creds.ExpiresAt.After(time.Now()))
	})

	t.Run("refresh without rotated token preserves refresh expiry", func(t *testing.T) {
		t.Parallel()

		creds := expiredCreds()
		creds.RefreshExpiresAt = time.Now().Add(time.Hour)

		store := &fakeStore{creds: creds}
		exchanger := &fakeExchanger{response: &auth.OAuth2TokenResponse{AccessToken: "fresh", ExpiresIn: 3600}}
		a := auth.NewAuthenticator(store, exchanger, auth.AuthenticatorConfig{})

		_, err := a.AccessToken()
		assert.NoError(t, err)
		assert.Equal(t, creds.RefreshExpiresAt, store.creds.RefreshExpiresAt, "carried over refresh token should keep its expiry")
	})

	t.Run("refresh returning same token preserves refresh expiry", func(t *testing.T) {
		t.Parallel()

		creds := expiredCreds()
		creds.RefreshExpiresAt = time.Now().Add(time.Hour)

		store := &fakeStore{creds: creds}
		exchanger := &fakeExchanger{response: &auth.OAuth2TokenResponse{AccessToken: "fresh", RefreshToken: creds.RefreshToken, ExpiresIn: 3600}}
		a := auth.NewAuthenticator(store, exchanger, auth.AuthenticatorConfig{})

		_, err := a.AccessToken()
		assert.NoError(t, err)
		assert.Equal(t, creds.RefreshExpiresAt, store.creds.RefreshExpiresAt, "explicitly returned same token is a carry-over and should keep its expiry")
	})

	t.Run("refresh with rotated token clears refresh expiry", func(t *testing.T) {
		t.Parallel()

		creds := expiredCreds()
		creds.RefreshExpiresAt = time.Now().Add(time.Hour)

		store := &fakeStore{creds: creds}
		exchanger := &fakeExchanger{response: &auth.OAuth2TokenResponse{AccessToken: "fresh", RefreshToken: "rotated", ExpiresIn: 3600}}
		a := auth.NewAuthenticator(store, exchanger, auth.AuthenticatorConfig{})

		_, err := a.AccessToken()
		assert.NoError(t, err)
		assert.Equal(t, "rotated", store.creds.RefreshToken)
		assert.True(t, store.creds.RefreshExpiresAt.IsZero(), "rotated refresh token must not inherit the old expiry")
	})

	t.Run("failed persistence keeps the rotated credentials in memory", func(t *testing.T) {
		t.Parallel()

		store := &fakeStore{creds: expiredCreds(), setErr: errors.New("disk full")}
		// Expiring at once keeps every call on the refresh path, so the token each exchange
		// is handed is what the assertions below observe.
		exchanger := &fakeExchanger{response: &auth.OAuth2TokenResponse{AccessToken: "fresh", RefreshToken: "rotated", ExpiresIn: 0}}
		a := auth.NewAuthenticator(store, exchanger, auth.AuthenticatorConfig{Backoff: &fakeBackoff{}})

		token, err := a.AccessToken()
		assert.NoError(t, err, "a persistence failure must not discard a successful exchange")
		assert.Equal(t, "fresh", token)
		assert.Equal(t, "refresh", exchanger.lastToken)
		assert.Equal(t, "refresh", store.creds.RefreshToken, "the store still holds the stale token")

		_, err = a.AccessToken()
		assert.NoError(t, err)
		assert.Equal(t, "rotated", exchanger.lastToken, "a rotating provider has invalidated the token still on disk")

		// A second failed rotation must not make the memory copy look stale against the store.
		_, err = a.AccessToken()
		assert.NoError(t, err)
		assert.Equal(t, "rotated", exchanger.lastToken)

		store.setErr = nil

		_, err = a.AccessToken()
		assert.NoError(t, err)
		assert.Equal(t, "rotated", store.creds.RefreshToken, "persistence should be retried on the refresh path")
	})

	t.Run("a persisted rotation is not shadowed by an older memory copy", func(t *testing.T) {
		t.Parallel()

		// A provider that keeps the refresh token leaves the store forever matching unsavedFrom,
		// so a memory copy outliving a successful save would win every later read.
		store := &fakeStore{creds: expiredCreds(), setErr: errors.New("disk full"), failOn: []int{1, 2}}
		exchanger := &fakeExchanger{response: &auth.OAuth2TokenResponse{AccessToken: "first", ExpiresIn: 0}}
		a := auth.NewAuthenticator(store, exchanger, auth.AuthenticatorConfig{Backoff: &fakeBackoff{}})

		token, err := a.AccessToken()
		assert.NoError(t, err)
		assert.Equal(t, "first", token, "the first exchange is kept in memory, the store refused it")

		exchanger.response = &auth.OAuth2TokenResponse{AccessToken: "second", ExpiresIn: 3600}

		token, err = a.AccessToken()
		assert.NoError(t, err)
		assert.Equal(t, "second", token)
		assert.Equal(t, "second", store.creds.AccessToken, "the retried write refused, the exchange persisted")

		exchanger.err = errors.New("network down")

		token, err = a.AccessToken()
		assert.NoError(t, err, "the persisted token must be read back instead of the stale memory copy")
		assert.Equal(t, "second", token)
		assert.Equal(t, "second", store.creds.AccessToken, "the stale memory copy must not be written over it")
	})

	t.Run("a rotation failing after a successful retry keeps its anchor", func(t *testing.T) {
		t.Parallel()

		// Writes in order: the first rotation, the retry that moves the store on to it, then the
		// rotation that follows in the same call. Only the middle one is allowed to land.
		store := &fakeStore{creds: expiredCreds(), setErr: errors.New("disk full"), failOn: []int{1, 3}}
		exchanger := &fakeExchanger{response: &auth.OAuth2TokenResponse{AccessToken: "a1", RefreshToken: "r1", ExpiresIn: 0}}
		a := auth.NewAuthenticator(store, exchanger, auth.AuthenticatorConfig{Backoff: &fakeBackoff{}})

		_, err := a.AccessToken()
		assert.NoError(t, err)

		exchanger.response = &auth.OAuth2TokenResponse{AccessToken: "a2", RefreshToken: "r2", ExpiresIn: 0}

		_, err = a.AccessToken()
		assert.NoError(t, err)
		assert.Equal(t, "r1", store.creds.RefreshToken, "the retry moved the store on to the first rotation")

		_, err = a.AccessToken()
		assert.NoError(t, err)
		assert.Equal(t, "r2", exchanger.lastToken, "the unpersisted rotation must not be dropped for the token it replaced")
	})

	t.Run("backoff suspends the persistence retry too", func(t *testing.T) {
		t.Parallel()

		store := &fakeStore{creds: expiredCreds(), setErr: errors.New("disk full")}
		exchanger := &fakeExchanger{response: &auth.OAuth2TokenResponse{AccessToken: "fresh", ExpiresIn: 3600}}
		b := &fakeBackoff{}
		// A RefreshLead longer than the token's life keeps every call on the refresh path.
		a := auth.NewAuthenticator(store, exchanger, auth.AuthenticatorConfig{Backoff: b, RefreshLead: 2 * time.Hour})

		_, err := a.AccessToken()
		assert.NoError(t, err)
		assert.Equal(t, 1, store.writes)

		b.should = true

		for range 5 {
			_, _ = a.AccessToken()
		}

		assert.Equal(t, 1, store.writes, "a store that keeps refusing must not be written per request while backoff holds")
		assert.Equal(t, 1, exchanger.calls)
	})

	t.Run("unpersisted credentials are dropped once the store changes out-of-band", func(t *testing.T) {
		t.Parallel()

		store := &fakeStore{creds: expiredCreds(), setErr: errors.New("disk full")}
		exchanger := &fakeExchanger{response: &auth.OAuth2TokenResponse{AccessToken: "fresh", RefreshToken: "rotated", ExpiresIn: 3600}}
		a := auth.NewAuthenticator(store, exchanger, auth.AuthenticatorConfig{})

		_, err := a.AccessToken()
		assert.NoError(t, err)

		store.setErr = nil
		store.creds = auth.Credentials{AccessToken: "relogin", RefreshToken: "relogin", ExpiresAt: time.Now().Add(time.Hour)}

		token, err := a.AccessToken()
		assert.NoError(t, err)
		assert.Equal(t, "relogin", token, "a fresh login must not be shadowed by the memory copy")
	})

	t.Run("successful exchange clears grace state even when persistence fails", func(t *testing.T) {
		t.Parallel()

		creds := validCreds()
		creds.ExpiresAt = time.Now().Add(time.Minute)

		store := &fakeStore{creds: creds}
		exchanger := &fakeExchanger{err: httpclient.ErrUnauthorized}
		a := auth.NewAuthenticator(store, exchanger, auth.AuthenticatorConfig{
			RefreshLead:       5 * time.Minute,
			UnauthorizedGrace: 50 * time.Millisecond,
			Backoff:           backoff.NewTolerantFixed(10, 0),
		})

		_, err := a.AccessToken()
		assert.NoError(t, err, "first rejection should start the grace window")

		exchanger.err = nil
		// Expiring immediately keeps the next call on the refresh path despite the exchange
		// having succeeded, so the rejection streak is what the assertion below observes.
		exchanger.response = &auth.OAuth2TokenResponse{AccessToken: "fresh", ExpiresIn: 0}
		store.setErr = errors.New("disk full")

		_, err = a.AccessToken()
		assert.NoError(t, err, "a persistence failure must not discard a successful exchange")

		time.Sleep(100 * time.Millisecond)

		exchanger.err = httpclient.ErrUnauthorized

		_, err = a.AccessToken()
		assert.ErrorIs(t, err, auth.ErrRefreshDeferred, "the accepted exchange should have refuted the rejection streak")
		assert.False(t, store.creds.Empty(), "credentials should be kept within the fresh grace window")
	})

	t.Run("rejected refresh token clears credentials and reports auth loss", func(t *testing.T) {
		t.Parallel()

		store := &fakeStore{creds: expiredCreds()}
		lossReason := ""
		a := auth.NewAuthenticator(store, &fakeExchanger{err: httpclient.ErrUnauthorized}, auth.AuthenticatorConfig{
			OnAuthLoss: func(reason string) { lossReason = reason },
		})

		_, err := a.AccessToken()
		assert.ErrorIs(t, err, httpclient.ErrUnauthorized)
		assert.True(t, store.creds.Empty())
		assert.Contains(t, lossReason, "refresh token rejected")
	})

	t.Run("auth loss resets backoff state for the next login", func(t *testing.T) {
		t.Parallel()

		store := &fakeStore{creds: expiredCreds()}
		exchanger := &fakeExchanger{err: httpclient.ErrUnauthorized}
		a := auth.NewAuthenticator(store, exchanger, auth.AuthenticatorConfig{
			Backoff: backoff.NewStateful(time.Hour, time.Hour, time.Hour, 0, 0),
		})

		_, err := a.AccessToken()
		assert.Error(t, err)

		store.creds = expiredCreds()
		exchanger.err = nil
		exchanger.response = &auth.OAuth2TokenResponse{AccessToken: "fresh", ExpiresIn: 3600}

		token, err := a.AccessToken()
		assert.NoError(t, err, "re-login after auth loss should not be suppressed by stale backoff")
		assert.Equal(t, "fresh", token)
	})

	t.Run("auth loss resets grace state for the next login", func(t *testing.T) {
		t.Parallel()

		creds := validCreds()
		creds.ExpiresAt = time.Now().Add(time.Minute)

		store := &fakeStore{creds: creds}
		exchanger := &fakeExchanger{err: httpclient.ErrUnauthorized}
		a := auth.NewAuthenticator(store, exchanger, auth.AuthenticatorConfig{
			RefreshLead:       5 * time.Minute,
			UnauthorizedGrace: 50 * time.Millisecond,
			Backoff:           backoff.NewTolerantFixed(10, 0),
		})

		_, err := a.AccessToken()
		assert.NoError(t, err, "first rejection should start the grace window")

		time.Sleep(100 * time.Millisecond)

		doomed := expiredCreds()
		doomed.RefreshExpiresAt = time.Now().Add(-time.Minute)
		store.creds = doomed

		_, err = a.AccessToken()
		assert.Error(t, err, "locally expired refresh token should conclude auth loss")

		store.creds = creds
		token, err := a.AccessToken()
		assert.NoError(t, err, "rejection after re-login should be tolerated by a fresh grace window")
		assert.Equal(t, "access", token)
	})

	t.Run("transient refresh failure keeps still valid token", func(t *testing.T) {
		t.Parallel()

		creds := auth.Credentials{AccessToken: "access", RefreshToken: "refresh", ExpiresAt: time.Now().Add(time.Minute)}
		a := auth.NewAuthenticator(&fakeStore{creds: creds}, &fakeExchanger{err: errors.New("proxy down")}, auth.AuthenticatorConfig{
			RefreshLead: 5 * time.Minute,
		})

		token, err := a.AccessToken()
		assert.NoError(t, err, "proactive refresh failure should not invalidate a still valid token")
		assert.Equal(t, "access", token)
	})

	t.Run("locally expired refresh token skips the exchange", func(t *testing.T) {
		t.Parallel()

		creds := expiredCreds()
		creds.RefreshExpiresAt = time.Now().Add(-time.Minute)

		store := &fakeStore{creds: creds}
		exchanger := &fakeExchanger{}
		a := auth.NewAuthenticator(store, exchanger, auth.AuthenticatorConfig{})

		_, err := a.AccessToken()
		assert.Error(t, err)
		assert.Zero(t, exchanger.calls, "doomed exchange should be skipped")
		assert.True(t, store.creds.Empty())
	})

	t.Run("locally expired refresh token keeps a still valid access token", func(t *testing.T) {
		t.Parallel()

		creds := auth.Credentials{
			AccessToken:      "access",
			RefreshToken:     "refresh",
			ExpiresAt:        time.Now().Add(time.Minute),
			RefreshExpiresAt: time.Now().Add(-time.Minute),
		}

		store := &fakeStore{creds: creds}
		exchanger := &fakeExchanger{}
		lossReason := ""
		a := auth.NewAuthenticator(store, exchanger, auth.AuthenticatorConfig{
			RefreshLead: 5 * time.Minute,
			OnAuthLoss:  func(reason string) { lossReason = reason },
		})

		token, err := a.AccessToken()
		assert.NoError(t, err, "auth loss must not be concluded a whole RefreshLead before the access token expires")
		assert.Equal(t, "access", token)
		assert.Zero(t, exchanger.calls, "doomed exchange should be skipped")
		assert.Empty(t, lossReason)
		assert.False(t, store.creds.Empty())
	})

	t.Run("unauthorized grace tolerates transient rejections", func(t *testing.T) {
		t.Parallel()

		creds := validCreds()
		creds.ExpiresAt = time.Now().Add(time.Minute)

		store := &fakeStore{creds: creds}
		a := auth.NewAuthenticator(store, &fakeExchanger{err: httpclient.ErrUnauthorized}, auth.AuthenticatorConfig{
			RefreshLead:       5 * time.Minute,
			UnauthorizedGrace: time.Hour,
			Backoff:           backoff.NewTolerantFixed(10, 0),
		})

		token, err := a.AccessToken()
		assert.NoError(t, err, "rejection within grace should keep the still valid token")
		assert.Equal(t, "access", token)
		assert.False(t, store.creds.Empty(), "credentials should be kept within grace")
	})

	t.Run("rejection within grace on an expired access token does not surface ErrUnauthorized", func(t *testing.T) {
		t.Parallel()

		store := &fakeStore{creds: expiredCreds()}
		a := auth.NewAuthenticator(store, &fakeExchanger{err: httpclient.ErrUnauthorized}, auth.AuthenticatorConfig{
			UnauthorizedGrace: time.Hour,
			Backoff:           backoff.NewTolerantFixed(10, 0),
		})

		_, err := a.AccessToken()
		assert.Error(t, err, "no usable token: the access token is expired and the refresh was rejected")
		assert.False(t, errors.Is(err, httpclient.ErrUnauthorized),
			"a caller keyed on errors.Is(err, httpclient.ErrUnauthorized) (e.g. a ConnectivityChecker "+
				"probe) must not conclude authorization loss while still inside the grace window")
		assert.ErrorIs(t, err, auth.ErrRefreshDeferred)
		assert.False(t, store.creds.Empty(), "credentials must be kept while grace is still active")
	})

	t.Run("backoff suspends refresh attempts", func(t *testing.T) {
		t.Parallel()

		store := &fakeStore{creds: expiredCreds()}
		exchanger := &fakeExchanger{err: errors.New("proxy down")}
		a := auth.NewAuthenticator(store, exchanger, auth.AuthenticatorConfig{
			UnauthorizedGrace: time.Nanosecond,
			Backoff:           backoff.NewStateful(time.Hour, time.Hour, time.Hour, 1, 1),
		})

		_, err := a.AccessToken()
		assert.Error(t, err)
		_, err = a.AccessToken()
		assert.Error(t, err)
		assert.Equal(t, 1, exchanger.calls, "second attempt should be suspended by backoff")
		assert.False(t, store.creds.Empty(), "a transient (non-401) backoff must not conclude auth loss even past grace")
	})

	t.Run("backoff concludes auth loss once grace elapses", func(t *testing.T) {
		t.Parallel()

		creds := validCreds()
		creds.ExpiresAt = time.Now().Add(-time.Minute)

		store := &fakeStore{creds: creds}
		lossReason := ""
		a := auth.NewAuthenticator(store, &fakeExchanger{err: httpclient.ErrUnauthorized}, auth.AuthenticatorConfig{
			UnauthorizedGrace: 50 * time.Millisecond,
			Backoff:           backoff.NewTolerantFixed(0, time.Hour),
			OnAuthLoss:        func(reason string) { lossReason = reason },
		})

		_, err := a.AccessToken()
		assert.Error(t, err, "first 401 starts the grace window and the backoff streak")

		time.Sleep(100 * time.Millisecond)

		_, err = a.AccessToken()
		assert.Error(t, err, "expired credentials must not linger once grace elapses under backoff")
		assert.True(t, store.creds.Empty(), "auth loss should clear credentials rather than stay suspended by backoff")
		assert.Contains(t, lossReason, "grace elapsed during backoff")
	})

	t.Run("backoff auth loss ignores credentials replaced after the rejection", func(t *testing.T) {
		t.Parallel()

		creds := validCreds()
		creds.ExpiresAt = time.Now().Add(-time.Minute)

		store := &fakeStore{creds: creds}
		a := auth.NewAuthenticator(store, &fakeExchanger{err: httpclient.ErrUnauthorized}, auth.AuthenticatorConfig{
			UnauthorizedGrace: 50 * time.Millisecond,
			Backoff:           backoff.NewTolerantFixed(0, time.Hour),
		})

		_, err := a.AccessToken()
		assert.Error(t, err, "first 401 starts the grace window for the rejected refresh token")

		// Re-authorization out-of-band: fresh credentials with a new refresh token, set
		// directly on the store (e.g. a manual set-tokens command), bypassing the exchange.
		fresh := validCreds()
		fresh.RefreshToken = "refresh-2"
		fresh.ExpiresAt = time.Now().Add(-time.Minute)
		store.creds = fresh

		time.Sleep(100 * time.Millisecond)

		_, err = a.AccessToken()
		assert.Error(t, err, "the replaced credentials still need a refresh that backoff suspends")
		assert.False(t, store.creds.Empty(),
			"a stale rejection streak from the old token must not clear the replaced credentials")
	})

	t.Run("exchange grace resets for credentials replaced after the rejection", func(t *testing.T) {
		t.Parallel()

		creds := validCreds()
		creds.ExpiresAt = time.Now().Add(-time.Minute)

		store := &fakeStore{creds: creds}
		a := auth.NewAuthenticator(store, &fakeExchanger{err: httpclient.ErrUnauthorized}, auth.AuthenticatorConfig{
			UnauthorizedGrace: 50 * time.Millisecond,
			Backoff:           &fakeBackoff{},
		})

		_, err := a.AccessToken()
		assert.Error(t, err, "first 401 on the original token starts the grace window")

		time.Sleep(100 * time.Millisecond)

		// Out-of-band replacement with a new refresh token that still needs a refresh; the
		// exchange path (backoff not suppressing) must give it its own grace, not the elapsed one.
		fresh := validCreds()
		fresh.RefreshToken = "refresh-2"
		fresh.ExpiresAt = time.Now().Add(-time.Minute)
		store.creds = fresh

		_, err = a.AccessToken()
		assert.Error(t, err, "the replaced token still fails to exchange")
		assert.False(t, store.creds.Empty(),
			"the replaced token gets its own grace window rather than inheriting the old token's elapsed one")
	})

	t.Run("backoff returns a still valid token without refreshing", func(t *testing.T) {
		t.Parallel()

		b := backoff.NewStateful(time.Hour, time.Hour, time.Hour, 0, 0)
		b.Fail() // prime the streak so Should() suppresses the refresh exchange

		creds := validCreds()
		creds.ExpiresAt = time.Now().Add(time.Minute) // inside RefreshLead but not yet expired

		exchanger := &fakeExchanger{}
		a := auth.NewAuthenticator(&fakeStore{creds: creds}, exchanger, auth.AuthenticatorConfig{
			RefreshLead: 5 * time.Minute,
			Backoff:     b,
		})

		token, err := a.AccessToken()
		assert.NoError(t, err, "a still valid access token is returned while backoff suppresses the refresh")
		assert.Equal(t, "access", token)
		assert.Zero(t, exchanger.calls, "backoff must suppress the refresh exchange")
	})
}
