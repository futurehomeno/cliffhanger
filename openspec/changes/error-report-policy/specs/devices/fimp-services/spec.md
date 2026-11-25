## MODIFIED Requirements

### Requirement: Specification Construction
Each package's `Specification(...)` SHALL return a `*fimptype.Service` whose `Address` is
`/rt:dev/rn:<resourceName>/ad:<resourceAddress>/sv:<serviceName>/ad:<address>`, whose `Enabled` is
`true`, whose `Groups` are the caller's groups, and whose `Interfaces` are the package's mandatory
interfaces — including an outgoing `evt.error.report` of value type `string` for every service that
accepts commands, `virtualmeter` included. `Specification(...)` in `chargepoint`, `numericmeter`,
`outbinswitch`, `outlvlswitch`, `mediaplayer` and `virtualmeter` SHALL additionally accept variadic
`adapter.SpecificationOption` values, applied in order after the base specification is built; the
`With*` helpers those packages export SHALL each set one entry in `Props`.

#### Scenario: options set specification properties
- **WHEN** `chargepoint.Specification(..., WithSupportedMaxCurrent(32), WithChargingModes("normal", "slow"))` is called
- **THEN** the returned specification carries `sup_max_current = 32` and `sup_charging_modes = ["normal", "slow"]`
- **AND** `sup_states` is populated from the mandatory `supportedStates` argument

#### Scenario: virtual meter declares the error report
- **WHEN** `virtualmeter.Specification(...)` is built
- **THEN** its interfaces include an outgoing `evt.error.report` of value type `string`, next to `cmd.meter.add`, `cmd.meter.remove` and `cmd.meter.get_report`

## ADDED Requirements

### Requirement: A Value Outside The Advertised Range Is Rejected
A command value outside the range a service advertises — `min_lvl`/`max_lvl`, `sup_max_current`,
`sup_range`/`sup_ranges`, a parameter's `min`/`max` — SHALL be rejected with the code
`out_of_range` before the controller is called. A value not in the list a service advertises —
`sup_modes`, `sup_setpoints`, `sup_phase_modes`, `sup_charging_modes`, `sup_playback`,
`sup_scenes`, `sup_units`, `sup_events`, a parameter's options — SHALL be rejected with the code
`not_supported`. A controller that finds a value outside a limit of the device the service does not
advertise SHALL reject it the same way. Neither the service nor the controller SHALL clamp, cap or
round such a value into the advertised range or list without rejecting it. Matching a listed value
case-insensitively and passing on its advertised spelling does not change the value and stays
allowed. Two existing behaviours are open points of change `error-report-policy` and stay as they
are until decided: the level clamp of requirement "Level Switch Commands" and the step rounding
before the range check of requirement "Water Heater Setpoint Normalisation".

#### Scenario: max current above the maximum
- **WHEN** `cmd.max_current.set` requests 40 A on a chargepoint whose `sup_max_current` is 32
- **THEN** the controller is not called and the command is answered with an `evt.error.report` whose value is `out_of_range`

#### Scenario: phase mode not in the list
- **WHEN** `cmd.phase_mode.set` carries a mode absent from `sup_phase_modes`
- **THEN** the controller is not called and the command is answered with an `evt.error.report` whose value is `not_supported`

#### Scenario: parameter above its maximum
- **WHEN** `cmd.param.set` carries 120 for an `int` parameter whose `input` widget has `max` 100
- **THEN** the controller's `SetParameter` is not called and the command is answered with an `evt.error.report` whose value is `out_of_range`
