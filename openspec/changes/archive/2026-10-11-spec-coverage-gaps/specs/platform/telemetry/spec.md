## ADDED Requirements

### Requirement: Suppression Report Shape
`Suppressed()` — the value answered by `cmd.config.get_telemetry_suppressed` and
`cmd.config.set_telemetry_suppressed` — SHALL return an empty map when no suppression is stored, and
otherwise a single entry keyed by the adapter's own resource name holding copies of the stored
`Domains` and `Events`, so app-wide suppression is reported as an entry with both lists empty.

#### Scenario: nothing suppressed
- **WHEN** the stored configuration has no `Suppressed` entry
- **THEN** `Suppressed()` returns an empty, non-nil map

#### Scenario: app-wide suppression
- **WHEN** suppression for the whole app is stored
- **THEN** `Suppressed()` returns one entry for this resource name with no domains and no events

### Requirement: Cloud Report Restarts The Validity Window
Each received `evt.telemetry.config_report` SHALL be applied through `Enable` and `SetSuppressed`, so
a report with `enabled: true` re-stamps `EnabledAt` and re-arms the validity timer for the full
validity, while a report with `enabled: false` disables telemetry and clears `EnabledAt`. A failure
to persist either part SHALL be logged and SHALL NOT stop the other part from being applied.

#### Scenario: cloud keeps telemetry on
- **WHEN** a report with `enabled: true` arrives 20 days into a 30 day window
- **THEN** `EnabledAt` becomes now and the window runs a full 30 days from the report

### Requirement: Config Poll Lifecycle
`config_poll.Config.Start` SHALL be a no-op while the poller is already running, and `Stop` SHALL be
a no-op before `Start`. `Stop` SHALL cancel the scheduled poll, close the listener and wait for its
goroutine to exit and unregister its MQTT channel. Each `Start` SHALL register its listener under a
fresh channel name, so a `Start` that follows a `Stop` is not torn down by the previous goroutine's
deferred unregister. A config report whose value cannot be decoded SHALL be logged and ignored
without applying anything or rescheduling, and a report handled after `Stop` SHALL NOT schedule a
new poll.

#### Scenario: stop then start
- **WHEN** `Stop` is followed immediately by `Start`
- **THEN** config reports on the broadcast topic are still delivered to the new listener

#### Scenario: malformed report
- **WHEN** an `evt.telemetry.config_report` carries a value that is not a config object
- **THEN** neither enablement nor suppression changes and the pending poll keeps its schedule
