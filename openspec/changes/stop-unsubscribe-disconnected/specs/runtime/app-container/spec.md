## MODIFIED Requirements

### Requirement: Best-Effort Shutdown
`Stop` SHALL attempt every shutdown step even when an earlier one fails, SHALL return the collected
failures joined with `errors.Join`, and SHALL mark the application as not running regardless. It
SHALL flush buffered diagnostics through a deferred `debug.FlushLogs`, so the log lines leading up to
a failing shutdown reach the log file. It SHALL skip the topic unsubscriptions when the MQTT
transport is not connected, since a lost connection holds no subscriptions to remove; an unsubscribe
failure on a connected transport SHALL still be returned.

#### Scenario: a failing step still flushes logs
- **WHEN** `taskManager.Stop()` fails because the manager was never started
- **THEN** `Stop` returns a non-nil error
- **AND** the log lines buffered before the failure are present in the configured log file

#### Scenario: stop after broker loss
- **WHEN** the application stops while its MQTT transport is not connected
- **THEN** no topic unsubscribe is attempted
- **AND** `Stop` returns nil when every other step succeeds

#### Scenario: unsubscribe failure on a connected transport
- **WHEN** the application stops while its MQTT transport is connected and an unsubscribe fails
- **THEN** `Stop` returns an error naming the topic
