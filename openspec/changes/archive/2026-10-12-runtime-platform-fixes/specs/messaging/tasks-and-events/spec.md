## MODIFIED Requirements

### Requirement: Task Panic Isolation
A panic raised by a task handler SHALL be recovered and logged with its stack, and SHALL NOT stop the
task's ticker, kill the manager, or affect other tasks. `NewManager` SHALL drop nil tasks and log how
many it dropped, so a nil task never reaches `Start` and the remaining tasks start and stop normally.

#### Scenario: handler panics on one tick
- **WHEN** a periodic task's handler panics
- **THEN** the panic is recovered and logged
- **AND** the same task runs again on the next tick

#### Scenario: nil task in the list
- **WHEN** `NewManager` receives a nil task next to a valid one
- **THEN** `Start` runs the valid task without panicking and `Stop` returns
