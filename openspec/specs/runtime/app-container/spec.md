# Application Container Specification

## Purpose
The `root` package is the process container every cliffhanger application is built around. It owns
the MQTT transport, the FIMP message router, the task manager, the application services and the
factory-reset resetters, and it defines the exact order in which they are started and stopped so a
partial failure never leaves the process half-running. It also owns the two cross-cutting behaviours
that must survive any adapter: reporting a lost authorization session, and handling the hub's
factory-reset event.

## Requirements

### Requirement: Builder Validation
`Builder.Build` SHALL refuse to construct an application unless an MQTT transport and the full
service-discovery identity — resource name, resource type, package name, instance ID and version —
have been provided. An edge application built with `NewEdgeAppBuilder` SHALL require a lifecycle
instance; a core application built with `NewCoreAppBuilder` SHALL reject one. `Build` SHALL return
the validation error and a nil application rather than a partially wired container.

#### Scenario: missing MQTT transport
- **WHEN** `Build` is called on a builder that never received `WithMQTT`
- **THEN** it returns an error and no application

#### Scenario: lifecycle by application kind
- **WHEN** `NewEdgeAppBuilder` is built without `WithLifecycle`
- **THEN** `Build` returns an error
- **AND** `NewCoreAppBuilder` built *with* `WithLifecycle` also returns an error

### Requirement: Composed Routing And Subscriptions
The builder SHALL append the service-discovery route and the `pt:j1/mt:cmd/rt:discovery` topic
subscription to whatever `WithRouting`, `WithTopicSubscription` and `WithRouterOptions` supplied, and
SHALL construct the router on `router.DefaultChannelID` with the accumulated options applied. The
gateway factory-reset route and its `pt:j1/mt:evt/rt:ad/rn:gateway/ad:1` subscription SHALL be added
only when at least one resetter was registered through `WithResetter`.

#### Scenario: discovery is always wired
- **WHEN** an application is built with no `WithRouting` calls at all
- **THEN** it still answers `cmd.discovery.request` on `pt:j1/mt:cmd/rt:discovery`

#### Scenario: no resetters, no reset route
- **WHEN** an application is built without `WithResetter`
- **THEN** neither the gateway event topic is subscribed nor the `evt.gateway.factory_reset` route
  is registered

### Requirement: Optional Component Scoping
Telemetry and the lifecycle are optional collaborators, so every ordering, rollback and shutdown
requirement below that names one SHALL be read as applying only when that collaborator was
configured. A container built without telemetry SHALL skip the telemetry step wherever it appears
rather than failing, and one built without a lifecycle — which `Builder.Build` permits only for core
applications — SHALL skip every app-health transition. This scoping SHALL NOT be read as making any
other step conditional.

#### Scenario: core application has no lifecycle
- **WHEN** a core application built without a lifecycle is started and later stopped
- **THEN** no app-health transition is attempted at any step, and the remaining ordering is unchanged

#### Scenario: telemetry was never configured
- **WHEN** an application built without `WithTelemetry` is started
- **THEN** the telemetry step is skipped and startup proceeds to the registered services

### Requirement: Startup Ordering
`Start` SHALL bring the container up in exactly this order: MQTT transport (with a 10 second connect
timeout), telemetry, the registered services in registration order, the message router, the
deduplicated topic subscriptions, the auth-loss watcher, and finally the task manager. Starting an
already running application SHALL be a no-op returning nil. On entry the lifecycle app health SHALL
be set to `STARTING`.

#### Scenario: watcher subscribes before the first task probe
- **WHEN** a `WhenAppIsRunning`-gated task loses authorization on its very first probe immediately
  after `taskManager.Start()`
- **THEN** the auth-loss watcher — subscribed one step earlier — observes the transition and reports
  it exactly once

#### Scenario: duplicate subscriptions collapse
- **WHEN** the same topic is passed twice through `WithTopicSubscription`
- **THEN** `mqtt.Subscribe` is invoked once for that topic

### Requirement: Startup Rollback
A failure at any startup step SHALL undo the steps that already succeeded, in reverse order, before
returning the error. Rollback SHALL be best effort: a failure while undoing one step SHALL be logged
and SHALL NOT abort the remaining undo steps. After a failed start the application SHALL remain
marked as not running, the lifecycle app health SHALL be set to `STARTUP_ERROR`, and buffered
diagnostics SHALL be flushed via `debug.FlushLogs`.

#### Scenario: a service fails to start
- **WHEN** the second registered service returns an error from `Start`
- **THEN** the first service is stopped again, the failing one is not stopped, and telemetry and the
  transport started earlier are torn down — the router and the subscriptions, which come later in the
  start order, were never started and are not touched
- **AND** the lifecycle reports app health `STARTUP_ERROR`

#### Scenario: task manager fails last
- **WHEN** `taskManager.Start()` fails
- **THEN** the auth-loss watcher started just before it is stopped and its stop and done channels are
  cleared, so a retried `Start` does not leak a second watcher goroutine

### Requirement: Telemetry Start Is Non-Fatal
The container SHALL start telemetry immediately after the MQTT transport so its subscription meets a
live transport, and SHALL treat a telemetry start failure as non-fatal: the error is logged and
startup continues.

#### Scenario: telemetry refuses to start
- **WHEN** `telemetry.Start()` returns an error during `Start`
- **THEN** the error is logged and the remaining startup steps still run
- **AND** `Start` returns nil if nothing else fails

### Requirement: Shutdown Ordering
`Stop` SHALL tear the container down in exactly this order: auth-loss watcher, lifecycle app health
set to `TERMINATING`, task manager, topic unsubscriptions, message router, the registered services in
reverse registration order, telemetry, and finally the MQTT transport. Telemetry SHALL be stopped
before the transport it publishes on, so a late cloud config report cannot land after a reset.
Stopping an application that is not running SHALL be a no-op returning nil.

#### Scenario: services stop in reverse
- **WHEN** services A then B were registered and the application is stopped
- **THEN** B is stopped before A

### Requirement: Best-Effort Shutdown
`Stop` SHALL attempt every shutdown step even when an earlier one fails, SHALL return the collected
failures joined with `errors.Join`, and SHALL mark the application as not running regardless. It
SHALL flush buffered diagnostics through a deferred `debug.FlushLogs`, so the log lines leading up to
a failing shutdown reach the log file.

#### Scenario: a failing step still flushes logs
- **WHEN** `taskManager.Stop()` fails because the manager was never started
- **THEN** `Stop` returns a non-nil error
- **AND** the log lines buffered before the failure are present in the configured log file

### Requirement: Factory Reset
`Reset` SHALL stop the application first — logging but not aborting on a stop error — and then invoke
every registered resetter in registration order, returning on the first resetter error. The container
SHALL handle the FIMP event `evt.gateway.factory_reset` received on
`pt:j1/mt:evt/rt:ad/rn:gateway/ad:1` by running `Reset` in a separate goroutine, because the reset
stops the message router that is dispatching the very message. The handler SHALL produce no reply and
SHALL suppress error replies (`router.WithSilentErrors`).

#### Scenario: gateway factory reset event
- **WHEN** `evt.gateway.factory_reset` arrives on the gateway event topic and a resetter is registered
- **THEN** the application is stopped and the resetter's `Reset` is invoked
- **AND** no FIMP reply is published for the event

### Requirement: Run And Wait
`Run` SHALL start the application, install a SIGINT/SIGTERM handler, and block until the application
stops, returning the shutdown error. `Wait` SHALL return nil immediately when the application is not
running, otherwise block on the internal error channel. Delivery of the shutdown result to that
channel SHALL be non-blocking, so a `Stop` with no waiter does not deadlock. `Run` SHALL recover from
a panic in itself or the signal goroutine, emit a panic telemetry event and re-panic.

#### Scenario: signal-driven shutdown
- **WHEN** the process receives SIGTERM while running
- **THEN** a filtered goroutine dump (entries matching mutex, semaphore, panic or lock) is logged at
  warning level
- **AND** `Stop` is invoked and `Run` returns its error

#### Scenario: reboot milestone
- **WHEN** `Run` starts an edge application whose lifecycle restart count is a positive multiple of 500
- **THEN** a reboot-milestone telemetry event carrying that count is emitted

### Requirement: Auth-Loss Watcher Arming
The container SHALL run an auth-loss watcher that subscribes to lifecycle state changes with a buffer
of 5 events, considers only `AUTH_STATE` events, and reports a loss only for a genuinely lost
session. The watcher SHALL seed its armed flag from the auth state read *before* it subscribes, SHALL
arm on `AUTHENTICATED`, SHALL report and disarm on `LOST` while armed, SHALL disarm without reporting
on `NOT_AUTHENTICATED`, and SHALL leave the armed flag unchanged for any other auth state. The
watcher SHALL NOT start when the lifecycle, the MQTT transport or the resource name is absent.

#### Scenario: never-authenticated application stays silent
- **WHEN** the auth state goes `NOT_AUTHENTICATED` and later `LOST` without ever reaching
  `AUTHENTICATED`
- **THEN** no auth-loss report is produced

#### Scenario: repeated loss does not re-fire
- **WHEN** `AUTHENTICATED` is followed by two consecutive `LOST` events
- **THEN** exactly one auth-loss report is produced
- **AND** a further `AUTHENTICATED` re-arms the watcher so the next `LOST` reports again

#### Scenario: already authenticated at boot
- **WHEN** the application starts with the lifecycle already in `AUTHENTICATED`
- **THEN** the watcher arms at subscribe time and reports the next `LOST` without having observed the
  `AUTHENTICATED` transition

### Requirement: Auth-Loss Reporting
On a reported loss the container SHALL publish the FIMP event `evt.app.state_report` carrying all
lifecycle states to `pt:j1/mt:evt/rt:app/rn:<resource name>/ad:1`, SHALL emit a telemetry event in
domain `auth` with event `logged_out` whose data carries the loss reason under the key `cause`, and
SHALL send the push notification event configured through `WithAuthLossNotification` when both a
notifier and an event were provided. The reason SHALL be taken from the lifecycle event's `reason`
parameter. A failure of any of the three SHALL be logged and SHALL NOT prevent the others.

#### Scenario: full report
- **WHEN** an armed watcher observes `LOST` with reason `unauthorized`
- **THEN** `evt.app.state_report` is published on the application's app topic, a telemetry
  `auth`/`logged_out` event with `data.cause = "unauthorized"` is emitted, and the configured
  notification event is sent

### Requirement: Auth-Loss Reporting Gate
The gate supplied through `WithAuthLossReporting` SHALL be evaluated on every loss, so it can follow a
runtime configuration flag. A nil gate SHALL report every loss. When the gate returns false, all
three outputs — app state report, telemetry event and push notification — SHALL be suppressed.

#### Scenario: reporting disabled
- **WHEN** the gate returns false and a loss is observed
- **THEN** no notification is sent, no app state report is published and no telemetry event is emitted
