## MODIFIED Requirements

### Requirement: Configuration Reports Never Fail The Message
The `cmd.config.extended_set` and `cmd.app.uninstall` handlers SHALL always reply with a
`ConfigurationReport` — `{op_status, op_error, app_state}` — and SHALL NOT surface a processing
error. On success `op_status` SHALL be `ok`; on a payload unmarshalling failure or a failing
`Configure` it SHALL be `error` with `op_error` set to `configure the app err: <error>`, and on a
failing `Uninstall` it SHALL be `error` with `op_error` set to `uninstall the app err: <error>`.
The report SHALL always carry the current lifecycle snapshot in `app_state`, so the hub learns the
resulting state even when the operation failed.

#### Scenario: configuration rejected by the application
- **WHEN** `Configure` returns an error
- **THEN** `evt.app.config_report` is published with `op_status` `error`, a populated `op_error` and
  the current app state

#### Scenario: unparsable configuration payload
- **WHEN** the `cmd.config.extended_set` value cannot be unmarshalled into the configuration type
- **THEN** `Configure` is not called and the same `evt.app.config_report` error shape is published

#### Scenario: uninstall fails
- **WHEN** `Uninstall` returns `boom`
- **THEN** `evt.app.uninstall_report` is published with `op_status` `error` and `op_error`
  `uninstall the app err: boom`

### Requirement: Lifecycle Capability Interfaces
`InitializableApp` and `CheckableApp` SHALL be asserted by `TaskApp`, not by `RouteApp`, and an
application implementing neither SHALL get no background tasks. `Initialize` SHALL run once at
startup and, if it returns an error, SHALL set the app health to startup error and SHALL be retried
every 10 minutes for as long as that state holds; the retry SHALL NOT run at startup alongside the
first attempt. `Check` SHALL run only while the app health is running, at the interval returned by
`CheckInterval()`, and SHALL fall back to 30 minutes when that interval is 0. A failing `Check` SHALL
only be logged.

#### Scenario: initialization fails at boot
- **WHEN** `Initialize` returns an error during startup
- **THEN** the app health becomes startup error and initialization is retried 10 minutes later
- **AND** `Initialize` is called exactly once at startup

#### Scenario: default check interval
- **WHEN** the application implements `CheckableApp` and returns 0 from `CheckInterval`
- **THEN** `Check` is scheduled every 30 minutes while the app is running
