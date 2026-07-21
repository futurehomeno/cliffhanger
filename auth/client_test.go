package auth_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/auth"
	"github.com/futurehomeno/cliffhanger/httpclient"
)

func TestProxyClient_RetryResendsBody(t *testing.T) {
	t.Parallel()

	var bodies []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		assert.NoError(t, err)

		bodies = append(bodies, string(b))
		if len(bodies) == 1 {
			w.WriteHeader(http.StatusBadGateway)

			return
		}

		_, err = w.Write([]byte(`{"access_token":"token","expires_in":3600}`))
		assert.NoError(t, err)
	}))
	defer srv.Close()

	client := auth.NewProxyClient(&auth.ProxyClientConfig{URL: srv.URL, Retry: 1, RetryDelay: time.Millisecond})

	response, err := client.ExchangeRefreshToken("refresh")
	assert.NoError(t, err)
	assert.Equal(t, "token", response.AccessToken)
	assert.Len(t, bodies, 2)
	assert.NotEmpty(t, bodies[0])
	assert.Equal(t, bodies[0], bodies[1], "retry must resend the full request body")
}

func TestProxyClient_EndpointResolvesURLAndTokenPerExchange(t *testing.T) {
	t.Parallel()

	var gotAuth, gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path

		_, err := w.Write([]byte(`{"access_token":"token","expires_in":3600}`))
		assert.NoError(t, err)
	}))
	defer srv.Close()

	calls := 0
	client := auth.NewProxyClient(&auth.ProxyClientConfig{
		// URL/Token are intentionally bogus: Endpoint must take precedence.
		URL:   "http://unused.invalid",
		Token: "unused",
		Endpoint: func() (string, string, error) {
			calls++

			return srv.URL, "hub-token", nil
		},
	})

	response, err := client.ExchangeRefreshToken("refresh")
	assert.NoError(t, err)
	assert.Equal(t, "token", response.AccessToken)
	assert.Equal(t, 1, calls, "Endpoint is resolved once per exchange")
	assert.Equal(t, "Bearer hub-token", gotAuth)
	assert.Equal(t, "/api/control/edge/proxy/refresh", gotPath)
}

func TestProxyClient_EndpointErrorFailsExchange(t *testing.T) {
	t.Parallel()

	client := auth.NewProxyClient(&auth.ProxyClientConfig{
		Endpoint: func() (string, string, error) { return "", "", assert.AnError },
	})

	_, err := client.ExchangeRefreshToken("refresh")
	assert.ErrorIs(t, err, assert.AnError)
}

func TestProxyClient_RateLimitedExchangeIsNotRetried(t *testing.T) {
	t.Parallel()

	requests := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++

		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	client := auth.NewProxyClient(&auth.ProxyClientConfig{URL: srv.URL, Retry: 2, RetryDelay: time.Millisecond})

	_, err := client.ExchangeRefreshToken("refresh")
	assert.ErrorIs(t, err, httpclient.ErrTooManyRequests)
	assert.Equal(t, 1, requests, "a rate-limited exchange must not be retried locally")
}
