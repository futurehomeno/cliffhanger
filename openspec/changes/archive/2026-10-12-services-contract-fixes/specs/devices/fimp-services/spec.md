## MODIFIED Requirements

### Requirement: Battery Service
The `battery` package SHALL route `cmd.lvl.get_report` to a forced `SendBatteryLevelReport`, which
publishes the reporter's level as an integer `evt.lvl.report`, and SHALL advertise `evt.lvl.report`
with value type `int`. `SendBatteryAlarmReport(event, force)`
SHALL match the event case-insensitively against `sup_events`, fail with an "event is unsupported"
error before calling the reporter when it does not match, and publish `evt.alarm.report` as a string
map of `event` and `status` under the advertised spelling, with storage strategy `aggregate` keyed by
the event. A reporter returning a nil alarm report SHALL cause no publication and no error. The
periodic task SHALL send the level report and then one alarm report per supported event.

#### Scenario: device without stateful alarms
- **WHEN** `SendBatteryAlarmReport("low_battery", true)` is called and the reporter returns `nil, nil`
- **THEN** nothing is published and `false, nil` is returned

#### Scenario: unsupported battery event
- **WHEN** `SendBatteryAlarmReport` is called with an event absent from `sup_events`
- **THEN** the reporter is not called and an "event is unsupported" error is returned

#### Scenario: level report value type
- **WHEN** a battery specification is built
- **THEN** its outgoing `evt.lvl.report` interface has value type `int`

### Requirement: Color Control Service
The `colorctrl` package SHALL route `cmd.color.set`, whose integer-map payload is passed to
`SetColorCtrlColor` without validation against `sup_components`, followed by a forced
`evt.color.report`; and `cmd.color.get_report` to a forced `evt.color.report`. The report SHALL be
published as an integer map of the controller's component values. `Specification(...)` SHALL always
set `sup_components`, SHALL set `sup_durations` only when the supplied durations map is non-empty,
and SHALL advertise `cmd.color.set` with value type `int_map`. `SupportedDurations()` SHALL return
the `sup_durations` map, or nil when it is absent.

#### Scenario: setting a color
- **WHEN** `cmd.color.set` arrives with `{"red": 255, "green": 0}`
- **THEN** the controller receives exactly that map
- **AND** a forced `evt.color.report` carrying the controller's reported color is published

#### Scenario: supported durations
- **WHEN** a service is built with durations `{"default": 500}`
- **THEN** `SupportedDurations()` returns `{"default": 500}`

### Requirement: Chargepoint Report Payloads
`cmd.charge.start` SHALL read the requested charging mode from the `charging_mode` property, and
both `cmd.charge.start` and `cmd.charge.stop` SHALL be followed by forced state and current session
reports; `cmd.current_session.set_current` SHALL be followed by a forced current session report. The
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

### Requirement: Setpoint String Map Validation
`thermostat.SetpointFromStringMap` SHALL reject a `temp` that parses to NaN or ±Inf. The water heater
SHALL use the same setpoint type and helpers. Both services SHALL key their setpoint report cache by the
normalized mode and SHALL compare the whole setpoint, value and unit, when deciding whether to report.

#### Scenario: NaN setpoint on a thermostat
- **WHEN** `cmd.setpoint.set` carries `temp` "NaN"
- **THEN** the command is rejected and the controller is not called

#### Scenario: setpoint mode sent in another case
- **WHEN** a thermostat or water heater setpoint is set with type "HEAT" and the periodic task then reports "heat" unchanged
- **THEN** no second report is published

#### Scenario: setpoint unit changes
- **WHEN** the controller reports the same setpoint value with unit "F" after it was reported with unit "C"
- **THEN** `evt.setpoint.report` is published again

## ADDED Requirements

### Requirement: Missing Parameter Report
`parameters.SendParameterReport` SHALL return a "parameter not found" error, without publishing, when
the controller's `GetParameter` returns a nil parameter and no error.

#### Scenario: controller returns no parameter
- **WHEN** `SendParameterReport("missing", true)` is called and `GetParameter` returns `nil, nil`
- **THEN** nothing is published and a "parameter not found" error is returned
