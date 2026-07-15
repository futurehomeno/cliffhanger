package auth

import (
	"net/http"
)

// TokenSource provides a valid bearer token. It is satisfied by Authenticator.
type TokenSource interface {
	AccessToken() (string, error)
}

// Transport is an http.RoundTripper injecting a bearer token into every request and
// reporting unauthorized responses through the optional callback. A single unauthorized
// response can be transient — prefer concluding authorization loss from the refresh path
// (Authenticator.OnAuthLoss) and use the callback only where no periodic probe exists.
type Transport struct {
	Base           http.RoundTripper
	Source         TokenSource
	OnUnauthorized func()
}

func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.Base
	if base == nil {
		base = http.DefaultTransport
	}

	// The bearer must not leak to another host on a redirect hop.
	first := req
	for first.Response != nil {
		first = first.Response.Request
	}

	if first.URL.Host != req.URL.Host {
		return base.RoundTrip(req)
	}

	token, err := t.Source.AccessToken()
	if err != nil {
		return nil, err
	}

	clone := *req
	clone.Header = req.Header.Clone()
	clone.Header.Set("Authorization", "Bearer "+token)
	req = &clone

	resp, err := base.RoundTrip(req)
	if err == nil && resp.StatusCode == http.StatusUnauthorized && t.OnUnauthorized != nil {
		t.OnUnauthorized()
	}

	return resp, err
}
