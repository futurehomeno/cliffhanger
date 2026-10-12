# Connectivity Checking Specification

## Purpose
An adapter that talks to a third party cloud must keep the hub honest about whether that cloud is
reachable and whether the stored session is still valid, without flapping the UI on every transient
network blip or rate limit. `app.ConnectivityChecker` implements `app.CheckableApp` by running a
caller-supplied probe and mapping its outcome onto the lifecycle auth and connectivity axes, with
backoff, a failure threshold, and teardown-safe cancellation. Every transition it makes re-announces
all things to the hub so node availability follows the app state.

## Requirements

### Requirement: Probe Contract And Error Classification
`NewConnectivityChecker(probe, lifecycle, reporter, cfg)` SHALL accept a `func() error` probe whose
returned error classifies the outcome. The checker SHALL classify with `errors.Is`, so the probe MAY
wrap the sentinels: an error matching `httpclient.ErrUnauthorized` means the session is invalid, an
error matching `httpclient.ErrTooManyRequests` means rate limiting, and any other non-nil error is an
unclassified failure. A probe that returns a bare vendor error on a 401 or a 429 SHALL therefore be
treated as an unclassified failure and take the backoff path instead of the auth-loss or rate-limit
path. `Check()` and `CheckNow()` SHALL always return `nil` — a probe failure is reported through
lifecycle states, never through the returned error.

#### Scenario: wrapped unauthorized
- **WHEN** the probe returns `fmt.Errorf("wrap: %w", httpclient.ErrUnauthorized)`
- **THEN** the auth-loss branch fires and `Check()` still returns `nil`

#### Scenario: unclassified error
- **WHEN** the probe returns a plain `errors.New("probe err")`
- **THEN** the failure is counted and rechecked with backoff, and neither the auth state nor an immediate disconnect is applied

### Requirement: Configuration Defaults
`CheckerConfig` SHALL be normalised once at construction, treating a zero value as "use the default":
`RecheckBackoff` defaults to `backoff.New(time.Minute, 5*time.Minute, 15*time.Minute, 1, 2)`,
`MaxRechecks` defaults to `1`, and `RateLimitDelay` defaults to `5 * time.Minute`. A nil
`AuthLossState` SHALL mean "always `AuthStateLost`" and a nil `Authorized` SHALL mean "always
authorized". `CheckInterval()` SHALL return `cfg.Interval` unchanged, and `app.TaskApp` SHALL
substitute its own 30 minute default when that interval is `0`.

#### Scenario: default recheck delays
- **WHEN** the default backoff is in force
- **THEN** the first consecutive failure is rechecked after 1 minute, the second and third after 5 minutes, and the fourth and later after 15 minutes

#### Scenario: zero interval
- **WHEN** an adapter leaves `CheckerConfig.Interval` at zero
- **THEN** the periodic check task runs every 30 minutes

### Requirement: Successful Probe Restores Running State
A probe returning `nil` SHALL first settle the checker — clearing the failure streak, the pending
recheck timer and the throttled warning memory — and then, only if the `Authorized` gate returns
true, set `AUTH_STATE=AUTHENTICATED` and `CONN_STATE=CONNECTED` and repair the app health. The
`Authorized` gate exists so a `Logout`/`Reset` that cleared credentials mid-probe cannot be undone by
a success landing afterwards.

#### Scenario: successful probe while gate denies
- **WHEN** the probe succeeds but `Authorized()` returns false
- **THEN** no lifecycle state is changed and no connectivity report is sent
- **AND** the failure streak and any pending recheck are still cleared

### Requirement: App Health Repair On Success
`repairAppHealth` SHALL set `APP_HEALTH=RUNNING` and `CONFIG_STATE=CONFIGURED` after a successful,
authorized probe, but ONLY when the app health is not already `RUNNING`. An app already `RUNNING`
SHALL be left untouched, so its config state is never force-overwritten. This repair is required
because the periodic check task is gated by `task.WhenAppIsRunning`: an app stranded in
`NOT_CONFIGURED` by a transient failure during authorization would otherwise never be checked again,
and only a `CheckNow` triggered recheck can rescue it.

#### Scenario: stranded app recovers
- **WHEN** the app is `NOT_CONFIGURED`/`NOT_CONFIGURED` and a probe succeeds
- **THEN** app health becomes `RUNNING` and config state becomes `CONFIGURED`

#### Scenario: running app with non-configured config state
- **WHEN** the app health is already `RUNNING` and a probe succeeds
- **THEN** neither the app health nor the config state is written

### Requirement: Authorization Loss
An error matching `httpclient.ErrUnauthorized` SHALL settle the checker and apply the auth-loss state
together with `CONN_STATE=DISCONNECTED`. The auth state SHALL be `lifecycle.AuthStateLost` by
default, or the value returned by `AuthLossState(current)` when configured — an adapter MAY return
`AuthStateNotAuthenticated` to stay silent for an app that was never authenticated. When the applied
auth state is `AuthStateLost`, both axes SHALL be written with a single
`SetConnAndAuthStateReason(conn, auth, "unauthorized")` call so the auth-loss watcher observes a
consistent bundle in one event and the trigger cannot be evicted by a separate connectivity event.
No recheck SHALL be scheduled for this branch.

#### Scenario: session revoked
- **WHEN** the probe returns an error matching `httpclient.ErrUnauthorized` for an authenticated app
- **THEN** `AUTH_STATE=LOST` and `CONN_STATE=DISCONNECTED` are applied atomically and the emitted auth event carries `reason=unauthorized`

#### Scenario: never authenticated with an override
- **WHEN** `AuthLossState` maps a non-authenticated current state to `AuthStateNotAuthenticated`
- **THEN** the auth state becomes `NOT_AUTHENTICATED` and the connectivity state still becomes `DISCONNECTED`

### Requirement: Rate Limiting Leaves Connectivity Untouched
An error matching `httpclient.ErrTooManyRequests` SHALL NOT change any lifecycle state — a rate limit
is not loss of connectivity — and SHALL NOT emit a connectivity report. The checker SHALL settle
(clearing the failure streak) and schedule a single recheck. The delay SHALL be the `RetryAfter`
carried by an `*httpclient.TooManyRequestsError` when it is greater than zero, and the configured
`RateLimitDelay` otherwise.

#### Scenario: server supplies Retry-After
- **WHEN** the probe returns `&httpclient.TooManyRequestsError{RetryAfter: d}` with `d > 0`
- **THEN** the recheck is scheduled after `d`, even when `d` is shorter than `RateLimitDelay`

#### Scenario: bare rate limit after a healthy probe
- **WHEN** a connected, authenticated app is rate limited
- **THEN** it stays `AUTHENTICATED`/`CONNECTED`, no report is sent, and a recheck fires after `RateLimitDelay`

### Requirement: Failure Threshold Before Disconnection
An unclassified probe error SHALL increment a consecutive failure counter and schedule a recheck
after `RecheckBackoff.Delay(failures)`. `CONN_STATE=DISCONNECTED` SHALL be applied only once the
counter STRICTLY EXCEEDS `MaxRechecks` — with the default of `1`, the second consecutive failure is
the first to report disconnection. That transition SHALL leave the auth state untouched, so a purely
network-level outage never clears an authenticated session. Any settling outcome — success,
authorization loss or a rate limit — SHALL reset the counter to zero.

#### Scenario: single transient failure
- **WHEN** one unclassified probe error occurs with `MaxRechecks = 1`
- **THEN** the connectivity state is unchanged and a recheck is scheduled after 1 minute

#### Scenario: outage then recovery
- **WHEN** failures continue past `MaxRechecks`
- **THEN** `CONN_STATE=DISCONNECTED` is applied while `AUTH_STATE` stays `AUTHENTICATED`
- **AND** the first successful recheck restores `CONNECTED` and clears the streak

### Requirement: Throttled Failure Logging
Repeated identical unclassified errors SHALL be logged at warning level only once per streak, via a
`utils.Throttle` keyed on the error text, so a persistent failure does not fill the hub log. The
throttle SHALL be reset whenever the checker settles, so the same error is logged again after an
intervening definitive outcome.

#### Scenario: persistent identical error
- **WHEN** the same error text is returned by many consecutive rechecks
- **THEN** exactly one warning is emitted for that streak

### Requirement: Connectivity Report On Every Transition
The checker SHALL invoke `ConnectivityReporter.SendConnectivityReport()` after any apply that
actually changed a lifecycle state, and SHALL NOT invoke it when the states already held the target
values. The reporter — satisfied by `adapter.Adapter` — re-announces ALL things in one
`evt.network.all_nodes_report`, so things reachable over a local transport keep reporting their own
connectivity even while the third party API is down. A nil reporter SHALL be accepted, and an error
from the report SHALL be logged and SHALL NOT abort the check or change the applied states.

#### Scenario: repeated failing probes past the threshold
- **WHEN** the connectivity state is already `DISCONNECTED` and another failing recheck exceeds `MaxRechecks`
- **THEN** no additional connectivity report is sent

#### Scenario: reporter fails
- **WHEN** `SendConnectivityReport()` returns an error
- **THEN** the error is logged and the lifecycle transition stands

### Requirement: Periodic Check Deferred To Pending Recheck
`Check()` SHALL skip the probe entirely and return `nil` when a recheck timer is already pending or
when the checker has been cancelled, reading both flags under one lock. Deferring to the pending
recheck keeps its backoff or `Retry-After` delay authoritative, and reading the cancelled flag in the
same critical section prevents a `Cancel` racing a periodic `Check` from being silently overwritten.

#### Scenario: recheck pending
- **WHEN** the periodic task calls `Check()` while a recheck is scheduled
- **THEN** the probe is not called and the scheduled recheck delay is preserved

### Requirement: Immediate Check On Fresh Credentials
`CheckNow()` SHALL stop and clear any pending recheck, clear the cancelled flag, reset the failure
counter to zero, and probe unconditionally. Clearing the timer inside the same `checkMu` section that
runs the probe SHALL prevent a concurrently failing probe from scheduling a new timer in the gap and
turning `CheckNow` into a silent no-op. Resetting the counter SHALL ensure a prior failure streak
cannot make the first probe on fresh credentials trip `MaxRechecks`. `CheckNow` is intended as the
check callback of `app.Authorize`.

#### Scenario: authorization after a failure streak
- **WHEN** one background failure has already been counted and `CheckNow()` then probes and fails once
- **THEN** the connectivity state is unchanged because the streak was reset before the probe

#### Scenario: resuming after cancel
- **WHEN** `CheckNow()` is called after `Cancel()`
- **THEN** the checker probes, and subsequent periodic `Check()` calls resume probing

### Requirement: Cancellation Discards In-Flight Results
`Cancel()` SHALL mark the checker cancelled and settle any pending recheck, and SHALL be used as the
teardown hook of logout and factory reset. A probe that was already running when `Cancel()` landed
SHALL have its result discarded — the checker re-reads the cancelled flag after the probe returns and
applies no lifecycle state and sends no report. The checker's own post-probe cleanup SHALL use
settling rather than cancellation, so the cancelled flag keeps meaning "torn down" and cannot be
cleared by the checker itself.

#### Scenario: logout during a slow probe
- **WHEN** `Cancel()` is called while a probe is blocked and the probe then succeeds
- **THEN** the auth and connectivity states are left as they were and no connectivity report is sent

#### Scenario: cancel before a periodic check
- **WHEN** `Cancel()` lands and the periodic task then calls `Check()`
- **THEN** no probe runs and no state is restored until a `CheckNow()` resumes the checker

### Requirement: Serialized Probes And Single Pending Timer
All probes SHALL be serialized by a dedicated check mutex so a periodic check, a `CheckNow` and a
firing recheck cannot interleave their state changes. `schedule` SHALL be a no-op when a timer is
already pending, so at most ONE recheck is outstanding at a time. The recheck callback SHALL acquire
the check mutex BEFORE clearing the timer — preventing a periodic `Check` from slipping into the gap
and probing back to back — and SHALL compare timer identity so a timer that fired concurrently with a
`Cancel` that replaced or cleared it returns without probing.

#### Scenario: concurrent checks
- **WHEN** many goroutines call `Check`, `CheckNow` and `Cancel` concurrently
- **THEN** probes still run, no two probes overlap, and no deadlock or data race occurs

#### Scenario: cancel racing a firing timer
- **WHEN** `Cancel()` clears the timer just as it fires
- **THEN** the callback observes the identity mismatch and performs no probe
