## ADDED Requirements

### Requirement: Optional Handler Locker
The `locker` argument of `RouteCmdConfigExtendedSet`, `RouteCmdAppUninstall`, `RouteCmdAppReset`,
`RouteConfigActionCommand`, `RouteCmdAuthLogin`, `RouteCmdAuthSetTokens`, `RouteCmdAuthLogout` and
their `Handle*` counterparts SHALL be optional. A nil locker SHALL make the handler run every
message without serialization and never reply with the "another operation is already running"
error; a non-nil locker SHALL be released when the handler returns, so the next mutating command
can run.

#### Scenario: handler built without a locker
- **WHEN** `HandleCmdConfigExtendedSet` is built with a nil locker and two messages are handled
- **THEN** `Configure` is called for both and both reply with `evt.app.config_report`

#### Scenario: lock released after a command
- **WHEN** a mutating handler sharing a locker has finished processing a message
- **THEN** the next mutating command on the same locker runs normally

## MODIFIED Requirements

### Requirement: Diagnostic Report
The `cmd.app.get_diag` handler SHALL reply with `evt.app.diag_report` carrying a `DiagReport`:
`uptime` in whole seconds since the `Lifecycle` was created by `lifecycle.New`, `restarts_count`
as the persisted restart counter the `Lifecycle` incremented at construction (`0` when it was built
without a store), and `errors` as returned by the `LogProvider`. A `LogProvider` returning an error
SHALL abort the reply with "failed to retrieve error log entries", so the router publishes an error
report and never a partial diag report. A nil `LogProvider` SHALL NOT be passed to
`RouteCmdAppDiagGetReport`: the handler dereferences it unconditionally, and the resulting panic is
contained by the router's per-message recovery, leaving the command silently unanswered.

#### Scenario: hub requests diagnostics
- **WHEN** `cmd.app.get_diag` arrives and `ErrorsReport` returns two entries
- **THEN** `evt.app.diag_report` is published with both entries, the current uptime and the restart
  count

#### Scenario: log provider fails
- **WHEN** `ErrorsReport` returns an error
- **THEN** no `evt.app.diag_report` is published
