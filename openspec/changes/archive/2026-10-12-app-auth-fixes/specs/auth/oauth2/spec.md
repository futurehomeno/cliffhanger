## MODIFIED Requirements

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
default `URL` to `ProxyURL(hub.EnvProd)` and `Timeout` to 60 seconds, and a negative `Retry` SHALL
count as 0; additional `Headers` SHALL be added to each request except `Authorization` and
`Content-Type`, which cannot be overridden.

#### Scenario: Refresh rejected by the proxy
- **WHEN** the proxy answers a refresh exchange with 401 or 403
- **THEN** an error matching `httpclient.ErrUnauthorized` is returned after a single attempt

#### Scenario: Proxy rate limits the exchange
- **WHEN** the proxy answers with 429
- **THEN** an error matching `httpclient.ErrTooManyRequests` is returned without a local retry delay

#### Scenario: Custom header collides with the bearer
- **WHEN** the client is configured with an `Authorization` entry in `Headers`
- **THEN** the request still carries `Bearer <Token>` from the configured `Token`

#### Scenario: negative retry
- **WHEN** the client is configured with `Retry: -1` and the proxy answers 502
- **THEN** exactly one request is made and an error is returned
