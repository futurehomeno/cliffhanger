# Tasks And Events Specification

## Purpose
Adapters need periodic work (polling a vendor cloud, refreshing tokens, reporting state) that must
only run while the application is in a suitable lifecycle state, and they need a way for components
inside the process to react to each other without going through MQTT. The `task` package provides
voted, interval-driven work; the `event` package provides an in-process typed event bus whose central
guarantee is that publishing never blocks the publisher.

## Requirements

### Requirement: Task Definition And Voting
`task.New(handler, duration, voters...)` SHALL define one unit of periodic work. Before each
execution the task SHALL evaluate all its voters and SHALL run the handler only if every voter
returns true; a task with no voters SHALL always run. Voters SHALL be re-evaluated on every tick, not
once at registration, so a rejected execution SHALL NOT stop the task. `task.Combine` SHALL flatten a
mix of `*Task` values and `[]*Task` slices into one slice in argument order.

#### Scenario: voter rejects an execution
- **WHEN** a periodic task ticks while one of its voters returns false
- **THEN** the handler is not invoked
- **AND** the task remains scheduled and runs on a later tick once the voter accepts

### Requirement: Run-Once Tasks
A task created with a duration of 0 SHALL be executed exactly once, in its own goroutine, when the
task manager starts, and SHALL NOT be rescheduled. Such a task SHALL still be subject to its voters,
so a voter rejecting the single execution means the handler never runs.

#### Scenario: startup task
- **WHEN** the task manager starts with a task whose duration is 0
- **THEN** the handler is invoked once and the goroutine exits

### Requirement: Periodic Tasks Fire Immediately
A task created with a non-zero duration SHALL run once immediately when the manager starts and then
once per tick of a ticker of that duration. The interval SHALL be measured between ticks, not from
the end of the previous handler run. A negative duration is not supported: constructing the ticker
panics, and that panic is logged and re-raised rather than swallowed.

#### Scenario: periodic task startup
- **WHEN** the task manager starts a task with a 30 second interval
- **THEN** the handler runs immediately, then again approximately every 30 seconds until the manager stops

### Requirement: Task Manager Lifecycle
`task.NewManager(tasks...)` SHALL run each task in its own goroutine started by `Start`. `Start` SHALL
return an error when the manager is already running and `Stop` SHALL return an error when it is
already stopped. `Stop` SHALL signal the periodic tasks and block until every task goroutine has
returned, so a handler in flight delays the stop until it finishes. A periodic task goroutine SHALL
observe the stop signal only between runs.

#### Scenario: stop while a handler is running
- **WHEN** `Stop` is called while a task handler is mid-execution
- **THEN** the call blocks until the handler returns and only then reports the manager stopped

#### Scenario: restart
- **WHEN** `Stop` completes and `Start` is called again
- **THEN** the manager starts the tasks again without error

### Requirement: Task Panic Isolation
A panic raised by a task handler SHALL be recovered and logged with its stack, and SHALL NOT stop the
task's ticker, kill the manager, or affect other tasks. The recovery covers the handler only: `Start`
walks the task list in order and reads each task's interval before spawning that task's goroutine, so
a nil task SHALL panic out of `Start` and MUST NOT be registered. Because the walk is sequential, the
goroutines for tasks earlier in the list are already running when it panics, so a failed `Start`
SHALL NOT be assumed to have left the manager idle.

#### Scenario: handler panics on one tick
- **WHEN** a periodic task's handler panics
- **THEN** the panic is recovered and logged
- **AND** the same task runs again on the next tick

### Requirement: Lifecycle-Aware Task Voters
The package SHALL provide voters reading the application lifecycle state:
`WhenAppIsStarting`, `WhenAppIsNotConfigured`, `WhenAppIsRunning`, `WhenAppIsTerminating`,
`WhenAppEncounteredStartupError` and `WhenAppEncounteredError` SHALL compare `APP_HEALTH` against
`STARTING`, `NOT_CONFIGURED`, `RUNNING`, `TERMINATING`, `STARTUP_ERROR` and `ERROR` respectively;
`WhenAppIsConnected` and `WhenAppIsDisconnected` SHALL compare `CONN_STATE` against `CONNECTED` and
`DISCONNECTED`; `WhenAppIsAuthenticated` and `WhenAppIsNotAuthenticated` SHALL compare `AUTH_STATE`
against `AUTHENTICATED` and `NOT_AUTHENTICATED`. A voter comparing one axis SHALL ignore the other
three, so guarding vendor-cloud work against both loss of authorization and loss of connectivity
requires combining the auth and connectivity voters on the same task. Each voter SHALL read the
current state at vote time.

#### Scenario: polling only while running
- **WHEN** a task guarded by `WhenAppIsRunning` ticks while app health is `NOT_CONFIGURED`
- **THEN** the handler is skipped, and it resumes on the first tick after health becomes `RUNNING`

#### Scenario: polling stops when authorization is lost
- **WHEN** a task guarded by `WhenAppIsAuthenticated` ticks while `AUTH_STATE` is `LOST`
- **THEN** the handler is skipped, so no request is made against the vendor cloud with credentials it has already rejected
- **AND** it resumes on the first tick after a successful login sets `AUTH_STATE` to `AUTHENTICATED`

#### Scenario: neither auth voter accepts an intermediate state
- **WHEN** `AUTH_STATE` is `IN_PROGRESS`, `ERROR`, `LOST` or `NA`
- **THEN** both `WhenAppIsAuthenticated` and `WhenAppIsNotAuthenticated` return false

### Requirement: In-Process Event Bus Scope
The event bus SHALL deliver values only between components inside the adapter process. It SHALL NOT
publish to MQTT and SHALL NOT be reachable from the hub; FIMP traffic is the router's concern. The
generic `Bus[T]` SHALL back both the event manager (`Bus[Event]`) and the application lifecycle
(`Bus[SystemEvent]`).

#### Scenario: event publication
- **WHEN** a component publishes an event on the event manager
- **THEN** only in-process subscribers receive it and no MQTT message is produced

### Requirement: Subscription Registry Semantics
`Subscribe(subID, buffer, filters...)` SHALL create and return a fresh channel with the requested
buffer for each call, even when the ID is already in use, and each such subscription SHALL keep its
own filters. `Unsubscribe(subID)` SHALL close every channel registered under that ID and remove them,
so an ID is only safe to share between subscribers with the same lifetime. The filter slice SHALL be
cloned on subscribe so that later mutation of the caller's slice cannot alter a live subscription. A
nil filter SHALL be ignored rather than invoked.

#### Scenario: two subscribers share an ID
- **WHEN** two components subscribe under the same ID and one of them unsubscribes
- **THEN** both channels are closed and both subscriptions are removed

#### Scenario: subscriber-specific filters
- **WHEN** two subscriptions under the same ID declare different filters
- **THEN** each receives only the values matching its own filters

### Requirement: Non-Blocking Publish
`Publish` SHALL deliver a value to every subscription whose filters match, and SHALL NEVER block the
publisher. When a matching subscriber's channel buffer is full, the value SHALL be dropped for that
subscriber and reported to the drop callback, while delivery to the other subscribers continues. A
nil drop callback SHALL be permitted and SHALL drop silently. The event manager SHALL log dropped
events at warning level with the subscriber ID, domain and class. Delivery order across different
subscribers SHALL NOT be guaranteed.

#### Scenario: slow subscriber
- **WHEN** a subscriber has not drained its channel and its buffer is full
- **THEN** the published value is dropped for that subscriber and a warning naming the subscriber is logged
- **AND** `Publish` returns without waiting and other subscribers still receive the value

### Requirement: Bus Callback Re-Entrancy Constraint
Filters and the drop callback SHALL be executed while the bus holds its read lock. A filter or drop
callback therefore SHALL NOT call `Subscribe` or `Unsubscribe`, which take the write lock and would
deadlock; it SHOULD record what happened and act after `Publish` returns.

#### Scenario: unsubscribing from a drop callback
- **WHEN** a drop callback attempts to unsubscribe the reported subscriber
- **THEN** the call deadlocks, so the callback must only report and defer the action

### Requirement: Event Manager Wait For Event
`WaitFor(timeout, filters...)` SHALL return a channel of capacity 1 that yields the first event
matching all filters, or nil if the timeout expires first. It SHALL subscribe under a generated
unique ID with a buffer of 10 and SHALL unsubscribe when it resolves, in both the matched and the
timed-out case.

#### Scenario: no matching event before the deadline
- **WHEN** the timeout expires before any matching event is published
- **THEN** the returned channel yields nil and the temporary subscription is removed

### Requirement: Event Identity And Filters
An `Event` SHALL be identified by its domain and class. `NewWithPayload` SHALL additionally carry an
arbitrary payload exposed through `EventWithPayload`. `WaitForDomain` and `WaitForClass` SHALL match
those fields exactly; `WaitForPayload` SHALL match by deep equality and SHALL reject events that
carry no payload; `WaitFor[T]` SHALL match by type assertion on the event value; `And` and `Or` SHALL
compose filters as conjunction and disjunction.

#### Scenario: payload filter on a payload-less event
- **WHEN** `WaitForPayload` is evaluated against an event created by `event.New`
- **THEN** the filter returns false

### Requirement: Listener Handler Isolation
A `Listener` SHALL subscribe each of its handlers on `Start` — with the handler's own subscription ID,
buffer and filters — and run each handler in its own goroutine. A panic raised by a handler's
processor SHALL be recovered and logged with the event domain and class, and the handler goroutine
SHALL continue consuming subsequent events. `Start` SHALL error when already started and `Stop` when
already stopped. `Stop` SHALL signal the goroutines, wait for them, and only then unsubscribe every
handler. A handler goroutine SHALL also exit when its subscription channel is closed.

#### Scenario: processor panics on one event
- **WHEN** an event processor panics while handling an event
- **THEN** the panic is recovered and logged with the event's domain and class
- **AND** the same handler processes the next event from its channel

#### Scenario: listener stop
- **WHEN** `Stop` is called on a started listener
- **THEN** it blocks until all handler goroutines have returned and then unsubscribes each handler ID
