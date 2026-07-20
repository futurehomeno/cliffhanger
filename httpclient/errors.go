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

// errorFromStatus maps an HTTP status code to a shared sentinel error.
// It returns nil for success codes and a generic error for other failures.
func errorFromStatus(statusCode int) error {
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

// ErrorFromResponse maps the response status to a shared sentinel error, carrying
// the Retry-After delay in a TooManyRequestsError when rate limited.
func ErrorFromResponse(resp *http.Response) error {
	if resp.StatusCode == http.StatusTooManyRequests {
		return &TooManyRequestsError{RetryAfter: retryAfter(resp)}
	}

	return errorFromStatus(resp.StatusCode)
}

// retryAfter returns the delay the server asks for via the Retry-After header
// (delta-seconds or HTTP-date form), or 0.
// maxRetryAfter caps a server-supplied Retry-After so a misconfigured or hostile server cannot
// park the client (e.g. block the connectivity recheck) for an unbounded time.
const maxRetryAfter = time.Hour

func retryAfter(resp *http.Response) time.Duration {
	header := resp.Header.Get("Retry-After")

	secs, err := strconv.Atoi(header)
	if err == nil {
		if secs <= 0 {
			return 0
		}

		// A large value can overflow the multiplication to a negative Duration; treat any
		// non-positive or over-cap result as the cap.
		if d := time.Duration(secs) * time.Second; d > 0 && d < maxRetryAfter {
			return d
		}

		return maxRetryAfter
	}

	// A numeric header too large for int (Atoi returns ErrRange with the max magnitude value)
	// is an over-cap delay, not a date — clamp positive overflow to the cap.
	if errors.Is(err, strconv.ErrRange) && secs > 0 {
		return maxRetryAfter
	}

	if date, err := http.ParseTime(header); err == nil {
		if delay := time.Until(date); delay > 0 {
			return min(delay, maxRetryAfter)
		}
	}

	return 0
}
