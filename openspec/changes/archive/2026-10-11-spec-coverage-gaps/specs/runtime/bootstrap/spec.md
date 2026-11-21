## MODIFIED Requirements

### Requirement: Default Route Bundle
`DefaultRoute` SHALL return the routes every service exposes: the configuration report route, the
eight debug log-control routes and the six telemetry configuration routes — fifteen routes in total.
The `config.RoutingOption` values passed to it SHALL be forwarded to the configuration-change
publication of the debug and telemetry setter routes. The debug routes read the process log manager,
so `debug.InitializeLogger` MUST have been called before `DefaultRoute` (and therefore
`EdgeRouting`); otherwise `DefaultRoute` SHALL panic with `debug: Route called before
InitializeLogger` at wiring time rather than on the first log command.

#### Scenario: bundle contents
- **WHEN** `DefaultRoute` is called with a service name, a config getter and a live telemetry service
- **THEN** the returned slice contains fifteen routes
- **AND** `cmd.config.get_report`, `cmd.log.get_level` and `cmd.config.get_telemetry_enabled` are all
  answered by it

#### Scenario: telemetry disabled at build time
- **WHEN** `DefaultRoute` is called with a nil telemetry service
- **THEN** the six telemetry routes are omitted and the remaining routes still work

#### Scenario: logger not initialised
- **WHEN** `DefaultRoute` is called before `debug.InitializeLogger`
- **THEN** it panics instead of returning routes
