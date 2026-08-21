# Adapter Network Commands Specification

## Purpose
Defines the adapter-level FIMP surface the hub drives every adapter through: fetching an inclusion
report, deleting a single thing, resetting the whole network, querying one or all nodes for
connectivity, and pinging a device. It also defines the two background tasks every adapter runs —
loading persisted things once at start-up and reporting connectivity periodically — together with
the voters that keep them from running too early.

## Requirements

### Requirement: Adapter Routing Set
`RouteAdapter` SHALL return exactly six routings, handling `cmd.thing.get_inclusion_report`,
`cmd.thing.delete`, `cmd.network.reset`, `cmd.network.get_node`, `cmd.network.get_all_nodes` and
`cmd.ping.send`. Every routing SHALL be voted on both the payload service being equal to the
adapter's resource name and the payload interface being the command in question, so an adapter
never answers another adapter's network commands.

#### Scenario: Command for a different adapter
- **WHEN** a `cmd.network.get_all_nodes` message arrives whose payload service is another adapter's name
- **THEN** none of the adapter's routings handle it

### Requirement: Inclusion Report On Demand
On `cmd.thing.get_inclusion_report` the router SHALL read the thing address from the message's
string payload, resolve the thing, and call `SendInclusionReport(true)`. The forced send SHALL
bypass the persisted checksum, so the hub always receives a report even when nothing changed. The
handler SHALL return no reply of its own; the report reaches the hub as the
`evt.thing.inclusion_report` published on the adapter topic.

#### Scenario: Known address
- **WHEN** `cmd.thing.get_inclusion_report` carries the address of a registered thing whose report is unchanged
- **THEN** an `evt.thing.inclusion_report` is published anyway

#### Scenario: Unknown address
- **WHEN** the address resolves to no registered thing
- **THEN** the handler fails with "thing not found under the provided address" and an `evt.error.report` is returned

### Requirement: Thing Deletion
On `cmd.thing.delete` the router SHALL read the address from the `address` key of the message's
string-map payload, rejecting a payload of the wrong type or an empty address with an error. It
SHALL then resolve the thing's internal ID with `ExchangeAddress` before destroying it, because
destroying drops the record that maps address to ID. Destroying SHALL publish an
`evt.thing.exclusion_report` for the address even when no thing is registered under it.

#### Scenario: Empty address
- **WHEN** the payload's `address` value is an empty string
- **THEN** the handler fails with "provided address is empty" and nothing is destroyed

#### Scenario: Address of a registered thing
- **WHEN** a valid address is supplied
- **THEN** the thing is unregistered, disconnected, removed from the persisted state, and an exclusion report is published

### Requirement: Deselection Before Destruction
When `WithSelection` is configured, the delete handler SHALL remove the device from the user's
selection *before* destroying the thing, and SHALL abort the delete when the removal fails. The
reverse order would resurrect the device: a failed deselect after a successful destroy leaves it
selected and the next sync recreates the thing the user just deleted. When the address has no state
record, the handler SHALL pass the address itself to the remover, covering a hub node left behind by
an older, ID-addressed version of the adapter. `WithSelection` SHALL panic at wiring time when either
the remover or the locker is nil, since `WithExternalLock` with a nil locker silently leaves the
handler unlocked.

#### Scenario: Selection removal fails
- **WHEN** the selection remover returns an error
- **THEN** the handler returns that error and the thing is left intact and retryable

#### Scenario: Nil locker
- **WHEN** `WithSelection` is called with a non-nil remover and a nil locker
- **THEN** it panics rather than returning an option that defeats the serialisation

### Requirement: Delete Serialisation Against Configuration Handlers
The delete handler SHALL be wrapped in `router.WithExternalLock` with the locker supplied through
`WithSelection`, which SHALL be the same locker passed to `app.RouteApp`, so the remover's
read-then-write cannot interleave with `cmd.config.extended_set` rewriting the selection. A handler
that cannot acquire the lock SHALL NOT queue: it SHALL immediately answer with an error reporting
that another operation is already running.

#### Scenario: Concurrent configuration change
- **WHEN** `cmd.thing.delete` arrives while a configuration handler holding the same locker is running
- **THEN** the delete is not executed and an error reply is returned

### Requirement: Network Reset
On `cmd.network.reset` the router SHALL destroy every thing and reply with an
`evt.network.reset_done` null message whose service is the adapter name and whose correlation ID is
the request's UID. `DestroyAllThings` SHALL be best effort per device — errors are joined — but SHALL always
leave the adapter's thing map empty, never half reset. When any destroy fails the handler SHALL
return an error and SHALL NOT send the reset-done reply.

#### Scenario: Successful reset
- **WHEN** `cmd.network.reset` is received
- **THEN** an exclusion report is published for every thing and `evt.network.reset_done` is returned to the sender

#### Scenario: Partial failure
- **WHEN** destroying one of the things fails
- **THEN** the remaining things are still destroyed, the adapter holds no things, and an `evt.error.report` is returned instead of `evt.network.reset_done`

### Requirement: Single Node Query
On `cmd.network.get_node` the router SHALL resolve the thing from the string payload address and
call `SendConnectivityReport(true)`, so the resulting `evt.network.node_report` is published
regardless of the thing's connectivity reporting strategy.

#### Scenario: Node queried while unchanged
- **WHEN** `cmd.network.get_node` names a thing whose connectivity has not changed since its last report
- **THEN** an `evt.network.node_report` is published on the adapter topic anyway

### Requirement: All Nodes Query
On `cmd.network.get_all_nodes` the router SHALL call `Adapter.SendConnectivityReport`, which
publishes a single `evt.network.all_nodes_report` object message on the adapter topic carrying the
array of connectivity reports for all registered things. The reports SHALL be collected without
consulting any reporting strategy or cache, and the handler SHALL return no direct reply.

#### Scenario: No things registered
- **WHEN** `cmd.network.get_all_nodes` is received by an adapter with no registered things
- **THEN** an `evt.network.all_nodes_report` is still published, carrying no connectivity reports

### Requirement: Ping
On `cmd.ping.send` the router SHALL resolve the thing from the string payload address and call
`SendPingReport`, which performs a real ping through the connector and publishes an
`evt.ping.report` carrying the measured delay.

#### Scenario: Ping of a known thing
- **WHEN** `cmd.ping.send` names a registered thing
- **THEN** the connector's `Ping` is executed and an `evt.ping.report` with the round-trip delay is published

### Requirement: Command Error Reporting
Any adapter command handler that returns an error SHALL cause the router to log it and reply with an
`evt.error.report` string message addressed to the request's address with message type `evt`. The
reply SHALL carry the original error text under the `msg` property together with `cmd_topic`,
`cmd_service` and `cmd_type` identifying the failed request. A malformed payload SHALL be reported
the same way as a failed operation.

#### Scenario: Malformed payload
- **WHEN** `cmd.thing.delete` arrives with a payload that is not a string map
- **THEN** an `evt.error.report` is returned whose `msg` property explains that the provided address has an incorrect format

### Requirement: Adapter Tasks
`TaskAdapter(adapter, reportingInterval, voters...)` SHALL return exactly two tasks, both scheduled
at `reportingInterval`: an initialization task and a connectivity-reporting task. The initialization
task SHALL be voted by `task.WhenNot(IsInitialized(adapter))` and the connectivity-reporting task by
the caller-supplied voters plus `IsInitialized(adapter)`. A task SHALL run only when all of its
voters agree.

#### Scenario: Before initialization
- **WHEN** the first tick fires on an adapter that has not yet initialized its things
- **THEN** the initialization task runs and the connectivity-reporting task does not

#### Scenario: After initialization
- **WHEN** a tick fires on an initialized adapter
- **THEN** the initialization task no longer runs and the connectivity-reporting task does

### Requirement: Initialization Task
The initialization task SHALL call `InitializeThings`, which loads every persisted thing record,
builds the thing, publishes its inclusion report and only then registers it, and marks the adapter
initialized. The adapter SHALL be marked initialized only after the whole pass succeeded, so a
failed pass leaves the voter open and the task retries on the next tick. A failure SHALL be logged
and SHALL NOT stop the task manager. A record whose thing is already live — created by an inbound
command that arrived before the task ran — SHALL be skipped rather than rebuilt and re-announced.

#### Scenario: Persisted thing loaded
- **WHEN** the initialization task runs against a state file holding one thing record
- **THEN** the thing is created, its inclusion report is sent, it is registered and connected

#### Scenario: Report fails mid-pass
- **WHEN** the inclusion report for one of the records fails to publish
- **THEN** the adapter is not marked initialized, the failure is logged, and the next tick retries the pass

### Requirement: Connectivity Reporting Task
The connectivity-reporting task SHALL iterate every registered thing and call
`SendConnectivityReport(false)`, letting each thing's reporting strategy decide whether a FIMP
message is actually published. A failure for one thing SHALL be logged with that thing's address and
SHALL NOT abort the iteration over the remaining things.

#### Scenario: One thing fails to report
- **WHEN** publishing the node report for one thing returns an error
- **THEN** the error is logged and the remaining things are still asked to report
