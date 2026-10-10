## ADDED Requirements

### Requirement: Setpoint String Map Validation
`thermostat.SetpointFromStringMap` SHALL reject a `temp` that parses to NaN or ±Inf. The water heater
SHALL use the same setpoint type and helpers. Its setpoint report cache SHALL be keyed by the
normalized mode.

#### Scenario: NaN setpoint on a thermostat
- **WHEN** `cmd.setpoint.set` carries `temp` "NaN"
- **THEN** the command is rejected and the controller is not called

#### Scenario: setpoint mode sent in another case
- **WHEN** a water heater setpoint is set with type "HEAT" and the periodic task then reports "heat" unchanged
- **THEN** no second report is published
