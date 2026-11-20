## MODIFIED Requirements

### Requirement: Chargepoint Report Payloads
`cmd.charge.start` SHALL read the requested charging mode from the `charging_mode` property, and
both `cmd.charge.start` and `cmd.charge.stop` SHALL be followed by forced state and current session
reports; when starting or stopping charging fails, including a rejected charging mode, the command
SHALL still be followed by a forced state report and SHALL answer with the start or stop error, and a
failure of that state report SHALL be logged without replacing that error. `cmd.current_session.set_current` SHALL be
followed by a forced current session report. The
current session report SHALL publish the session energy as a float with properties
`previous_session` (two decimals, only when positive), `started_at` and `finished_at` (RFC 3339, only
when non-zero) and `offered_current` (only when offered-current adjustment is supported). The cable
lock report SHALL publish the lock state as a bool with a `cable_current` property only when the
reported cable current is non-nil. The periodic task SHALL always send the session report, the state
report when `sup_states` is non-empty, the max current report when max-current adjustment is
supported, the phase mode report when the service is phase-mode aware and the cable lock report when
the service is cable-lock aware.

#### Scenario: session report of an idle charger
- **WHEN** the session report has zero `StartedAt`, zero `FinishedAt` and zero `PreviousSessionEnergy`
- **THEN** `evt.current_session.report` carries none of `started_at`, `finished_at` and `previous_session`

#### Scenario: phase mode of an aware-only charger is polled
- **WHEN** the controller implements `PhaseModeAwareController` but not `AdjustablePhaseModeController`
- **THEN** the periodic task publishes `evt.phase_mode.report`

#### Scenario: charge start fails
- **WHEN** `cmd.charge.start` is received and `StartCharging` returns an error
- **THEN** `evt.state.report` is published and the command is answered with `evt.error.report` carrying the start error

#### Scenario: charge stop fails
- **WHEN** `cmd.charge.stop` is received and `StopCharging` returns an error
- **THEN** `evt.state.report` is published and the command is answered with `evt.error.report` carrying the stop error

#### Scenario: state read fails after a failed start
- **WHEN** `StartCharging` returns an error and reading the state for the report fails too
- **THEN** the command is answered with `evt.error.report` carrying the start error, not the state error

#### Scenario: charge start with an unsupported charging mode
- **WHEN** `cmd.charge.start` carries a `charging_mode` the charger does not support
- **THEN** the controller is not asked to start, `evt.state.report` is published and the command is answered with `evt.error.report`
