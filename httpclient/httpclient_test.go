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

func TestRequestBuilder(t *testing.T) {
	t.Parallel()

	req, err := httpclient.NewRequest(http.MethodPost, "https://example.com/api").
		WithJSONBody(map[string]string{"key": "value"}).
		WithHeader("Authorization", "Bearer token").
		Build(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, http.MethodPost, req.Method)
	assert.Equal(t, "https://example.com/api", req.URL.String())
	assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
	assert.Equal(t, "Bearer token", req.Header.Get("Authorization"))

	body, err := io.ReadAll(req.Body)
	assert.NoError(t, err)
	assert.JSONEq(t, `{"key":"value"}`, string(body))

	req, err = httpclient.NewRequest(http.MethodGet, "https://example.com").Build(context.Background())
	assert.NoError(t, err)
	assert.Nil(t, req.Body)
}

func TestErrorFromStatus(t *testing.T) {
	t.Parallel()

	assert.NoError(t, httpclient.ErrorFromStatus(http.StatusOK))
	assert.NoError(t, httpclient.ErrorFromStatus(http.StatusNoContent))
	assert.ErrorIs(t, httpclient.ErrorFromStatus(http.StatusUnauthorized), httpclient.ErrUnauthorized)
	assert.ErrorIs(t, httpclient.ErrorFromStatus(http.StatusForbidden), httpclient.ErrUnauthorized)
	assert.ErrorIs(t, httpclient.ErrorFromStatus(http.StatusNotFound), httpclient.ErrNotFound)
	assert.ErrorIs(t, httpclient.ErrorFromStatus(http.StatusTooManyRequests), httpclient.ErrTooManyRequests)
	assert.Error(t, httpclient.ErrorFromStatus(http.StatusBadGateway))
}

func TestErrorFromResponse(t *testing.T) {
	t.Parallel()

	resp := &http.Response{StatusCode: http.StatusTooManyRequests, Header: http.Header{}}
	resp.Header.Set("Retry-After", "30")

	err := httpclient.ErrorFromResponse(resp)
	assert.ErrorIs(t, err, httpclient.ErrTooManyRequests)

	var rateLimitErr *httpclient.TooManyRequestsError

	assert.ErrorAs(t, err, &rateLimitErr)
	assert.Equal(t, 30*time.Second, rateLimitErr.RetryAfter)

	assert.NoError(t, httpclient.ErrorFromResponse(&http.Response{StatusCode: http.StatusOK}))
	assert.ErrorIs(t, httpclient.ErrorFromResponse(&http.Response{StatusCode: http.StatusForbidden}), httpclient.ErrUnauthorized)
}

func TestRetryAfter(t *testing.T) {
	t.Parallel()

	resp := &http.Response{Header: http.Header{}}
	assert.Equal(t, time.Duration(0), httpclient.RetryAfter(resp))

	resp.Header.Set("Retry-After", "30")
	assert.Equal(t, 30*time.Second, httpclient.RetryAfter(resp))

	resp.Header.Set("Retry-After", "invalid")
	assert.Equal(t, time.Duration(0), httpclient.RetryAfter(resp))

	resp.Header.Set("Retry-After", time.Now().Add(2*time.Minute).UTC().Format(http.TimeFormat))
	delay := httpclient.RetryAfter(resp)
	assert.Greater(t, delay, time.Minute)
	assert.LessOrEqual(t, delay, 2*time.Minute)

	resp.Header.Set("Retry-After", time.Now().Add(-time.Minute).UTC().Format(http.TimeFormat))
	assert.Equal(t, time.Duration(0), httpclient.RetryAfter(resp))
}
