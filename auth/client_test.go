package auth_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
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

func TestProxyClient_EmptyAccessTokenIsARejection(t *testing.T) {
	t.Parallel()

	requests := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++

		_, err := w.Write([]byte(`{"error":"invalid_grant"}`))
		assert.NoError(t, err)
	}))
	defer srv.Close()

	client := auth.NewProxyClient(&auth.ProxyClientConfig{URL: srv.URL, Retry: 2, RetryDelay: time.Millisecond})

	_, err := client.ExchangeRefreshToken("refresh")
	assert.ErrorIs(t, err, httpclient.ErrUnauthorized)
	assert.Equal(t, 1, requests, "an empty access token must not be retried locally")
}

func TestProxyClient_ExchangeRequests(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		exchange func(auth.ProxyClient) (*auth.OAuth2TokenResponse, error)
		path     string
		body     string
	}{
		{
			name:     "authorization code",
			exchange: func(c auth.ProxyClient) (*auth.OAuth2TokenResponse, error) { return c.ExchangeAuthorizationCode("code") },
			path:     "/api/control/edge/proxy/auth-code",
			body:     `{"code":"code","partnerCode":"partner"}`,
		},
		{
			name:     "refresh token",
			exchange: func(c auth.ProxyClient) (*auth.OAuth2TokenResponse, error) { return c.ExchangeRefreshToken("refresh") },
			path:     "/api/control/edge/proxy/refresh",
			body:     `{"refreshToken":"refresh","partnerCode":"partner"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				b, err := io.ReadAll(r.Body)
				assert.NoError(t, err)

				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, tt.path, r.URL.Path)
				assert.JSONEq(t, tt.body, string(b))
				assert.Equal(t, []string{"Bearer token"}, r.Header.Values("Authorization"))
				assert.Equal(t, []string{"application/json"}, r.Header.Values("Content-Type"))
				assert.Equal(t, "value", r.Header.Get("X-Custom"))

				_, err = w.Write([]byte(`{"access_token":"access","refresh_token":"rotated","expires_in":60}`))
				assert.NoError(t, err)
			}))
			defer srv.Close()

			client := auth.NewProxyClient(&auth.ProxyClientConfig{
				URL:         srv.URL,
				PartnerCode: "partner",
				Token:       "token",
				Headers:     map[string]string{"authorization": "Bearer other", "Content-Type": "text/plain", "X-Custom": "value"},
			})

			response, err := tt.exchange(client)
			assert.NoError(t, err)
			assert.Equal(t, "access", response.AccessToken)
			assert.Equal(t, "rotated", response.RefreshToken)
		})
	}
}

func TestProxyClient_RejectedExchangeIsNotRetried(t *testing.T) {
	t.Parallel()

	for _, status := range []int{http.StatusUnauthorized, http.StatusForbidden} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			t.Parallel()

			var requests atomic.Int32

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				requests.Add(1)
				w.WriteHeader(status)
			}))
			defer srv.Close()

			client := auth.NewProxyClient(&auth.ProxyClientConfig{URL: srv.URL, Retry: 2, RetryDelay: time.Millisecond})

			_, err := client.ExchangeRefreshToken("refresh")
			assert.ErrorIs(t, err, httpclient.ErrUnauthorized)
			assert.Equal(t, int32(1), requests.Load(), "a rejection is definitive and must not be retried")
		})
	}
}

func TestProxyClient_RetriesExhausted(t *testing.T) {
	t.Parallel()

	var requests atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	client := auth.NewProxyClient(&auth.ProxyClientConfig{URL: srv.URL, Retry: 2, RetryDelay: time.Millisecond})

	response, err := client.ExchangeRefreshToken("refresh")
	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Equal(t, int32(3), requests.Load())
}
