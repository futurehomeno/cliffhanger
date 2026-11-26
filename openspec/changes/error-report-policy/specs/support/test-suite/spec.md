## ADDED Requirements

### Requirement: Coded Error Expectation
`ExpectCodedError(topic, service, code)` SHALL return an expectation that matches an
`evt.error.report` on that topic and service only when its value is the given code, its correlation
ID is not empty and it has a `msg` property that holds no URL (no `://`). It is the one check every adapter
shares for a failed command; that the correlation ID is the command's UID is checked by the router's
own tests, the expectation only requires one to be present.

#### Scenario: report following the policy
- **WHEN** an `evt.error.report` with value `not_found`, a correlation ID and `msg` `service not found under the provided address` arrives
- **THEN** `ExpectCodedError(topic, service, not_found)` matches it

#### Scenario: report without a correlation ID
- **WHEN** the same report arrives with an empty correlation ID
- **THEN** the expectation does not match it

#### Scenario: report without msg
- **WHEN** the same report arrives without a `msg` property
- **THEN** the expectation does not match it

#### Scenario: URL in msg
- **WHEN** the report's `msg` is `Get "https://api.example.com/v1": timeout`
- **THEN** the expectation does not match it

#### Scenario: another code
- **WHEN** the report's value is `internal`
- **THEN** `ExpectCodedError(topic, service, not_found)` does not match it
