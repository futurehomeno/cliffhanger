package auth_test

import (
	"errors"
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
}

func (s *fakeStore) Credentials() auth.Credentials { return s.creds }

func (s *fakeStore) SetCredentials(c auth.Credentials) error {
	if s.setErr != nil {
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
	response *auth.OAuth2TokenResponse
	err      error
	calls    int
}

func (e *fakeExchanger) ExchangeRefreshToken(string) (*auth.OAuth2TokenResponse, error) {
	e.calls++

	return e.response, e.err
}

type fakeBackoff struct {
	resets int
}

func (b *fakeBackoff) Next() time.Duration { return 0 }
func (b *fakeBackoff) Fail()               {}
func (b *fakeBackoff) Reset()              { b.resets++ }
func (b *fakeBackoff) Should() bool        { return false }

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

	t.Run("failed persistence does not commit refresh state", func(t *testing.T) {
		t.Parallel()

		b := &fakeBackoff{}
		store := &fakeStore{creds: expiredCreds(), setErr: errors.New("disk full")}
		exchanger := &fakeExchanger{response: &auth.OAuth2TokenResponse{AccessToken: "fresh", ExpiresIn: 3600}}
		a := auth.NewAuthenticator(store, exchanger, auth.AuthenticatorConfig{Backoff: b})

		_, err := a.AccessToken()
		assert.Error(t, err)
		assert.Zero(t, b.resets, "state should not be committed before credentials are persisted")

		store.setErr = nil

		token, err := a.AccessToken()
		assert.NoError(t, err)
		assert.Equal(t, "fresh", token)
		assert.Equal(t, 1, b.resets)
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
		exchanger.response = &auth.OAuth2TokenResponse{AccessToken: "fresh", ExpiresIn: 3600}
		store.setErr = errors.New("disk full")

		_, err = a.AccessToken()
		assert.Error(t, err, "persistence failure should surface")

		time.Sleep(100 * time.Millisecond)

		exchanger.err = httpclient.ErrUnauthorized

		token, err := a.AccessToken()
		assert.NoError(t, err, "the accepted exchange should have refuted the rejection streak")
		assert.Equal(t, "access", token)
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
}
