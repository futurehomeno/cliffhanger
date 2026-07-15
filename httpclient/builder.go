package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
)

// NewJSONRequest builds an HTTP request, encoding a non-nil body as JSON with the
// Content-Type header set accordingly, and applying the provided headers.
func NewJSONRequest(ctx context.Context, method, url string, body any, headers map[string]string) (*http.Request, error) {
	var reader io.Reader

	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}

		reader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return nil, err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	return req, nil
}
