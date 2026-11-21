## MODIFIED Requirements

### Requirement: Setpoint String Map Validation
`thermostat.SetpointFromStringMap` SHALL reject a `temp` that parses to NaN or ±Inf. The water heater
SHALL use the same setpoint type. Both services SHALL key their setpoint report cache by the
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

### Requirement: Water Heater Setpoint Value Type
The `waterheater` package SHALL read `cmd.setpoint.set` and publish `evt.setpoint.report` with value
type `object` carrying `type`, `temp` (a number) and `unit`, as the FIMP `water_heater` specification
defines, and SHALL advertise `object` for both interfaces.

#### Scenario: setpoint set as an object
- **WHEN** `cmd.setpoint.set` carries the object `{"type": "heat", "temp": 65, "unit": "C"}`
- **THEN** the controller receives 65 for `heat` and `evt.setpoint.report` is published as an object

#### Scenario: setpoint set as a string map
- **WHEN** `cmd.setpoint.set` carries a `str_map`
- **THEN** the command is rejected and the controller is not called
