# FIMP Service Catalogue Specification

## Purpose
The `adapter/service/*` packages implement the FIMP services a thing can expose on the hub — one
package per service family, each translating between the adapter author's device-specific
`Controller`/`Reporter` code and the FIMP messages on MQTT. Every package follows the same
four-file contract (`service.go`, `specification.go`, `routing.go`, `tasks.go`), so an adapter
author learns one shape and reuses it across all 19 packages. This capability defines that shared
contract — specification building, reporting deduplication, routing and periodic tasks — plus the
behaviour specific to the service families that deviate from it.
## Requirements
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

### Requirement: Service Catalogue
The framework SHALL provide the following FIMP services, one package each:

| package | FIMP service name(s) |
| --- | --- |
| `alarm` | `alarm_system` |
| `battery` | `battery` |
| `chargepoint` | `chargepoint` |
| `colorctrl` | `color_ctrl` |
| `devsys` | `dev_sys` |
| `diagnostic` | `diagnostic` |
| `fanctrl` | `fan_ctrl` |
| `mediaplayer` | `media_player` |
| `numericmeter` | `meter_elec`, `meter_gas`, `meter_water`, `meter_heating`, `meter_cooling` |
| `numericsensor` | the `sensor_*` family (`sensor_temp`, `sensor_humid`, `sensor_co2`, ... 39 in total) |
| `ota` | `ota` |
| `outbinswitch` | `out_bin_switch` |
| `outlvlswitch` | `out_lvl_switch` |
| `parameters` | `parameters` |
| `presence` | `sensor_presence` |
| `scenectrl` | `scene_ctrl` |
| `thermostat` | `thermostat` |
| `virtualmeter` | `virtual_meter_elec` |
| `waterheater` | `water_heater` |

Packages whose service name is fixed (`alarm`, `battery`, `chargepoint`, `colorctrl`, `devsys`,
`diagnostic`, `fanctrl`, `ota`, `parameters`, `thermostat`, `waterheater`) SHALL overwrite
`cfg.Specification.Name` in `NewService`. `numericmeter` and `numericsensor` SHALL instead take the
service name as an argument to `Specification(...)`.

#### Scenario: a mis-named specification is corrected
- **WHEN** `alarm.NewService` receives a specification whose `Name` is not `alarm_system`
- **THEN** the specification's `Name` is overwritten with `alarm_system` before the service is built

#### Scenario: choosing a meter family
- **WHEN** an adapter author calls `numericmeter.Specification(numericmeter.MeterGas, ...)`
- **THEN** the resulting specification carries the service name `meter_gas`
- **AND** `numericmeter.RouteService` routes its `cmd.meter.get_report` because routing matches the
  `meter_` service-name prefix, not an exact name

### Requirement: Specification Construction
Each package's `Specification(...)` SHALL return a `*fimptype.Service` whose `Address` is
`/rt:dev/rn:<resourceName>/ad:<resourceAddress>/sv:<serviceName>/ad:<address>`, whose `Enabled` is
`true`, whose `Groups` are the caller's groups, and whose `Interfaces` are the package's mandatory
interfaces — including an outgoing `evt.error.report` of value type `string` for services that
accept commands. `Specification(...)` in `chargepoint`, `numericmeter`, `outbinswitch`,
`outlvlswitch`, `mediaplayer` and `virtualmeter` SHALL additionally accept variadic
`adapter.SpecificationOption` values, applied in order after the base specification is built; the
`With*` helpers those packages export SHALL each set one entry in `Props`.

#### Scenario: options set specification properties
- **WHEN** `chargepoint.Specification(..., WithSupportedMaxCurrent(32), WithChargingModes("normal", "slow"))` is called
- **THEN** the returned specification carries `sup_max_current = 32` and `sup_charging_modes = ["normal", "slow"]`
- **AND** `sup_states` is populated from the mandatory `supportedStates` argument

### Requirement: Capability-Driven Interface Advertisement
Optional device capabilities SHALL be expressed as additional Go interfaces that the concrete
controller may implement, never as boolean configuration flags. `NewService` SHALL type-assert the
supplied controller against each optional interface, combine the result with the relevant
specification property where one gates the capability, expose the outcome through a `Supports*()`
or `Is*Aware()` method, and call `EnsureInterfaces` to append the corresponding FIMP interfaces to
the specification. `EnsureInterfaces` SHALL be idempotent: an interface already present is not
duplicated, so a caller may declare interfaces by hand without breaking the specification.

#### Scenario: capability advertised only when both halves are present
- **WHEN** a chargepoint controller implements `AdjustableMaxCurrentController` but the
  specification carries no `sup_max_current` property
- **THEN** `SupportsAdjustingMaxCurrent()` returns false
- **AND** `cmd.max_current.set`, `cmd.max_current.get_report` and `evt.max_current.report` are not
  added to the specification
- **AND** `SetMaxCurrent` and `SendMaxCurrentReport` return an "adjusting max current is not
  supported" error

#### Scenario: diagnostic capabilities are controller-only
- **WHEN** a `diagnostic` controller implements `UptimeReporter` and `ErrorsReporter` but none of
  the other reporter interfaces
- **THEN** the specification gains only the uptime and errors interfaces
- **AND** `SendLQIReport` returns a "LQI reporting is not supported" error rather than publishing

### Requirement: Report Deduplication
A service SHALL consult a `cache.ReportingCache` before publishing a periodic report and skip the
publication when the configured `cache.ReportingStrategy` does not require it. Strategy semantics —
what `ReportAlways`, `ReportOnChangeOnly` and `ReportAtLeastEvery` each require, `reflect.DeepEqual`
comparison, the never-reported-key rule and the `force` bypass — are specified once in the
`adapter/reporting` capability and SHALL NOT be restated here. This requirement adds only what is
specific to service packages: the cache SHALL be keyed by event name plus a sub key — unit, extended
value name, alarm event or parameter ID, and the empty string when the service has a single value.
When `cfg.ReportingStrategy` is nil, `NewService` SHALL substitute
the package's `DefaultReportingStrategy`, which is `cache.ReportOnChangeOnly()` for every package
except `numericmeter` and `numericsensor` (`cache.ReportAtLeastEvery(30 * time.Minute)`).
`chargepoint` SHALL take two strategies: `DefaultStateReportingStrategy` =
`cache.ReportOnChangeOnly()` for state, cable lock, max current and phase mode, and
`DefaultSessionReportingStrategy` = `cache.ReportAtLeastEvery(30 * time.Minute)` for the current
session report. `parameters` SHALL fix its strategy to `cache.ReportOnChangeOnly()` and accept no
override, deduplicating `evt.param.report` under the parameter ID sub key and `evt.sup_params.report`
under the empty one; `devsys`, `diagnostic` and `ota` SHALL have no reporting cache and SHALL NOT
deduplicate their reports.

#### Scenario: unchanged value is not republished
- **WHEN** `SendBinaryReport(false)` is called twice with the controller returning the same value
  and the default `ReportOnChangeOnly` strategy
- **THEN** the first call publishes `evt.binary.report` and returns `true`
- **AND** the second call publishes nothing and returns `false, nil`

#### Scenario: forced report bypasses the cache
- **WHEN** `SendBinaryReport(true)` is called with an unchanged value
- **THEN** `evt.binary.report` is published and `true` is returned

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

### Requirement: Controller Call Serialisation
Each service instance SHALL hold its own mutex and SHALL hold it for the whole of every
`Send*Report`, `Set*` and other controller-invoking method, so that a controller implementation
never sees two concurrent calls from the same service. Distinct services on the same thing SHALL
NOT share a lock; a controller shared between services must provide its own safety.

#### Scenario: concurrent report and set
- **WHEN** a reporting task calls `SendStateReport(false)` while a routed `cmd.charge.start`
  calls `StartCharging` on the same chargepoint service
- **THEN** the two controller calls are serialised by the service lock

### Requirement: Command Routing
`RouteService(serviceRegistry)` SHALL return one `*router.Routing` per supported command, each
matched by `router.ForType(<command>)` combined with `router.ForService(<service name>)` — or
`router.ForServicePrefix("meter_")` / `router.ForServicePrefix("sensor_")` for the meter and sensor
families. Each handler SHALL resolve the service with `serviceRegistry.ServiceByTopic(message.Topic)`,
SHALL fail when no service is found or the found service does not implement the package's `Service`
interface, and SHALL return an error rather than a reply on failure — the router turns that into an
`evt.error.report` carrying the error text. A handler serving a `cmd.*.set` command SHALL follow a
successful set with a forced report of the affected value, so the hub observes the new state without
waiting for the next reporting cycle.

#### Scenario: set command confirms with a forced report
- **WHEN** `cmd.binary.set` with value `true` is routed to an `out_bin_switch` service
- **THEN** `SetBinaryState(true)` is called on the service
- **AND** `SendBinaryReport(true)` is called afterwards, publishing `evt.binary.report` regardless
  of the reporting strategy

#### Scenario: unknown address produces an error report
- **WHEN** a command arrives on a topic no service is registered under
- **THEN** the handler returns a "service not found under the provided address" error
- **AND** the router publishes `evt.error.report` back to the sender

#### Scenario: get_report with no unit reports every supported unit
- **WHEN** `cmd.sensor.get_report` arrives with a null value or an empty string
- **THEN** a forced report is sent for each unit in the service's `sup_units`
- **AND** a payload naming a single unit reports only that unit

### Requirement: Periodic Reporting Tasks
`TaskReporting(serviceRegistry, frequency, voters...)` SHALL return a `*task.Task` that appends
`adapter.IsRegistryInitialized(serviceRegistry)` to the caller's voters, so no reporting runs before
the service registry is populated. The task handler SHALL iterate the matching services, SHALL skip
any service whose thing reports `ConnStatusDown` via `adapter.ShouldSkipServiceTask`, SHALL call the
service's `Send*Report` with `force = false`, and SHALL log and continue on error rather than
aborting the sweep. The `virtualmeter` package's level polling task SHALL apply the same skip to the
`out_lvl_switch` services it polls; its modes reporting and garbage cleaning SHALL NOT be skipped.

#### Scenario: a disconnected thing is skipped
- **WHEN** the reporting task runs and one thing's `ConnectivityReport().ConnStatus` is
  `ConnStatusDown`
- **THEN** no controller call is made for that thing's services
- **AND** services on connected things are still reported

#### Scenario: the virtual meter does not poll a disconnected thing
- **WHEN** the virtual meter's level polling task runs and the thing of an `out_lvl_switch` is `ConnStatusDown`
- **THEN** the switch's controller is not asked for its level
- **AND** the thing's virtual meter still sends its modes report

#### Scenario: one failing service does not stop the sweep
- **WHEN** a controller returns an error for one service during the reporting task
- **THEN** the error is logged and the task continues with the remaining services

### Requirement: Chargepoint Optional Capabilities
The `chargepoint` package SHALL define the mandatory `Controller` (start/stop charging, current
session report, state report) and the optional `AdjustableMaxCurrentController`,
`AdjustableOfferedCurrentController`, `PhaseModeAwareController`, `AdjustablePhaseModeController`,
`CableLockAwareController` and `AdjustableCableLockController`. Adjustable capabilities SHALL embed
or depend on their "aware" counterpart: `AdjustablePhaseModeController` requires
`PhaseModeAwareController` and a non-empty `sup_phase_modes`, and `AdjustableCableLockController`
requires `CableLockAwareController`. Both max-current and offered-current adjustment SHALL
additionally require the `sup_max_current` property. When cable lock adjustment is supported the
service SHALL advertise `cmd.cable_lock.set`, `cmd.cable_lock.get_report` and
`evt.cable_lock.report`; when only cable-lock awareness is supported it SHALL advertise the latter
two only.

#### Scenario: awareness without adjustment
- **WHEN** a controller implements `CableLockAwareController` but not `AdjustableCableLockController`
- **THEN** `IsCableLockAware()` is true and `SupportsAdjustingCableLock()` is false
- **AND** the specification gains `cmd.cable_lock.get_report` and `evt.cable_lock.report` but not
  `cmd.cable_lock.set`

### Requirement: Chargepoint Command Validation
The chargepoint service SHALL reject a requested current below 6 A or above the `sup_max_current`
property, SHALL reject a phase mode not listed in `sup_phase_modes`, and SHALL reject a charging
mode not listed in `sup_charging_modes`. A charging mode SHALL be lower-cased before the comparison,
and an empty charging mode, or any mode when `sup_charging_modes` is absent, SHALL be normalised to
the empty string and accepted. Supported states SHALL be read from the `sup_states` property.

#### Scenario: current below the floor
- **WHEN** `cmd.max_current.set` requests 4 A
- **THEN** the controller is not called
- **AND** the handler fails with "configured current must be at least 6A"

#### Scenario: unsupported charging mode
- **WHEN** `StartCharging` is called with a mode absent from `sup_charging_modes`
- **THEN** the controller is not called and an "unsupported mode" error listing the supported modes
  is returned

### Requirement: Numeric Meter Family
The `numericmeter` package SHALL serve the whole `meter_*` family from one implementation, taking
the service name and the supported `Unit` list as `Specification(...)` arguments. Units SHALL be the
`Unit` enum (`kWh`, `W`, `A`, `V`, `VA`, `kVAh`, `VAr`, `kVArh`, `Hz`, `power_factor`, `pulse_c`,
`cub_m`, `cub_f`, `gallon`). A requested unit or extended value SHALL be matched case-insensitively
against the advertised set and reported under its advertised spelling; an unmatched one SHALL be
rejected. Export reporting SHALL require both an `ExportReporter` controller and a non-empty
`sup_export_units`; extended reporting SHALL require both an `ExtendedReporter` and a non-empty
`sup_extended_vals`; `cmd.meter.reset` SHALL require a `ResettableReporter`, and a service holding
one SHALL advertise exactly one extra incoming interface for it — `cmd.meter.reset` of value type
`null`, since the command carries no payload. Simple and export reports SHALL be published as float
messages carrying the `unit` and `is_virtual` properties with
storage strategy `aggregate` keyed by unit; the extended report SHALL be published as a float map
with storage strategy `split`. An extended report SHALL be published when a report is required for
at least one of its values, and every value in the published map SHALL then be marked as reported.

#### Scenario: a resettable meter advertises its reset command
- **WHEN** a meter service is built with a reporter implementing `ResettableReporter`
- **THEN** the specification gains an incoming `cmd.meter.reset` interface of value type `null`
- **AND** no export or extended interface is added by the reset capability alone

#### Scenario: extended report is all-or-nothing
- **WHEN** a periodic extended report finds one of five values changed
- **THEN** a single `evt.meter_ext.report` carrying all five values is published
- **AND** all five values are marked as reported

#### Scenario: extended report supersedes per-unit reports in the task
- **WHEN** the reporting task processes a meter for which `SupportsExtendedReport()` is true
- **THEN** only the extended report is sent
- **AND** no per-unit `evt.meter.report` or `evt.meter_export.report` is sent for that meter

### Requirement: Numeric Sensor Family
The `numericsensor` package SHALL serve the whole `sensor_*` family from one implementation, taking
the service name as a `Specification(...)` argument and routing `cmd.sensor.get_report` by the
`sensor_` service-name prefix. Its handler SHALL additionally verify that the resolved service's
name equals the service named in the message payload, so a command addressed to one sensor cannot be
answered by another sensor on the same address. A requested unit SHALL be matched case-insensitively
against `sup_units` and the report published under the advertised spelling in the `unit` property,
with storage strategy `aggregate` keyed by that unit.

#### Scenario: payload service mismatch
- **WHEN** `cmd.sensor.get_report` arrives on a `sensor_temp` topic with a payload naming
  `sensor_humid`
- **THEN** the handler returns an "incorrect service found under the provided address" error and
  publishes no sensor report

#### Scenario: unsupported unit
- **WHEN** `SendSensorReport("kWh", true)` is called on a service whose `sup_units` is `["C"]`
- **THEN** the reporter is not called and an "unit is unsupported" error is returned

### Requirement: Virtual Meter Composition
The `virtualmeter` package SHALL derive electricity consumption for devices that have no physical
meter. A `Manager`, created with `NewManager(db, recalculationPeriod, garbageCleaningPeriod)`, SHALL
own a `Storage` backed by the `database` package under the `virtualManager` domain and the `device`
key, one entry per service topic. `RegisterThing` SHALL, for each group of a thing that contains an
`out_lvl_switch` service, attach a `virtual_meter_elec` service supporting units `W` and modes
`on`/`off`, and attach a `meter_elec` service — reporting `W` and `kWh` and carrying `is_virtual` —
whose `Reporter` is a controller delegating to the manager. The `meter_elec` service SHALL be
attached at registration only when the stored device already has configured modes; otherwise it is
attached when `cmd.meter.add` configures them. `WithAdapter` SHALL supply the adapter separately so
the manager never reaches for it while holding its own lock.

#### Scenario: adding a meter announces the thing
- **WHEN** `cmd.meter.add` arrives with a float map of modes and a `unit` property, both validated
  against `sup_modes` and `sup_units`
- **THEN** the modes and unit are persisted, the `meter_elec` service is added to the thing if
  absent, and a forced inclusion report is published — on every add, not only the first, so a retry
  after a failed publish still reaches the hub
- **AND** when the stored device has never recorded a mode, a forced `out_lvl_switch` level report
  is emitted to seed the level

#### Scenario: energy accrual
- **WHEN** the manager recalculates energy for an active device
- **THEN** the accumulated energy grows by `elapsedHours * modes[currentMode] * level / 1000`
- **AND** the elapsed time counted is capped at twice the recalculation period, so an interruption
  cannot credit an unbounded amount of energy
- **AND** an inactive device — one whose thing reported `ConnStatusDown` — accrues nothing

#### Scenario: orphaned storage entries
- **WHEN** the garbage-cleaning task finds a stored entry with no live `virtual_meter_elec` service
- **THEN** `OrphanedSince` is stamped on the entry and persisted rather than the entry being deleted
- **AND** the entry is deleted only on a later sweep, once `OrphanedSince` is older than the
  configured garbage-cleaning period and the topic is still not registered in this process
- **AND** an entry whose service reappears has `OrphanedSince` cleared

#### Scenario: level events drive the meter
- **WHEN** an `out_lvl_switch` service publishes a level event with a changed value
- **THEN** the virtual meter records mode `off` for level 0 and `on` otherwise, with the level
  normalised against the switch's `max_lvl`
- **AND** an unchanged level event is dropped by the handler's `adapter.WaitForChange()` filter

### Requirement: Alarm System Service
The `alarm` package SHALL implement the `alarm_system` service, publishing `evt.alarm.report` as a
string map of `event` and `status` (`activ` / `deactiv`) with storage strategy `aggregate` keyed by
the event. Supported events SHALL be read from the `sup_events` property and a requested event
matched against them case-insensitively; the report SHALL be republished under the advertised
spelling, and the service SHALL cache a copy rather than the reporter's struct so that a reporter
reusing its return value cannot corrupt the cached snapshot. A `Reporter` returning a nil report
SHALL cause no publication and no error, which is how a device without stateful alarms is
represented. The package SHALL define the fault event names shared with the zigbee car charger —
`GROUNDING_FAULT`, `PEN_FAULT`, `OVER_TEMP_FAULT`, `RCD_TRIP_FAULT`, `OVER_VOLTAGE_FAULT`,
`UNDER_VOLTAGE_FAULT`, `OVER_CURRENT_FAULT`, `METER_COM_FAULT`, `GRIDTYPE_CONFIG_FAULT` and
`CP_OTHER_ERROR` — so cloud adapters render faults identically to zigbee ones.

#### Scenario: ephemeral alarm source
- **WHEN** `SendAlarmReport("OVER_TEMP_FAULT", true)` is called and the reporter returns `nil, nil`
- **THEN** nothing is published and `false, nil` is returned

#### Scenario: reporter spelling is normalised
- **WHEN** the reporter returns a report whose `Event` differs in case from the advertised event
- **THEN** the published payload and the cache key both use the advertised spelling

### Requirement: Setpoint String Map Validation
`thermostat.SetpointFromStringMap` SHALL reject a `temp` that parses to NaN or ±Inf. The water heater
SHALL use the same setpoint type and helpers. Both services SHALL key their setpoint report cache by the
normalized mode.

#### Scenario: NaN setpoint on a thermostat
- **WHEN** `cmd.setpoint.set` carries `temp` "NaN"
- **THEN** the command is rejected and the controller is not called

#### Scenario: setpoint mode sent in another case
- **WHEN** a thermostat or water heater setpoint is set with type "HEAT" and the periodic task then reports "heat" unchanged
- **THEN** no second report is published

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

