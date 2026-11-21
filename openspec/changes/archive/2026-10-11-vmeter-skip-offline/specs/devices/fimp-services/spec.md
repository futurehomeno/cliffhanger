## MODIFIED Requirements

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
