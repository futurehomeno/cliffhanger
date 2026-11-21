## REMOVED Requirements

### Requirement: Empty Access Token Is A Rejection
**Reason**: A 200 is not a 401/403; only a definitive rejection may start the rejection streak.
**Migration**: None; see "Empty Access Token Is Transient".

## ADDED Requirements

### Requirement: Empty Access Token Is Transient
`auth.ProxyClient` SHALL treat a 200 answer with an empty `access_token` as a transient failure: the
attempt is retried like any other, and the returned error SHALL NOT match `httpclient.ErrUnauthorized`,
so it does not start the authenticator's rejection streak.

#### Scenario: proxy answers 200 without a token
- **WHEN** a refresh exchange keeps returning 200 with an empty `access_token` and `Retry` is 2
- **THEN** three requests are made and the error does not match `httpclient.ErrUnauthorized`
