## ADDED Requirements

### Requirement: Connector Calls Outside The Thing Lock
A thing SHALL call its connector's `Connectivity()` and `Ping()` without holding the thing's lock,
because a connector may wait on a device or a cloud and the lock would stall every other caller on
the thing for that long.

#### Scenario: Connectivity is slow
- **WHEN** `Connectivity()` is blocked inside `ConnectivityReport` or `SendConnectivityReport`
- **THEN** `Update` and other calls that lock the thing still complete

#### Scenario: Ping is slow
- **WHEN** `Ping()` is blocked inside `SendPingReport`
- **THEN** `Update` and other calls that lock the thing still complete
