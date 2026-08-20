# App Flows Specification

## Purpose
Every cloud adapter has to implement the same four destructive or promoting operations — reset,
logout, authorization and periodic health checking — and each of them touches the lifecycle, the
persisted configuration, the credential store and the thing set in an order that matters when a step
fails halfway. The `app` flow helpers (`app.Reset`, `app.Logout`, `app.Authorize`, `app.ConfigModel`)
and the app task builders (`app.TaskApp`, `app.TaskInitialization`, `app.TaskCheck`) fix that order
once so an adapter's `Reset()`, `Logout()`, `Authorize()` and `Initialize()` implementations are
reduced to the vendor-specific parts. The invariant they exist to protect is that a partially failed
teardown never leaves the hub believing the adapter is configured over a wiped device set.

## Requirements

### Requirement: Reset Teardown Order
`app.Reset` SHALL run, in order: every supplied teardown hook, `Lifecycle.MarkNotConfigured()`,
`ThingDestroyer.DestroyAllThings()`, and the supplied `resetConfig` function. The lifecycle SHALL be
marked not configured BEFORE the destructive steps, so a failure of either destructive step still
leaves the app in a consistent unconfigured state rather than reporting "configured" over an already
wiped device set that a later boot would restore.

#### Scenario: successful reset
- **WHEN** `Reset(lc, destroyer, resetConfig, teardown)` is called on a running app and every step succeeds
- **THEN** the teardown hook has run, all things have been destroyed and the configuration reset
- **AND** the lifecycle reports `APP_HEALTH=NOT_CONFIGURED`, `CONFIG_STATE=NOT_CONFIGURED`, `CONN_STATE=DISCONNECTED`, `AUTH_STATE=NOT_AUTHENTICATED`

#### Scenario: reset without teardown hooks
- **WHEN** `Reset` is called with no variadic teardown hooks
- **THEN** the flow proceeds directly to marking the app not configured and the destructive steps

### Requirement: Reset Continues Past A Failed Step
`app.Reset` SHALL attempt both destructive steps even when the first one fails: a failure of
`DestroyAllThings` SHALL NOT skip `resetConfig`. The returned error SHALL be the join of all step
errors, each wrapped with its step name (`destroy things: …`, `reset configuration: …`), and SHALL be
nil only when both steps succeeded. A failure SHALL NOT restore the previous lifecycle states.

#### Scenario: thing destruction fails
- **WHEN** `DestroyAllThings` returns an error and `resetConfig` succeeds
- **THEN** `resetConfig` has still been called and a non-nil joined error is returned
- **AND** the lifecycle remains at `APP_HEALTH=NOT_CONFIGURED`

### Requirement: Logout Clears Credentials First
`app.Logout` SHALL call `clearCredentials` first and, on failure, SHALL return the error wrapped as
`clear credentials: …` WITHOUT running any teardown hook and WITHOUT touching the lifecycle, so a
failed clear leaves the previous session intact instead of tearing down connections while the app
still reports the old session. Only after a successful clear SHALL the flow run every teardown hook
and then call `Lifecycle.MarkNotConfigured()`.

#### Scenario: credential clearing fails
- **WHEN** `clearCredentials` returns an error
- **THEN** no teardown hook is invoked and the lifecycle keeps `AUTH_STATE=AUTHENTICATED`
- **AND** the wrapped error is returned to the caller

#### Scenario: successful logout
- **WHEN** `clearCredentials` succeeds
- **THEN** the teardown hooks run and the lifecycle is marked not configured, reporting `AUTH_STATE=NOT_AUTHENTICATED`

### Requirement: Authorize Promotes Only On Confirmed Connectivity
`app.Authorize` SHALL call `persistCredentials` first and abort with `persist credentials: …` on
failure, leaving the lifecycle untouched. It SHALL then call the supplied `check` function; an error
returned by `check` is a hard failure that SHALL be propagated unwrapped and SHALL abort promotion.
The app SHALL be promoted with `Lifecycle.MarkRunning()` if and only if
`Lifecycle.ConnectionState()` equals `CONNECTED` after `check` returns — connectivity failures are
reported through the lifecycle states that `check` itself sets, matching `CheckableApp` semantics,
not through the flow's return value.

#### Scenario: check leaves the app disconnected
- **WHEN** `check` returns nil but has set `CONN_STATE=DISCONNECTED`
- **THEN** `Authorize` returns nil and the app is NOT promoted to `APP_HEALTH=RUNNING`

#### Scenario: check confirms connectivity
- **WHEN** `check` returns nil and has set `CONN_STATE=CONNECTED`
- **THEN** the lifecycle is marked running, reporting `APP_HEALTH=RUNNING` and `CONFIG_STATE=CONFIGURED`

#### Scenario: hard check failure
- **WHEN** `check` returns an error
- **THEN** that error is returned unwrapped and the app health is left unchanged

### Requirement: Teardown Hooks Own Connectivity Checker Cancellation
The flow helpers SHALL NOT call `ConnectivityChecker.Cancel` or `ConnectivityChecker.CheckNow`
themselves; they SHALL expose those points as caller-supplied functions. An adapter running a
`ConnectivityChecker` SHOULD pass `Cancel` as a teardown hook of `Reset` and `Logout` so an in-flight
probe's result is discarded and periodic checks stop, and SHOULD pass `CheckNow` as the `check`
callback of `Authorize` so the fresh credentials are validated immediately rather than waiting for a
pending recheck delay.

#### Scenario: logout while a probe is in flight
- **WHEN** `Logout` runs with `checker.Cancel` as its teardown hook and a probe is in flight
- **THEN** the checker is cancelled after credentials are cleared and the in-flight probe's result is discarded

#### Scenario: re-authorization after a cancel
- **WHEN** `Authorize` runs with `checker.CheckNow` as its `check` callback after a previous `Cancel`
- **THEN** the checker resumes, probes immediately, and the app is promoted only if that probe reports `CONNECTED`

### Requirement: Configuration Model Casting
`app.ConfigModel[T]` SHALL cast the untyped model passed to `App.Configure` to the adapter's
configuration type and SHALL return an error naming both the expected and the received type when the
cast fails. It SHALL return the zero value of `T` alongside that error.

#### Scenario: wrong model type
- **WHEN** `ConfigModel[*cfg]` receives a value that is not a `*cfg`
- **THEN** an error containing `expected configuration model` is returned

### Requirement: App Task Set Derived From Implemented Interfaces
`app.TaskApp` SHALL build the app's task set by capability detection on the supplied `App`: it SHALL
add the initialization tasks only if the app implements `InitializableApp`, and the check task only
if it implements `CheckableApp`. An app implementing neither SHALL yield an empty task set.

#### Scenario: checkable-only app
- **WHEN** `TaskApp` is given an app implementing `CheckableApp` but not `InitializableApp`
- **THEN** exactly one task — the check task — is returned

### Requirement: Initialization Runs Once At Startup And Retries On Startup Error
`app.TaskInitialization` SHALL create TWO tasks sharing one handler: a task with interval `0`, which
the task manager runs exactly once at startup with no voter, and a periodic task gated by
`task.WhenAppEncounteredStartupError`, which therefore runs only while `APP_HEALTH` is
`STARTUP_ERROR`. `app.TaskApp` SHALL use a default initialization interval of **10 minutes**.

#### Scenario: initialization succeeds at startup
- **WHEN** `Initialize()` returns nil on the startup run
- **THEN** the app health is never set to `STARTUP_ERROR` and the retry task's voter rejects every subsequent tick

#### Scenario: retry after a failed startup
- **WHEN** `Initialize()` failed and left `APP_HEALTH=STARTUP_ERROR`
- **THEN** the retry task's voter passes and `Initialize()` is invoked again on the next 10-minute tick

### Requirement: Initialization Failure Marks Startup Error Only
`app.HandleInitialization` SHALL, on an `Initialize()` error, set `APP_HEALTH` to `STARTUP_ERROR` via
`SetAppHealth` with no parameters and log the error together with the retry interval. It SHALL NOT
return the error, SHALL NOT panic, and SHALL NOT touch config, connectivity or auth state. On success
it SHALL leave every lifecycle axis unchanged — promoting the app to running is the adapter's job,
not the initialization task's.

#### Scenario: initialize returns an error
- **WHEN** `Initialize()` fails
- **THEN** `APP_HEALTH` becomes `STARTUP_ERROR` and the task returns normally so the task manager keeps running

#### Scenario: initialize succeeds
- **WHEN** `Initialize()` returns nil
- **THEN** no lifecycle state is written by the initialization handler

### Requirement: Periodic Check Runs Only While The App Is Running
`app.TaskCheck` SHALL create a single periodic task gated by `task.WhenAppIsRunning`, so `Check()` is
never invoked while the app is starting, unconfigured, in startup error or terminating.
`app.TaskApp` SHALL take the interval from `CheckableApp.CheckInterval()` and SHALL substitute the
default of **30 minutes** when that method returns `0`.

#### Scenario: app not running
- **WHEN** the check task ticks while `APP_HEALTH` is `NOT_CONFIGURED`
- **THEN** the voter rejects the run and `Check()` is not called

#### Scenario: zero check interval
- **WHEN** `CheckInterval()` returns `0`
- **THEN** the check task is scheduled with a 30-minute interval

### Requirement: Check Errors Are Logged And Swallowed
`app.HandleCheck` SHALL invoke `CheckableApp.Check()` and, on error, SHALL only log it — it SHALL NOT
modify any lifecycle state and SHALL NOT panic or stop the task. Reporting a connectivity or
authorization outcome is the responsibility of the `Check()` implementation itself, e.g. the
`ConnectivityChecker`.

#### Scenario: check returns an error
- **WHEN** `Check()` returns an error
- **THEN** the error is logged, the handler returns normally, and the next tick calls `Check()` again
