## MODIFIED Requirements

### Requirement: Bearer Injection Transport
`auth.Transport` SHALL implement `http.RoundTripper` by obtaining a token from its `TokenSource`
and setting `Authorization: Bearer <token>` on a clone of the request, leaving the caller's request
and headers untouched. A `TokenSource` error SHALL abort the round trip and be returned to the
caller without sending the request. `Base` SHALL default to `http.DefaultTransport`. The bearer
SHALL be dropped for the remainder of a redirect chain as soon as any hop changed host or
downgraded from `https` to plaintext, and a hop the transport cannot verify — one whose
`Response.Request` or its URL is nil, as produced by a synthesizing `Base` — SHALL count as
stripped. A stripped hop SHALL be forwarded to `Base` unchanged, without asking the `TokenSource`
for a token and without invoking `OnUnauthorized`. When a bearer-carrying request gets a 401 or 403
response and `OnUnauthorized` is set, the callback SHALL be invoked; a round-trip error SHALL NOT
invoke it.

#### Scenario: Redirect to another host
- **WHEN** a request is the result of a redirect from a different host
- **THEN** the request is forwarded to `Base` without an `Authorization` header

#### Scenario: Redirect bouncing back to the original host
- **WHEN** a chain redirects from host a to host b and back to host a
- **THEN** the bearer stays stripped for the final hop

#### Scenario: No token available
- **WHEN** `TokenSource.AccessToken` returns an error
- **THEN** `RoundTrip` returns that error and performs no HTTP request

#### Scenario: Forbidden response
- **WHEN** a request carrying the bearer is answered with 403
- **THEN** `OnUnauthorized` is invoked exactly as for a 401

## ADDED Requirements

### Requirement: Successful Exchange Ends A Rejection Streak
A successful `ExchangeRefreshToken` SHALL clear any open rejection streak, whether or not the new
credentials could be persisted afterwards, so a later rejection starts a fresh `UnauthorizedGrace`
window instead of inheriting the elapsed time of the refuted one.

#### Scenario: Rejection, success, rejection
- **WHEN** a refresh is rejected inside the grace window, the next refresh succeeds and a later
  refresh is rejected again after the original grace would have elapsed
- **THEN** the later rejection is still inside a fresh grace window and authorization loss is not
  concluded
