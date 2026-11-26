## MODIFIED Requirements

### Requirement: Chargepoint Command Validation
The chargepoint service SHALL reject a requested maximum current below 6 A, SHALL reject a requested
session current below 0 A, and SHALL reject either above the `sup_max_current` property. It SHALL
reject a phase mode not listed in `sup_phase_modes`, and SHALL reject a charging mode not listed in
`sup_charging_modes`. A charging mode SHALL be lower-cased before the comparison, and an empty
charging mode, or any mode when `sup_charging_modes` is absent, SHALL be normalised to the empty
string and accepted. Supported states SHALL be read from the `sup_states` property.

#### Scenario: current below the floor
- **WHEN** `cmd.max_current.set` requests 4 A
- **THEN** the controller is not called
- **AND** the handler fails with "configured current must be at least 6A"

#### Scenario: session current below 6 A
- **WHEN** `cmd.current_session.set_current` requests 5 A or 0 A
- **THEN** the controller is called with that current
- **AND** a forced current session report is sent

#### Scenario: negative session current
- **WHEN** `cmd.current_session.set_current` requests −1 A
- **THEN** the controller is not called
- **AND** the handler fails with "configured current must be at least 0A"

#### Scenario: unsupported charging mode
- **WHEN** `StartCharging` is called with a mode absent from `sup_charging_modes`
- **THEN** the controller is not called and an "unsupported mode" error listing the supported modes
  is returned
