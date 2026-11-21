## ADDED Requirements

### Requirement: Empty Access Token Is A Rejection
`auth.ProxyClient` SHALL return an error matching `httpclient.ErrUnauthorized` when the proxy answers
200 with an empty `access_token`, so the attempt is not retried and the authenticator's grace applies.

#### Scenario: proxy answers 200 without a token
- **WHEN** a refresh exchange returns 200 with an empty `access_token`
- **THEN** an error matching `httpclient.ErrUnauthorized` is returned after a single attempt
