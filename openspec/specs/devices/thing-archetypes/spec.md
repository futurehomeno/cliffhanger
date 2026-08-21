# Device Archetypes Specification

## Purpose
The `adapter/thing` package packages the common device shapes — light, thermostat, boiler, car
charger, sensor, scene controller, battery and main electricity meter — as ready-made compositions
of FIMP services. An adapter author supplies one config struct per thing and gets the correct set of
services, routings and reporting tasks, instead of assembling `adapter/service/*` packages by hand
and risking a thing that advertises a service it cannot route or report. Each archetype is exactly
three functions: a `New*` factory, a `Route*` routing builder and a `Task*` task builder.

## Requirements

### Requirement: Archetype Triplet
Each archetype SHALL expose exactly three exported functions and one config struct:
`New<Archetype>(publisher adapter.Publisher, ts adapter.ThingState, cfg *<Archetype>Config)
adapter.Thing`, `Route<Archetype>(adapter adapter.Adapter) []*router.Routing` and
`Task<Archetype>(adapter adapter.Adapter, reportingInterval time.Duration, reportingVoters
...task.Voter) []*task.Task`. The config struct SHALL carry `ThingConfig *adapter.ThingConfig` plus
one field per composable service, each holding that service package's own `*Config`. `New*` SHALL
construct the services and delegate to `adapter.NewThing`; it SHALL NOT register routings or tasks.

#### Scenario: composing a thing factory
- **WHEN** an adapter's `ThingFactory.Create` calls `thing.NewCarCharger` with a
  `CarChargerConfig` whose `ChargepointConfig` is populated
- **THEN** a `adapter.Thing` exposing a `chargepoint` service is returned
- **AND** nothing is published and no adapter state is mutated by the call itself

### Requirement: Archetype Catalogue
The package SHALL provide the following archetypes, each pre-assembling the listed services:

| archetype | mandatory service | optional services |
| --- | --- | --- |
| `Battery` | `battery` | — |
| `Boiler` | `water_heater` | `sensor_wattemp`, `meter_elec` |
| `CarCharger` | `chargepoint` | `alarm_system`, `diagnostic`, `dev_sys`, `meter_elec`, `ota`, `parameters` |
| `Light` | `out_lvl_switch` | `color_ctrl` |
| `MainElec` | `meter_elec` | — |
| `Scene` | `scene_ctrl` | — |
| `Sensor` | — | `sensor_presence`, any number of `sensor_*` services |
| `Thermostat` | `thermostat` | `sensor_temp`, `meter_elec` |

#### Scenario: a car charger with meter and alarm
- **WHEN** `NewCarCharger` receives a config with `ChargepointConfig`, `MeterElecConfig` and
  `AlarmConfig` set and the remaining optional fields nil
- **THEN** the thing exposes exactly the `chargepoint`, `alarm_system` and `meter_elec` services

### Requirement: Optional Services Are Nil-Gated
An archetype SHALL include an optional service if and only if its config field is non-nil, and SHALL
construct the mandatory service unconditionally. A nil mandatory config is not tolerated — the
archetype dereferences it.

#### Scenario: omitting an optional service
- **WHEN** `NewLight` is called with `ColorCtrlConfig` nil
- **THEN** the thing exposes only the `out_lvl_switch` service
- **AND** the inclusion report advertises no `color_ctrl` service

### Requirement: Sensor Name Gating In Boiler And Thermostat
`NewBoiler` SHALL attach its optional numeric sensor only when the supplied
`SensorWatTempConfig.Specification.Name` equals `sensor_wattemp`, and `NewThermostat` only when
`SensorTempConfig.Specification.Name` equals `sensor_temp`. A config for any other sensor name SHALL
be silently ignored rather than attached, so an archetype cannot advertise a sensor the hub does not
expect from it.

#### Scenario: wrong sensor for the archetype
- **WHEN** `NewThermostat` is given a `SensorTempConfig` whose specification names `sensor_humid`
- **THEN** the thing exposes only the `thermostat` service and any configured `meter_elec`
- **AND** no error is raised

### Requirement: Sensor Archetype Composition
`NewSensor` SHALL take a slice `NumericSensorConfigs []*numericsensor.Config` and attach one
`numericsensor` service per entry, preceded by a `sensor_presence` service when `PresenceConfig` is
non-nil. It SHALL apply no name gating, so any of the `sensor_*` service names may be combined on
one thing.

#### Scenario: multi-sensor device
- **WHEN** `NewSensor` receives configs for `sensor_temp` and `sensor_humid` and a nil
  `PresenceConfig`
- **THEN** the thing exposes both numeric sensor services and no presence service

### Requirement: Main Electricity Meter Defaults To Always Reporting
`NewMainElec` SHALL substitute `cache.ReportAlways()` for a nil `MeterElecConfig.ReportingStrategy`,
overriding the `numericmeter` package default of `cache.ReportAtLeastEvery(30 * time.Minute)`, so
the hub receives every periodic reading of the house meter rather than only changed ones. An
explicitly configured strategy SHALL be left untouched.

#### Scenario: unchanged reading is still published
- **WHEN** `TaskMainElec` runs twice and the reporter returns the same value both times
- **THEN** `evt.meter.report` is published on both runs

### Requirement: Thing Assembly Delegated To The Adapter
`New*` SHALL pass the constructed services to `adapter.NewThing`, which SHALL discard any services
already listed on `cfg.ThingConfig.InclusionReport` and rebuild the list from the specifications of
the services actually constructed. `adapter.NewThing` SHALL default a nil
`ThingConfig.ConnectivityReportingStrategy` to `cache.ReportAtLeastEvery(time.Hour)`.

#### Scenario: inclusion report reflects the constructed services
- **WHEN** an archetype is given a `ThingConfig.InclusionReport` that already lists services
- **THEN** those entries are replaced by the specifications of the services the archetype built
- **AND** the thing's inclusion report advertises exactly the constructed services

### Requirement: Routing And Tasks Are Adapter-Wide
`Route*` and `Task*` SHALL be built once per adapter from the `adapter.Adapter` — which serves as
the service registry — and SHALL NOT be derived from any individual thing. They SHALL therefore
cover every service the archetype can host, including the optional ones no current thing exposes;
handlers resolve their target by topic at message time and tasks iterate the registry, so unused
routings and tasks are inert. An adapter composing several archetypes SHALL combine their outputs
with `router.Combine` and `task.Combine`.

#### Scenario: routing for an absent optional service
- **WHEN** `RouteCarCharger` is registered for an adapter whose things expose no `ota` service
- **THEN** the `cmd.ota_update.start` routing exists but resolves no service
- **AND** a command sent to it produces an `evt.error.report` rather than a panic

#### Scenario: adding an optional service later
- **WHEN** a thing is rebuilt with an `alarm_system` service that earlier things did not have
- **THEN** the already-registered `RouteCarCharger` routings serve it without re-registration

### Requirement: Archetype Reporting Tasks
`Task*` SHALL return the reporting tasks of the services the archetype composes, each built with the
same `reportingInterval` and voters. `TaskCarCharger` SHALL return reporting tasks for
`chargepoint`, `alarm_system` and `meter_elec` only — `diagnostic`, `dev_sys`, `ota` and
`parameters` have no periodic reporting task and are driven by commands or events instead.

#### Scenario: car charger task set
- **WHEN** `TaskCarCharger(adapter, 30*time.Second)` is called
- **THEN** three tasks are returned, all firing every 30 seconds
- **AND** none of them polls a diagnostic, dev_sys, OTA or parameters controller
