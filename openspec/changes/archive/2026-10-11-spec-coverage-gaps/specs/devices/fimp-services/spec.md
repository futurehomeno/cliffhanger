## ADDED Requirements

### Requirement: Battery Service
The `battery` package SHALL route `cmd.lvl.get_report` to a forced `SendBatteryLevelReport`, which
publishes the reporter's level as an integer `evt.lvl.report`. `SendBatteryAlarmReport(event, force)`
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

### Requirement: Color Control Service
The `colorctrl` package SHALL route `cmd.color.set`, whose integer-map payload is passed to
`SetColorCtrlColor` without validation against `sup_components`, followed by a forced
`evt.color.report`; and `cmd.color.get_report` to a forced `evt.color.report`. The report SHALL be
published as an integer map of the controller's component values. `Specification(...)` SHALL always
set `sup_components` and SHALL set `sup_durations` only when the supplied durations map is non-empty.

#### Scenario: setting a color
- **WHEN** `cmd.color.set` arrives with `{"red": 255, "green": 0}`
- **THEN** the controller receives exactly that map
- **AND** a forced `evt.color.report` carrying the controller's reported color is published

### Requirement: Fan Control Service
The `fanctrl` package SHALL route `cmd.mode.set` and `cmd.mode.get_report`. `SetMode` SHALL match the
requested mode case-insensitively against `sup_modes`, reject an unmatched mode before calling the
controller, and pass the advertised spelling to `SetFanCtrlMode`; a successful set SHALL be followed
by a forced `evt.mode.report`. A reported mode absent from `sup_modes` SHALL still be published,
with a warning logged.

#### Scenario: mode sent in another case
- **WHEN** `cmd.mode.set` carries `"AUTO"` and `sup_modes` is `["auto", "low"]`
- **THEN** the controller receives `"auto"`

#### Scenario: unsupported fan mode
- **WHEN** `cmd.mode.set` carries a mode not in `sup_modes`
- **THEN** the controller is not called and the handler fails, producing `evt.error.report`

### Requirement: Device System Reboot
The `devsys` package SHALL advertise `evt.error.report` and `evt.success.report` always, and
`cmd.thing.reboot` (value type `bool`) only when the controller implements `RebootController`. The
`cmd.thing.reboot` handler SHALL pass the boolean payload as the `hard` flag — a payload that is not
a boolean means a soft reboot — and SHALL confirm a successful reboot with `evt.success.report`.
`Reboot` on a controller without `RebootController` SHALL fail with a "reboot functionality is not
supported" error.

#### Scenario: hard reboot
- **WHEN** `cmd.thing.reboot` arrives with value `true` on a service whose controller implements `RebootController`
- **THEN** `RebootDevice(true)` is called and `evt.success.report` is published back

#### Scenario: reboot unsupported
- **WHEN** the controller does not implement `RebootController`
- **THEN** the specification carries no `cmd.thing.reboot` interface and `Reboot` returns an error

### Requirement: Diagnostic Reports
The `diagnostic` package SHALL support six optional reports, each enabled by its own reporter
interface: LQI (`evt.lqi.report`, int), RSSI (`evt.rssi.report`, int), reboot reason
(`evt.reboot_reason.report`, string), reboots count (`evt.reboots_count.report`, int), uptime
(`evt.uptime.report`, int) and errors (`evt.errors.report`, string array). Each matching
`cmd.<name>.get_report` SHALL publish the corresponding report on every call, without deduplication,
and SHALL fail with a "... reporting is not supported" error when the controller lacks the
interface.

#### Scenario: errors report
- **WHEN** `cmd.errors.get_report` arrives and the controller's `ErrorsReport` returns `["E1", "E2"]`
- **THEN** `evt.errors.report` is published with the string array `["E1", "E2"]`

### Requirement: Media Player Service
The `mediaplayer` package SHALL route set and get-report commands for playback (string), playback
mode (bool map), volume (int) and mute (bool), and a get-report command for metadata (string map).
`SetPlayback` SHALL reject an action not listed verbatim in `sup_playback`, and `SetPlaybackMode`
SHALL reject a map containing any key not listed verbatim in `sup_modes`, before calling the
controller; volume and mute SHALL be passed through unvalidated. A successful `cmd.playback.set`
SHALL be followed by a forced playback report and a non-forced metadata report. The periodic task
SHALL send the playback, playback mode, volume, mute and metadata reports.

#### Scenario: unsupported playback action
- **WHEN** `cmd.playback.set` carries `"next_track"` and `sup_playback` is `["play", "pause"]`
- **THEN** the controller is not called and the handler fails

#### Scenario: playback change refreshes metadata
- **WHEN** `cmd.playback.set` with `"play"` succeeds
- **THEN** `evt.playback.report` is published regardless of the cache
- **AND** `evt.metadata.report` is published only if the metadata changed since it was last reported

### Requirement: OTA Update Service
The `ota` package SHALL route `cmd.ota_update.start`, whose string payload is the firmware path; an
empty path SHALL be rejected before the controller is called. `SendStatusReport` SHALL read the
controller's `StatusReport`, validate it — a known status, progress between 0 and 100, non-negative
remaining minutes and seconds, an error from the allowed set — and fail without publishing when it
is invalid. A valid report SHALL map to: `StatusIdle` publishes nothing; `StatusStarted` publishes
`evt.ota_start.report` (null); `StatusInProgress` publishes `evt.ota_progress.report` as an int map
with `progress` and, only when positive, `remaining_min` and `remaining_sec`; `StatusDone` publishes
`evt.ota_end.report` as an object `{success, error}` where `success` is true exactly when the error
is empty. The package SHALL have no reporting task: the adapter decides when to call
`SendStatusReport`.

#### Scenario: progress report omits zero remaining time
- **WHEN** the controller reports `StatusInProgress` with progress 40 and no remaining time
- **THEN** `evt.ota_progress.report` carries only `{"progress": 40}`

#### Scenario: failed update
- **WHEN** the controller reports `StatusDone` with error `invalid_image`
- **THEN** `evt.ota_end.report` carries `{"success": false, "error": "invalid_image"}`

### Requirement: Level Switch Commands
The `outlvlswitch` package SHALL route `cmd.lvl.set`, `cmd.binary.set` and `cmd.lvl.get_report`, each
followed by a forced `evt.lvl.report`. `cmd.lvl.set` SHALL read an optional `duration` property in
seconds, defaulting to zero, and `SetLevel` SHALL clamp the level into `[min_lvl, max_lvl]` before
calling the controller, failing without calling it when either property is missing or `min_lvl`
exceeds `max_lvl`. `cmd.binary.set` SHALL call `SetLevelSwitchBinaryState`. `RouteService` SHALL
also route `cmd.lvl.start` (direction `up` or `down`, optional `start_lvl` and `duration`
properties) and `cmd.lvl.stop`, both followed by a forced level report, but the service SHALL
advertise them only when the controller implements `LevelTransitionController`, and without it both
transition methods SHALL fail. `start_lvl` SHALL be rejected when outside `[min_lvl, max_lvl]`, and
`start_lvl` and `duration` SHALL be dropped before reaching the controller unless `sup_start_lvl`
and `sup_duration` respectively are set.

#### Scenario: level above the maximum
- **WHEN** `cmd.lvl.set` carries 150 on a switch with `min_lvl` 0 and `max_lvl` 100
- **THEN** the controller is asked to set level 100 with a zero duration

#### Scenario: transition parameters the device does not support
- **WHEN** `cmd.lvl.start` carries `up` with `duration` 5 on a switch without `sup_duration`
- **THEN** the controller's `StartLevelTransition` receives a nil `Duration`

### Requirement: Level Switch Events
`outlvlswitch.SendLevelReport` SHALL publish a `*LevelEvent` carrying the level on the service's
event bus whenever it read a level: with `HasChanged()` true after a successful publication, and
false when the reporting strategy skipped the report. No event SHALL be published when reading the
level or publishing the report fails. `WaitForLevelEvent()` SHALL return a filter matching these
events.

#### Scenario: deduplicated level still emits an event
- **WHEN** `SendLevelReport(false)` is called twice with the controller returning the same level
- **THEN** two level events are published, the first with `HasChanged()` true and the second false

### Requirement: Parameter Validation
`parameters.SetParameter` SHALL validate before calling the controller: the parameter ID SHALL be
non-empty, its value type SHALL be one of `int`, `string`, `bool`, `int_array`, `string_array` and
its raw value SHALL decode as that type; the controller's specification with the same ID SHALL
exist, SHALL NOT be read-only and SHALL declare the same value type. The specification's widget SHALL
then constrain the value: `input` enforces `min`/`max` for `int`; `select` requires an `int` or
`string` value to be one of the options; `multiselect` requires every element of an `int_array` or
`string_array` to be one of the options; any other widget type SHALL be rejected.

#### Scenario: read-only parameter
- **WHEN** `cmd.param.set` targets a parameter whose specification is `read_only`
- **THEN** the controller's `SetParameter` is not called and the handler fails

#### Scenario: multiselect outside the options
- **WHEN** a `string_array` parameter contains a value absent from its `multiselect` options
- **THEN** `SetParameter` returns an error and the controller is not called

### Requirement: Supported Parameters On Inclusion
`parameters.NewInclusionReportSentEventHandler(thingRegistry)` SHALL return an event handler that,
for every adapter thing event of class inclusion-report-sent, looks up the thing by address and
sends a forced `evt.sup_params.report` from its first `parameters` service. A thing without a
`parameters` service SHALL be ignored, and an unknown address or a failed report SHALL be logged.
The adapter author SHALL register this handler; archetypes do not.

#### Scenario: inclusion report triggers the supported parameters report
- **WHEN** a thing with a `parameters` service publishes its inclusion report
- **THEN** `evt.sup_params.report` is published for that thing regardless of the reporting cache

### Requirement: Scene Control Service
The `scenectrl` package SHALL route `cmd.scene.set`, which rejects a scene not listed verbatim in
`sup_scenes` before calling the controller and is followed by a forced `evt.scene.report`, and
`cmd.scene.get_report`. The report SHALL publish only the scene name as a string, while
deduplication SHALL compare the whole `SceneReport` including its `Timestamp`, so the same scene
activated again with a new timestamp is reported again.

#### Scenario: same scene activated twice
- **WHEN** the periodic task sees scene `"movie"` with a later timestamp than the one last reported
- **THEN** `evt.scene.report` with `"movie"` is published again

### Requirement: Thermostat And Water Heater Modes
`thermostat` and `waterheater` SHALL match a requested mode case-insensitively against `sup_modes`
and a setpoint type against `sup_setpoints`, reject an unmatched value before calling the
controller, and pass the advertised spelling on. A successful `cmd.mode.set` SHALL be followed by a
forced mode report and, when the mode is also a supported setpoint type, a forced setpoint report
for it. `cmd.setpoint.get_report` SHALL take the setpoint type as a string. The setpoint report
SHALL be a string map of `type`, `temp` (formatted with one decimal) and `unit`, with storage
strategy `aggregate` keyed by the type. The periodic task SHALL send the mode report only when
`sup_modes` is non-empty, one setpoint report per `sup_setpoints` entry, and the state report only
when `sup_states` is non-empty.

#### Scenario: mode with a matching setpoint
- **WHEN** `cmd.mode.set` carries `"heat"` and `"heat"` is in both `sup_modes` and `sup_setpoints`
- **THEN** a forced `evt.mode.report` and a forced `evt.setpoint.report` for `heat` are published

### Requirement: Water Heater Setpoint Normalisation
`waterheater.SetSetpoint` SHALL, when `sup_step` is positive, round the requested temperature to the
nearest multiple of the step, and SHALL then reject a value outside the range for the setpoint type,
taken from `sup_ranges[type]` when present and otherwise from `sup_range`; without either the value
SHALL be accepted. The controller SHALL receive the rounded value.

#### Scenario: rounding to the step
- **WHEN** a setpoint of 52.3 is set with `sup_step` 0.5
- **THEN** the controller receives 52.5

#### Scenario: per-mode range overrides the global one
- **WHEN** `sup_ranges` gives `eco` the range 40–60 and `sup_range` is 30–80, and 70 is set for `eco`
- **THEN** the command is rejected and the controller is not called

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
supported and the cable lock report when the service is cable-lock aware.

#### Scenario: session report of an idle charger
- **WHEN** the session report has zero `StartedAt`, zero `FinishedAt` and zero `PreviousSessionEnergy`
- **THEN** `evt.current_session.report` carries none of `started_at`, `finished_at` and `previous_session`

### Requirement: Meter Get Report Payloads
`cmd.meter.get_report` and `cmd.meter_export.get_report` SHALL accept a `string` or `null` payload
and reject any other value type; a null payload or an empty string SHALL report every supported
(export) unit. `cmd.meter_ext.get_report` SHALL accept a `str_array` or `null` payload; a null
payload or an empty array SHALL request every value in `sup_extended_vals`, and the result SHALL be
published as one forced extended report.

#### Scenario: export report for all units
- **WHEN** `cmd.meter_export.get_report` arrives with a null value on a meter with `sup_export_units` `["kWh", "W"]`
- **THEN** a forced `evt.meter_export.report` is published for each of `kWh` and `W`

## MODIFIED Requirements

### Requirement: Service Package Structure
Every service package SHALL expose, at minimum, a `Controller` or `Reporter` interface for the
adapter author to implement, a `Service` interface embedding `adapter.Service`, a `Config` struct
whose first field is `Specification *fimptype.Service`, and a `NewService(publisher
adapter.ServicePublisher, cfg *Config) Service` constructor. Command and event names SHALL be
declared as `Cmd*`/`Evt*` constants in `routing.go` alongside the FIMP service name constant, and
the specification builder SHALL live in `specification.go`. Packages that support periodic
reporting SHALL expose `TaskReporting(...)` from `tasks.go` (`task.go` in `mediaplayer`); `devsys`,
`diagnostic`, `ota` and `parameters` have no reporting task.

#### Scenario: constructing a service
- **WHEN** an adapter author calls `NewService` with a specification built by the package's
  `Specification(...)` and a controller implementing the package's `Controller`/`Reporter`
- **THEN** a `Service` usable as an `adapter.Service` is returned
- **AND** no further wiring is needed for the package's `RouteService` and `TaskReporting` to find
  it through the `adapter.ServiceRegistry`

### Requirement: Report Emission Ordering
A service SHALL mark a value as reported only after `SendMessage` returns without error, so that a
failed publication is retried by the next reporting cycle rather than silently deduplicated away.
A failure to read from the controller SHALL be returned before anything is published. Errors
returned by `Send*Report` and by setter methods SHALL wrap the underlying controller or publisher
error, so callers can match it with `errors.Is`. Most messages also carry the service name as a
prefix, but this is not uniform: `colorctrl`, `fanctrl`, `ota`, the battery level report, scene
setting and parts of `mediaplayer` omit it. A service whose reported value is a pointer or a
slice the controller owns — `parameters` and `alarm` — SHALL cache a copy owning its own memory
rather than the controller's value itself, so a controller reusing and mutating what it returned
cannot corrupt the cached snapshot and make a genuine change look unchanged.

#### Scenario: publication failure leaves the cache untouched
- **WHEN** the controller returns a new value but `SendMessage` fails
- **THEN** `Send*Report` returns `false` and a wrapped error
- **AND** the next non-forced call still sees the value as changed and attempts the publication again

#### Scenario: controller reuses the value it returned
- **WHEN** a `parameters` controller returns the same `*Parameter` twice, mutating its `Value`
  between the two `SendParameterReport(id, false)` calls
- **THEN** the second call sees a changed value and publishes `evt.param.report` again
