## MODIFIED Requirements

### Requirement: Initial State And Restart Counting
`New(store)` SHALL initialise the lifecycle to `APP_HEALTH=STARTING`, `CONFIG_STATE=NOT_CONFIGURED`,
`AUTH_STATE=NA`, `CONN_STATE=NA`, record the process start time, and — when a store is supplied —
increment and read the persisted restart count. A failure to increment the count SHALL be logged and
SHALL NOT prevent construction; `RestartsCount()` SHALL then report zero. A nil store SHALL be
accepted and leave the count at zero. `Uptime()` SHALL report whole seconds since construction and
`RestartsCount()` the value read at construction.

#### Scenario: adapter restart
- **WHEN** the adapter process starts with a configuration store present
- **THEN** the persisted restart count is incremented and `RestartsCount()` returns the new value

#### Scenario: store increment fails
- **WHEN** the store cannot increment the restart count
- **THEN** the error is logged, the lifecycle is still constructed in its initial state and
  `RestartsCount()` returns zero
