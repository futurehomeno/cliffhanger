# Application State Specification

## Purpose
The hub and the Futurehome UI decide what to show a user — a login screen, a configuration wizard, an
error banner or a working integration — from the state an adapter reports. The lifecycle keeps that
state as four independent axes, serves them as the `evt.app.state_report` payload, and lets internal
components wait on transitions. Its central guarantee is that a multi-axis change is published as one
consistent snapshot, never as a torn sequence of partial updates.

## Requirements

### Requirement: Four Independent State Axes
The lifecycle SHALL track exactly four state types, each with a closed set of values:
`APP_HEALTH` is one of `STARTING`, `STARTUP_ERROR`, `NOT_CONFIGURED`, `ERROR`, `RUNNING`,
`TERMINATING`; `CONFIG_STATE` is one of `NOT_CONFIGURED`, `CONFIGURED`, `PART_CONFIGURED`,
`IN_PROGRESS`, `NA`; `AUTH_STATE` is one of `NOT_AUTHENTICATED`, `AUTHENTICATED`, `IN_PROGRESS`,
`ERROR`, `LOST`, `NA`; `CONN_STATE` is one of `CONNECTING`, `CONNECTED`, `DISCONNECTED`, `NA`. The
axes SHALL be independent — no setter SHALL derive one axis from another. `State` SHALL return an
empty value for an unknown state type.

#### Scenario: reading a single axis
- **WHEN** `State(StateTypeConnState)` is called
- **THEN** the current connectivity value is returned without regard to health, config or auth

#### Scenario: applications without auth or connectivity
- **WHEN** an adapter has no vendor account or connection to track
- **THEN** it leaves `AUTH_STATE` and `CONN_STATE` at `NA` and drives only health and config

### Requirement: Initial State And Restart Counting
`New(store)` SHALL initialise the lifecycle to `APP_HEALTH=STARTING`, `CONFIG_STATE=NOT_CONFIGURED`,
`AUTH_STATE=NA`, `CONN_STATE=NA`, record the process start time, and — when a store is supplied —
increment and read the persisted restart count. A failure to increment the count SHALL be logged and
SHALL NOT prevent construction. A nil store SHALL be accepted and leave the count at zero.
`Uptime()` SHALL report whole seconds since construction and `RestartsCount()` the value read at
construction.

#### Scenario: adapter restart
- **WHEN** the adapter process starts with a configuration store present
- **THEN** the persisted restart count is incremented and `RestartsCount()` returns the new value

#### Scenario: store increment fails
- **WHEN** the store cannot increment the restart count
- **THEN** the error is logged and the lifecycle is still constructed in its initial state

### Requirement: Single-Axis Setters Emit Only On Change
`SetAppHealth`, `SetConfigState`, `SetAuthState` and `SetConnState` SHALL each write one axis and emit
one system event carrying that axis's state type and new value. Setting an axis to the value it
already holds SHALL be a no-op that emits no event. `SetAppHealth` SHALL additionally forward
caller-supplied parameters on the emitted event.

#### Scenario: redundant set
- **WHEN** `SetConfigState(CONFIGURED)` is called while the config state is already `CONFIGURED`
- **THEN** no event is emitted to any subscriber

### Requirement: Atomic Multi-Axis Bundles
`SetConnAndAuthState`, `SetConnAndAuthStateReason` and `SetAppState` SHALL apply all of their axes
under a single write lock and SHALL emit EXACTLY ONE event, of type `AUTH_STATE`, carrying the new
auth value. No separate event SHALL be emitted for health, config or connectivity in a bundle, so no
subscriber can observe a torn, half-applied snapshot, and no bundled transition can be lost by having
its trigger event evicted from a slow subscriber's buffer. Consequently a subscriber tracking health,
config or connectivity SHALL re-read the state on any received event rather than filter on the event
type. `SetConnAndAuthStateReason` SHALL attach the reason as the `reason` parameter of the emitted
auth event. A bundle whose values all equal the current ones SHALL emit nothing.

#### Scenario: connectivity and auth change together
- **WHEN** `SetConnAndAuthState(DISCONNECTED, LOST)` is called
- **THEN** both axes are updated and exactly one `AUTH_STATE` event with value `LOST` is emitted
- **AND** any subscriber reading the lifecycle upon that event sees both new values

#### Scenario: reported session loss
- **WHEN** `SetConnAndAuthStateReason` is used
- **THEN** the emitted auth event carries a `reason` parameter explaining why the session was lost

#### Scenario: full bundle
- **WHEN** `SetAppState` changes health, config, connectivity and auth
- **THEN** a single event is emitted rather than up to four

### Requirement: Standard State Bundles
`MarkNotConfigured()` SHALL set exactly `APP_HEALTH=NOT_CONFIGURED`, `CONFIG_STATE=NOT_CONFIGURED`,
`CONN_STATE=DISCONNECTED`, `AUTH_STATE=NOT_AUTHENTICATED`, and SHALL be used on uninstall, factory
reset and logout. `MarkRunning()` SHALL set exactly `APP_HEALTH=RUNNING`, `CONFIG_STATE=CONFIGURED`,
`CONN_STATE=CONNECTED`, `AUTH_STATE=AUTHENTICATED`. Both SHALL inherit the single-event guarantee of
`SetAppState`. Adapters keeping auth or connectivity at `NA` SHOULD set axes individually instead of
using these bundles.

#### Scenario: user logs out
- **WHEN** the logout flow calls `MarkNotConfigured()`
- **THEN** the four axes take exactly the unconfigured tuple and one auth event with value `NOT_AUTHENTICATED` is emitted

### Requirement: State Snapshot Reporting
`AllStates()` SHALL return a consistent snapshot of all four axes, read under a single read lock, as
the object serialised with the keys `app`, `connection`, `config` and `auth`. That object SHALL be
the payload of the `evt.app.state_report` message answering `cmd.app.get_state`, and SHALL be
embedded in the app's configuration and manifest reports and in the discovery response.

#### Scenario: hub asks for the app state
- **WHEN** the adapter receives `cmd.app.get_state`
- **THEN** it replies with `evt.app.state_report` whose object value carries the `app`, `connection`, `config` and `auth` fields

### Requirement: State Change Subscription
`Subscribe(subID, bufSize)` SHALL return a dedicated buffered channel of system events and
`Unsubscribe(subID)` SHALL close every channel registered under that ID, so an ID is only safe to
share between subscribers with the same lifetime. Emission SHALL NOT block the state setter: a
subscriber whose buffer is full SHALL have the event dropped and a warning logged naming the
subscriber, the state type and the state value.

#### Scenario: subscriber not draining its channel
- **WHEN** a state change is emitted while a subscriber's buffer is full
- **THEN** the event is dropped for that subscriber with a warning and the setter returns immediately

### Requirement: Waiting For A Target State
`WaitFor(subID, stateType, targetState)` SHALL block until the given axis holds the target value. It
SHALL subscribe before reading the current state, so a transition happening between the two cannot be
missed, and SHALL return immediately if the axis already holds the target. It SHALL append a
generated unique suffix to the supplied subscription ID so concurrent waiters cannot close each
other's channel, and SHALL unsubscribe on return. On every received event it SHALL re-read the axis
rather than inspect the event, so a target reached by a bundled setter — which emits an event of a
different type — is still observed.

#### Scenario: target already reached
- **WHEN** `WaitFor` is called for a state the axis already holds
- **THEN** it returns without waiting for any event

#### Scenario: target reached through a bundle
- **WHEN** a waiter is blocked on `APP_HEALTH=RUNNING` and `MarkRunning()` is called
- **THEN** the single emitted `AUTH_STATE` event makes the waiter re-read health, find `RUNNING` and return
