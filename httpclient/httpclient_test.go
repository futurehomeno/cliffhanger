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

	errFor := func(code int) error {
		return httpclient.ErrorFromResponse(&http.Response{StatusCode: code, Header: http.Header{}})
	}

	assert.NoError(t, errFor(http.StatusOK))
	assert.NoError(t, errFor(http.StatusNoContent))
	assert.ErrorIs(t, errFor(http.StatusUnauthorized), httpclient.ErrUnauthorized)
	assert.ErrorIs(t, errFor(http.StatusForbidden), httpclient.ErrUnauthorized)
	assert.ErrorIs(t, errFor(http.StatusNotFound), httpclient.ErrNotFound)
	assert.ErrorIs(t, errFor(http.StatusTooManyRequests), httpclient.ErrTooManyRequests)
	assert.Error(t, errFor(http.StatusBadGateway))
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

	// A misconfigured or hostile server must not park the client indefinitely: over-cap and
	// overflow-large values clamp to the 1h ceiling rather than a huge (or negative) delay.
	assert.Equal(t, time.Hour, rateLimited("100000"), "an over-cap Retry-After clamps to the ceiling")
	assert.Equal(t, time.Hour, rateLimited("999999999999"), "an overflow-large Retry-After clamps, not negative")
	assert.Equal(t, time.Hour, rateLimited("99999999999999999999999"), "a value too large for int64 clamps to the ceiling, not 0")
	assert.Equal(t, time.Hour, rateLimited(time.Now().Add(48*time.Hour).UTC().Format(http.TimeFormat)),
		"a far-future Retry-After date clamps to the ceiling")
}
