## ADDED Requirements

### Requirement: Task Voter Negation
`task.WhenNot(v)` SHALL return a voter that accepts exactly when `v` rejects. It SHALL evaluate `v`
on every vote, so a negated lifecycle voter follows the current state like the voter it wraps.

#### Scenario: running only while disconnected from the vendor
- **WHEN** a task is guarded by `WhenNot(WhenAppIsConnected(l))` and `CONN_STATE` is `DISCONNECTED`
- **THEN** the handler runs, and it is skipped on the first tick after `CONN_STATE` becomes `CONNECTED`

## MODIFIED Requirements

### Requirement: Run-Once Tasks
A task created with a duration of 0 SHALL be executed exactly once per `Start` of the task manager,
in its own goroutine, and SHALL NOT be rescheduled; a manager stopped and started again SHALL run it
again. Such a task SHALL still be subject to its voters, so a voter rejecting the single execution
means the handler never runs for that start. `Stop` SHALL wait for a run-once handler still in
flight.

#### Scenario: startup task
- **WHEN** the task manager starts with a task whose duration is 0
- **THEN** the handler is invoked once and the goroutine exits

#### Scenario: restarted manager
- **WHEN** the manager is stopped and started again
- **THEN** the run-once handler is invoked a second time

### Requirement: Subscription Registry Semantics
`Subscribe(subID, buffer, filters...)` SHALL create and return a fresh channel with the requested
buffer for each call, even when the ID is already in use, and each such subscription SHALL keep its
own filters. `Unsubscribe(subID)` SHALL close every channel registered under that ID and remove them,
so an ID is only safe to share between subscribers with the same lifetime; unsubscribing an unknown
or already removed ID SHALL be a no-op. The filter slice SHALL be cloned on subscribe so that later
mutation of the caller's slice cannot alter a live subscription. `Bus.Subscribe` SHALL ignore a nil
predicate rather than invoke it.

#### Scenario: two subscribers share an ID
- **WHEN** two components subscribe under the same ID and one of them unsubscribes
- **THEN** both channels are closed and both subscriptions are removed

#### Scenario: subscriber-specific filters
- **WHEN** two subscriptions under the same ID declare different filters
- **THEN** each receives only the values matching its own filters

### Requirement: Event Manager Wait For Event
`WaitFor(timeout, filters...)` SHALL return a channel of capacity 1 that yields the first event
matching all filters, or nil if the timeout expires first; with no filters the first event of any
kind matches. It SHALL subscribe under a generated unique ID with a buffer of 10 before it returns,
so an event published right after the call is observed, and SHALL unsubscribe when it resolves, in
both the matched and the timed-out case.

#### Scenario: no matching event before the deadline
- **WHEN** the timeout expires before any matching event is published
- **THEN** the returned channel yields nil and the temporary subscription is removed

#### Scenario: event published right after the call
- **WHEN** a matching event is published immediately after `WaitFor` returns
- **THEN** the returned channel yields that event
