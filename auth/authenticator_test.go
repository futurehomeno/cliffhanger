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
	creds auth.Credentials
}

func (s *fakeStore) Credentials() auth.Credentials { return s.creds }

func (s *fakeStore) SetCredentials(c auth.Credentials) error {
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

		exchanger := &fakeExchanger{err: errors.New("proxy down")}
		a := auth.NewAuthenticator(&fakeStore{creds: expiredCreds()}, exchanger, auth.AuthenticatorConfig{
			Backoff: backoff.NewStateful(time.Hour, time.Hour, time.Hour, 1, 1),
		})

		_, err := a.AccessToken()
		assert.Error(t, err)
		_, err = a.AccessToken()
		assert.Error(t, err)
		assert.Equal(t, 1, exchanger.calls, "second attempt should be suspended by backoff")
	})
}
