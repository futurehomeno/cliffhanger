package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
)

// RequestBuilder builds an HTTP request with optional JSON body and headers.
type RequestBuilder struct {
	method  string
	url     string
	body    any
	headers map[string]string
}

func NewRequest(method, url string) *RequestBuilder {
	return &RequestBuilder{
		method:  method,
		url:     url,
		headers: make(map[string]string),
	}
}

// WithJSONBody sets the request body to the JSON encoding of v and the Content-Type header accordingly.
func (b *RequestBuilder) WithJSONBody(v any) *RequestBuilder {
	b.body = v
	b.headers["Content-Type"] = "application/json"

	return b
}

func (b *RequestBuilder) WithHeader(key, value string) *RequestBuilder {
	b.headers[key] = value

	return b
}

func (b *RequestBuilder) Build(ctx context.Context) (*http.Request, error) {
	var body io.Reader

	if b.body != nil {
		data, err := json.Marshal(b.body)
		if err != nil {
			return nil, err
		}

		body = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, b.method, b.url, body)
	if err != nil {
		return nil, err
	}

	for key, value := range b.headers {
		req.Header.Set(key, value)
	}

	return req, nil
}
