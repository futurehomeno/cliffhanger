# OAuth2 Authentication Specification

## Purpose
Edge adapters talk to a vendor cloud with an OAuth2 bearer token that expires long before the
adapter does, and the hub has no user present to re-authorize on demand. This capability keeps a
usable access token available: it refreshes proactively before expiry through the Futurehome OAuth
proxy, serializes and paces refresh attempts so a flapping partner API cannot be hammered, and
distinguishes a transient failure from a genuine loss of authorization that only a new login can
repair. It also supplies the bearer transport, credential persistence contract and JWT expiry
helper the adapters build on.

## Requirements

### Requirement: Proactive Access Token Refresh
`auth.Authenticator.AccessToken` SHALL return the stored access token unchanged while it is more
than `RefreshLead` away from `ExpiresAt`, and SHALL exchange the refresh token for new credentials
once it is within that lead. `RefreshLead` SHALL default to 5 minutes when left zero. When the
stored credentials are empty — both access and refresh token blank — `AccessToken` SHALL return
`ErrNotLoggedIn` without contacting the exchanger.

#### Scenario: Token still fresh
- **WHEN** `AccessToken` is called and the stored token expires in more than `RefreshLead`
- **THEN** the stored access token is returned
- **AND** no token exchange is performed

#### Scenario: Token inside the refresh lead
- **WHEN** `AccessToken` is called and the stored token expires in less than `RefreshLead`
- **THEN** `TokenExchanger.ExchangeRefreshToken` is called with the stored refresh token
- **AND** the resulting access token is returned and the new credentials are handed to
  `CredentialsStore.SetCredentials`

#### Scenario: No credentials stored
- **WHEN** `AccessToken` is called and `CredentialsStore.Credentials` returns empty credentials
- **THEN** `ErrNotLoggedIn` is returned

### Requirement: Single-Flight Refresh
The `Authenticator` SHALL hold one internal mutex for the whole of `AccessToken`, so concurrent
callers never issue overlapping refreshes and the second caller observes the first caller's result.
The `OnAuthLoss` callback SHALL be invoked with that lock held, and therefore SHALL NOT call back
into the `Authenticator`.

#### Scenario: Concurrent callers during an expiry
- **WHEN** several goroutines call `AccessToken` at once with an access token inside the refresh lead
- **THEN** exactly one `ExchangeRefreshToken` call is made
- **AND** every caller receives the token produced by that single exchange

### Requirement: Backoff Suppression Of Refresh Attempts
The `Authenticator` SHALL pace failed refreshes with the configured `backoff.Stateful`, defaulting
to `backoff.NewTolerantFixed(2, 5*time.Minute)`: the first two consecutive failures are not delayed
and every later failure imposes a fixed 5 minute wait. While the backoff suppresses an attempt and
the access token has already expired, `AccessToken` SHALL return `ErrRefreshSuspended` rather than
attempting an exchange. A successful exchange and an authorization loss SHALL both reset the
backoff.

#### Scenario: Refresh suppressed by backoff
- **WHEN** the backoff reports that an attempt should be delayed and the stored access token is expired
- **THEN** `ErrRefreshSuspended` is returned
- **AND** no call is made to the token exchanger

#### Scenario: Backoff cleared by success
- **WHEN** a refresh attempt succeeds after earlier failures
- **THEN** the backoff is reset so the next failure is tolerated again

### Requirement: Valid Token Survives A Failed Refresh
A refresh that cannot be performed or does not succeed SHALL NOT invalidate an access token that
has not yet reached `ExpiresAt`. On the backoff-suppressed path, the doomed-refresh-token path and
the exchange-error path alike, `AccessToken` SHALL return the still-valid stored access token
instead of an error.

#### Scenario: Transient exchange failure inside the refresh lead
- **WHEN** `ExchangeRefreshToken` fails and the stored access token has not yet expired
- **THEN** the stored access token is returned with no error

#### Scenario: Exchange failure after expiry
- **WHEN** `ExchangeRefreshToken` fails with a non-authorization error and the stored access token
  has already expired
- **THEN** the wrapped exchange error is returned

### Requirement: Unauthorized Grace Window
When `UnauthorizedGrace` is non-zero, the `Authenticator` SHALL tolerate refresh rejections
(`httpclient.ErrUnauthorized`) for that long, measured from the first rejection of the current
refresh token, before concluding authorization loss. Inside the window with an expired access
token, `AccessToken` SHALL return `ErrRefreshDeferred`, which SHALL NOT match
`errors.Is(err, httpclient.ErrUnauthorized)` so that a connectivity probe wrapping this transport
does not read a single rejection as authorization loss. A rejection streak SHALL be tied to the
refresh token that produced it, so credentials replaced out of band start a fresh grace window. A
zero `UnauthorizedGrace` SHALL conclude authorization loss on the first rejection.

#### Scenario: First rejection within grace
- **WHEN** `ExchangeRefreshToken` returns `httpclient.ErrUnauthorized`, the access token has expired
  and `UnauthorizedGrace` has not elapsed
- **THEN** `ErrRefreshDeferred` is returned
- **AND** the credentials are not cleared and `OnAuthLoss` is not invoked

#### Scenario: Refresh token replaced during a streak
- **WHEN** a rejection streak is open and the store returns a different refresh token
- **THEN** the streak timestamp is restarted against the new token before the grace is evaluated

#### Scenario: Grace elapses while backoff suppresses attempts
- **WHEN** the backoff suppresses new exchanges, the access token has expired, the open rejection
  streak belongs to the current refresh token and `UnauthorizedGrace` has elapsed
- **THEN** authorization loss is concluded and an error wrapping `ErrReloginRequired` is returned

### Requirement: Terminal Authorization Loss
Once the refresh token can no longer produce an access token, `AccessToken` SHALL return an error
wrapping `ErrReloginRequired` and SHALL clear the persisted credentials through
`CredentialsStore.ClearCredentials`, reset the backoff, drop any rejection streak and any unsaved
in-memory credentials, and invoke `OnAuthLoss` with a reason naming the cause. For a rejected
refresh token the returned error SHALL wrap both `httpclient.ErrUnauthorized` and
`ErrReloginRequired`, so a connectivity probe can map the outcome onto the auth-loss lifecycle
state. `ErrReloginRequired` SHALL be terminal: only a new login clears it.

#### Scenario: Refresh token rejected past the grace window
- **WHEN** `ExchangeRefreshToken` returns `httpclient.ErrUnauthorized` and no grace remains
- **THEN** `ClearCredentials` is called and `OnAuthLoss` is invoked with the rejection reason
- **AND** the returned error matches both `httpclient.ErrUnauthorized` and `ErrReloginRequired`

#### Scenario: Failure to clear credentials
- **WHEN** `ClearCredentials` returns an error during authorization loss
- **THEN** the error is logged and `OnAuthLoss` is still invoked

### Requirement: Locally Expired Refresh Token
When `Credentials.RefreshExpiresAt` is set and already in the past, the `Authenticator` SHALL skip
the exchange as doomed. If the access token is still valid it SHALL be returned; otherwise
authorization loss SHALL be concluded with the reason `refresh token expired` and an error wrapping
`ErrReloginRequired` returned. A zero `RefreshExpiresAt` SHALL disable this check.

#### Scenario: Refresh token past its own expiry
- **WHEN** `AccessToken` is called with `RefreshExpiresAt` in the past and an expired access token
- **THEN** no exchange is attempted, the credentials are cleared and an error wrapping
  `ErrReloginRequired` is returned

### Requirement: Credentials Derivation And Refresh Token Carry-Over
`OAuth2TokenResponse.Credentials` SHALL produce credentials whose `ExpiresAt` is the current time
plus `ExpiresIn` seconds. When the exchange response carries no refresh token, or repeats the one
that was sent, the `Authenticator` SHALL keep the previous `RefreshToken` together with its
`RefreshExpiresAt` rather than overwriting them with zero values.

#### Scenario: Provider omits the refresh token
- **WHEN** an exchange succeeds and the response's `refresh_token` is empty
- **THEN** the stored credentials keep the previous refresh token and its `RefreshExpiresAt`

#### Scenario: Provider rotates the refresh token
- **WHEN** an exchange succeeds and returns a refresh token different from the one sent
- **THEN** the new refresh token is stored and `RefreshExpiresAt` is cleared

### Requirement: Retention Of Unsaved Rotated Credentials
When an exchange succeeds but `SetCredentials` fails, the `Authenticator` SHALL keep the new
credentials in memory and serve them, because a rotating provider may already have invalidated the
stored refresh token. The memory copy SHALL be preferred only while the store still holds the
refresh token it replaced, so credentials changed out of band — a fresh login or a logout — win.
Persistence SHALL be retried only on the refresh path, not on every `AccessToken` call.

#### Scenario: Store rejects a rotation
- **WHEN** `ExchangeRefreshToken` succeeds and `SetCredentials` returns an error
- **THEN** the new access token is still returned
- **AND** the next refresh uses the in-memory refresh token and retries the failed write first

#### Scenario: Credentials replaced out of band
- **WHEN** an unsaved rotation is held and the store's refresh token no longer matches the one it
  replaced
- **THEN** the store's credentials are used and the in-memory copy is discarded

### Requirement: Proxy Token Exchange
`auth.ProxyClient` SHALL exchange tokens against the Futurehome OAuth proxy by POSTing JSON to
`<URL>/api/control/edge/proxy/auth-code` for `ExchangeAuthorizationCode` (body `{code, partnerCode}`)
and `<URL>/api/control/edge/proxy/refresh` for `ExchangeRefreshToken` (body
`{refreshToken, partnerCode}`), sending `Content-Type: application/json` and
`Authorization: Bearer <Token>`. It SHALL retry a failed attempt up to `Retry` further times,
sleeping `RetryDelay` between them and replaying the request body each attempt, but SHALL return
immediately without retrying when the failure matches `httpclient.ErrUnauthorized` (rejection is
definitive) or `httpclient.ErrTooManyRequests` (the caller's backoff paces the next attempt). Any
non-200 response SHALL be mapped through `httpclient.ErrorFromResponse`. Unset configuration SHALL
default `URL` to `ProxyURL(hub.EnvProd)` and `Timeout` to 60 seconds; additional `Headers` SHALL be
added to each request except `Authorization` and `Content-Type`, which cannot be overridden.

#### Scenario: Refresh rejected by the proxy
- **WHEN** the proxy answers a refresh exchange with 401 or 403
- **THEN** an error matching `httpclient.ErrUnauthorized` is returned after a single attempt

#### Scenario: Proxy rate limits the exchange
- **WHEN** the proxy answers with 429
- **THEN** an error matching `httpclient.ErrTooManyRequests` is returned without a local retry delay

#### Scenario: Custom header collides with the bearer
- **WHEN** the client is configured with an `Authorization` entry in `Headers`
- **THEN** the request still carries `Bearer <Token>` from the configured `Token`

### Requirement: Environment-Keyed Proxy Endpoints
`ProxyURL` and `ProxyCallbackURL` SHALL select their host from `hub.Environment`: `hub.EnvBeta`
yields `https://partners-beta.futurehome.io` and
`https://app-static-beta.futurehome.io/playground_oauth_callback`; every other value, including
`hub.EnvProd`, yields `https://partners.futurehome.io` and
`https://app-static.futurehome.io/playground_oauth_callback`.

#### Scenario: Beta hub
- **WHEN** `ProxyURL(hub.EnvBeta)` is called
- **THEN** `https://partners-beta.futurehome.io` is returned

#### Scenario: Unknown environment
- **WHEN** `ProxyURL` is called with an environment other than `hub.EnvBeta`
- **THEN** the production host `https://partners.futurehome.io` is returned

### Requirement: Bearer Injection Transport
`auth.Transport` SHALL implement `http.RoundTripper` by obtaining a token from its `TokenSource`
and setting `Authorization: Bearer <token>` on a clone of the request, leaving the caller's request
and headers untouched. A `TokenSource` error SHALL abort the round trip and be returned to the
caller without sending the request. `Base` SHALL default to `http.DefaultTransport`. The bearer
SHALL be dropped for the remainder of a redirect chain as soon as any hop changed host or
downgraded from `https` to plaintext, and a hop the transport cannot verify — one whose
`Response.Request` or its URL is nil, as produced by a synthesizing `Base` — SHALL count as
stripped. When the response status is 401 and `OnUnauthorized` is set, the callback SHALL be
invoked.

#### Scenario: Redirect to another host
- **WHEN** a request is the result of a redirect from a different host
- **THEN** the request is forwarded to `Base` without an `Authorization` header

#### Scenario: Redirect bouncing back to the original host
- **WHEN** a chain redirects from host a to host b and back to host a
- **THEN** the bearer stays stripped for the final hop

#### Scenario: No token available
- **WHEN** `TokenSource.AccessToken` returns an error
- **THEN** `RoundTrip` returns that error and performs no HTTP request

### Requirement: JWT Expiry Inspection
`auth.TokenExpirationDate` SHALL parse a JWT without verifying its signature and return the `exp`
registered claim converted to UTC. When the token cannot be parsed, or carries no `exp` claim, it
SHALL return an error and a zero time.

#### Scenario: Token without an expiry claim
- **WHEN** `TokenExpirationDate` is given a syntactically valid JWT with no `exp` claim
- **THEN** an error is returned and the time is zero

#### Scenario: Token with an expiry claim
- **WHEN** `TokenExpirationDate` is given a JWT carrying `exp`
- **THEN** the claim is returned as a UTC time regardless of the signature's validity
