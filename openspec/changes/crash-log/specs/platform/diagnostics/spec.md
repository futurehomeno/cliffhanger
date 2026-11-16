## ADDED Requirements

### Requirement: Crash Output File
Wiring a log file SHALL open `crash.log` in the log file's directory for appending and pass it to
`runtime/debug.SetCrashOutput`, so an unrecovered panic in any goroutine and an unrecoverable runtime
fatal error are written there as well as to stderr. A `crash.log` larger than 1 MiB at setup SHALL
first be renamed to `crash.log.1`, replacing it. Once the app name and version are known
(`debug.SetCrashLogApp`, called by the root app builder), the line
`--- Start <app> v<version> <RFC3339 time> ---` SHALL be appended to `crash.log` once per process and
`crash.log` path. A failure here SHALL be logged as a warning and SHALL NOT fail logging setup.

#### Scenario: start marker
- **WHEN** the log file is wired and the app name and version are set
- **THEN** `crash.log` next to the log file ends with `--- Start <app> v<version> <time> ---`

#### Scenario: panic in an unprotected goroutine
- **WHEN** a goroutine without a deferred recover panics
- **THEN** `crash.log` contains the start marker followed by the `panic:` text and the goroutine stacks

#### Scenario: oversized crash log
- **WHEN** `crash.log` is larger than 1 MiB when the log file is wired
- **THEN** it is moved to `crash.log.1` and a new `crash.log` is started
