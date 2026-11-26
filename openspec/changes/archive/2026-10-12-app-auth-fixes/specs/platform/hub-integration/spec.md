## MODIFIED Requirements

### Requirement: Hub Token Retrieval
`hub.LoadToken(resourceName)` SHALL obtain the hub token from the Cloud Bridge service over FIMP,
using its own short-lived MQTT transport that it SHALL stop before returning. The default broker
SHALL be `tcp://localhost:1883` and the MQTT client ID prefix SHALL be
`<resourceName>_hub_token_loader`. The loader SHALL send `cmd.clbridge.get_auth_token` to
`pt:j1/mt:cmd/rt:app/rn:clbridge/ad:1` with the response topic
`pt:j1/mt:rsp/rt:app/rn:<resourceName>/ad:1` and a 5 second per-request timeout, retrying up to 7
times with 30 seconds between attempts and no delay after the final one. A response whose interface
is not `evt.clbridge.auth_token_report` SHALL be rejected as an error rather than parsed.

#### Scenario: Cloud Bridge unresponsive at boot
- **WHEN** the request times out on every attempt
- **THEN** 7 attempts are made, roughly 30 seconds apart, before an error is returned
- **AND** the error is returned right after the last attempt times out
- **AND** the MQTT transport and sync client are stopped regardless of the outcome

#### Scenario: wrong response type
- **WHEN** Cloud Bridge answers with an interface other than `evt.clbridge.auth_token_report`
- **THEN** an error naming both the expected and received interface is returned

### Requirement: Configurable Token Loader
`hub.NewTokenLoader(cfg)` SHALL build the loader behind `hub.LoadToken` from a `TokenLoaderConfig`
carrying the resource name, MQTT broker URI, username, password, client ID prefix, retry count and
retry delay. A zero broker URI or retry delay, or a retry count below 1, SHALL take the `LoadToken`
defaults, an empty client ID prefix SHALL default to the resource name, and `_hub_token_loader` SHALL
be appended to whichever prefix results. The loader SHALL default a copy and leave the caller's
config unchanged. The request SHALL carry the resource name as its source, and a report whose value
is not a string SHALL be returned as an error.

#### Scenario: custom broker
- **WHEN** `NewTokenLoader` is given `MQTTServerURI` `tcp://10.0.0.2:1883` and prefix `probe`
- **THEN** the loader connects to that broker with client ID prefix `probe_hub_token_loader`, still
  retrying 7 times 30 seconds apart

#### Scenario: negative retry
- **WHEN** `NewTokenLoader` is given `Retry: -1`
- **THEN** the loader retries 7 times like the default instead of making no request

#### Scenario: config reused
- **WHEN** one `TokenLoaderConfig` is passed to `NewTokenLoader` twice
- **THEN** the caller's config is unchanged and each loader uses the prefix `<prefix>_hub_token_loader`
