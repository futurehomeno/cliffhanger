package httpclient_test

import (
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/httpclient"
)

func TestNewJSONRequest(t *testing.T) {
	t.Parallel()

	req, err := httpclient.NewJSONRequest(context.Background(), http.MethodPost, "https://example.com/api",
		map[string]string{"key": "value"}, map[string]string{"Authorization": "Bearer token"})

	assert.NoError(t, err)
	assert.Equal(t, http.MethodPost, req.Method)
	assert.Equal(t, "https://example.com/api", req.URL.String())
	assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
	assert.Equal(t, "Bearer token", req.Header.Get("Authorization"))

	body, err := io.ReadAll(req.Body)
	assert.NoError(t, err)
	assert.JSONEq(t, `{"key":"value"}`, string(body))

	req, err = httpclient.NewJSONRequest(context.Background(), http.MethodGet, "https://example.com", nil, nil)
	assert.NoError(t, err)
	assert.Nil(t, req.Body)
	assert.Empty(t, req.Header.Get("Content-Type"))
}

func TestErrorFromResponse(t *testing.T) {
	t.Parallel()

	status := func(code int) *http.Response {
		return &http.Response{StatusCode: code, Header: http.Header{}}
	}

	assert.NoError(t, httpclient.ErrorFromResponse(status(http.StatusOK)))
	assert.NoError(t, httpclient.ErrorFromResponse(status(http.StatusNoContent)))
	assert.ErrorIs(t, httpclient.ErrorFromResponse(status(http.StatusUnauthorized)), httpclient.ErrUnauthorized)
	assert.ErrorIs(t, httpclient.ErrorFromResponse(status(http.StatusForbidden)), httpclient.ErrUnauthorized)
	assert.ErrorIs(t, httpclient.ErrorFromResponse(status(http.StatusNotFound)), httpclient.ErrNotFound)
	assert.ErrorIs(t, httpclient.ErrorFromResponse(status(http.StatusTooManyRequests)), httpclient.ErrTooManyRequests)
	assert.Error(t, httpclient.ErrorFromResponse(status(http.StatusBadGateway)))
}

func TestErrorFromResponse_RetryAfter(t *testing.T) {
	t.Parallel()

	rateLimited := func(retryAfter string) time.Duration {
		resp := &http.Response{StatusCode: http.StatusTooManyRequests, Header: http.Header{}}
		if retryAfter != "" {
			resp.Header.Set("Retry-After", retryAfter)
		}

		var rateLimitErr *httpclient.TooManyRequestsError

		err := httpclient.ErrorFromResponse(resp)
		assert.ErrorIs(t, err, httpclient.ErrTooManyRequests)
		assert.ErrorAs(t, err, &rateLimitErr)

		return rateLimitErr.RetryAfter
	}

	assert.Equal(t, time.Duration(0), rateLimited(""))
	assert.Equal(t, 30*time.Second, rateLimited("30"))
	assert.Equal(t, time.Duration(0), rateLimited("invalid"))
	assert.Equal(t, time.Duration(0), rateLimited("-5"))

	delay := rateLimited(time.Now().Add(2 * time.Minute).UTC().Format(http.TimeFormat))
	assert.Greater(t, delay, time.Minute)
	assert.LessOrEqual(t, delay, 2*time.Minute)

	assert.Equal(t, time.Duration(0), rateLimited(time.Now().Add(-time.Minute).UTC().Format(http.TimeFormat)))
}
