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

	// The bearer must not leak to another host or onto plaintext on a redirect hop, and it
	// must stay dropped for the rest of the chain: comparing only the first and the current
	// hop would reattach it on an a -> b -> a bounce, letting b pick the path it is sent to.
	if strippedOnRedirect(req) {
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

// strippedOnRedirect reports whether any hop of the redirect chain leading to req left the
// previous host or downgraded it to plaintext, which drops the bearer for good.
func strippedOnRedirect(req *http.Request) bool {
	for hop := req; hop.Response != nil; {
		previous := hop.Response.Request

		if previous.URL.Host != hop.URL.Host || (previous.URL.Scheme == "https" && hop.URL.Scheme != "https") {
			return true
		}

		hop = previous
	}

	return false
}
