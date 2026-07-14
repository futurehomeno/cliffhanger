package httpclient

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

var (
	// ErrUnauthorized is returned when the API responds with 401 or 403.
	ErrUnauthorized = errors.New("unauthorized")
	// ErrNotFound is returned when the API responds with 404.
	ErrNotFound = errors.New("not found")
	// ErrTooManyRequests is returned when the API responds with 429.
	ErrTooManyRequests = errors.New("too many requests")
)

// TooManyRequestsError carries the server-requested Retry-After delay.
// It matches ErrTooManyRequests with errors.Is.
type TooManyRequestsError struct {
	RetryAfter time.Duration
}

func (e *TooManyRequestsError) Error() string {
	return ErrTooManyRequests.Error()
}

func (e *TooManyRequestsError) Is(target error) bool {
	return target == ErrTooManyRequests
}

// ErrorFromStatus maps an HTTP status code to a shared sentinel error.
// It returns nil for success codes and a generic error for other failures.
func ErrorFromStatus(statusCode int) error {
	switch {
	case statusCode < http.StatusMultipleChoices:
		return nil
	case statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden:
		return ErrUnauthorized
	case statusCode == http.StatusNotFound:
		return ErrNotFound
	case statusCode == http.StatusTooManyRequests:
		return ErrTooManyRequests
	default:
		return fmt.Errorf("unexpected response status: %d", statusCode)
	}
}

// ErrorFromResponse maps the response status like ErrorFromStatus, carrying
// the Retry-After delay in a TooManyRequestsError when rate limited.
func ErrorFromResponse(resp *http.Response) error {
	if resp.StatusCode == http.StatusTooManyRequests {
		return &TooManyRequestsError{RetryAfter: RetryAfter(resp)}
	}

	return ErrorFromStatus(resp.StatusCode)
}

// RetryAfter returns the delay the server asks for via the Retry-After header (seconds), or 0.
func RetryAfter(resp *http.Response) time.Duration {
	secs, err := strconv.Atoi(resp.Header.Get("Retry-After"))
	if err != nil || secs <= 0 {
		return 0
	}

	return time.Duration(secs) * time.Second
}
