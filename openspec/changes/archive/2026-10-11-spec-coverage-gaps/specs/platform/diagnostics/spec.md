## ADDED Requirements

### Requirement: Log Control Command Validation
`cmd.log.set_level` SHALL reject a value that logrus cannot parse with an error reply, leaving the
applied and persisted level unchanged, and on success SHALL reply with the persisted level.
`cmd.log.set_revert_timeout` SHALL reject a value that `time.ParseDuration` cannot parse, and a zero
or negative duration, with an error reply that leaves the persisted timeout unchanged; on success it
SHALL reply with the normalised duration string. No configuration-change event SHALL be published
for a rejected set.

#### Scenario: unknown level
- **WHEN** `cmd.log.set_level` carries `"verbose"`
- **THEN** an error is returned and the level stays what it was

#### Scenario: non-positive revert timeout
- **WHEN** `cmd.log.set_revert_timeout` carries `"0s"` or `"-1h"`
- **THEN** an error is returned and the persisted revert timeout is unchanged

#### Scenario: revert timeout changed during an armed debug window
- **WHEN** `SetRevertTimeout(1h)` is called while a revert deadline is armed
- **THEN** the deadline is recomputed as now plus one hour
