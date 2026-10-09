## ADDED Requirements

### Requirement: Connector Calls Outside The Thing Lock
A thing SHALL call its connector's `Connectivity()` and `Ping()` without holding the thing's lock,
because a connector may wait on a device or a cloud and the lock would stall every other caller on
the thing for that long. The calls SHALL still run one at a time per thing, and
`SendConnectivityReport` SHALL read connectivity and publish it in one such turn, so a stale report
is never published or cached after a fresher one and a connector need not be safe for concurrent
calls.

#### Scenario: Connectivity is slow
- **WHEN** `Connectivity()` is blocked inside `ConnectivityReport` or `SendConnectivityReport`
- **THEN** `Update` and other calls that lock the thing still complete

#### Scenario: Ping is slow
- **WHEN** `Ping()` is blocked inside `SendPingReport`
- **THEN** `Update` and other calls that lock the thing still complete

#### Scenario: Two reports race
- **WHEN** `SendConnectivityReport` is waiting on `Connectivity()` and another connectivity report,
  connectivity read or ping starts on the same thing
- **THEN** the second call reaches the connector only after the first report has been published
