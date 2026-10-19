## MODIFIED Requirements

### Requirement: Successful Probe Restores Running State
A probe returning `nil` SHALL first settle the checker — clearing the failure streak, the pending
recheck timer and the throttled warning memory — and then, only if the `Authorized` gate returns
true, set `AUTH_STATE=AUTHENTICATED` and `CONN_STATE=CONNECTED` and repair the app health. The
`Authorized` gate exists so a `Logout`/`Reset` that cleared credentials mid-probe cannot be undone by
a success landing afterwards. The restore SHALL re-check cancellation under the lock `Cancel` takes
and hold it across the lifecycle writes, so a `Cancel` landing after the gate discards the success;
the connectivity report SHALL be sent after that lock is released.

#### Scenario: successful probe while gate denies
- **WHEN** the probe succeeds but `Authorized()` returns false
- **THEN** no lifecycle state is changed and no connectivity report is sent
- **AND** the failure streak and any pending recheck are still cleared

#### Scenario: logout lands after the gate
- **WHEN** the probe succeeds, `Authorized()` returns true and a logout calls `Cancel` before the restore
- **THEN** the lifecycle is left as the logout set it and no connectivity report is sent

### Requirement: Cancellation Discards In-Flight Results
`Cancel()` SHALL mark the checker cancelled and settle any pending recheck, and SHALL be used as the
teardown hook of logout and factory reset. A probe that was already running when `Cancel()` landed
SHALL have its result discarded — the checker re-reads the cancelled flag after the probe returns,
and again under the `Cancel` lock while writing any outcome's lifecycle states, and applies no
lifecycle state and sends no report. The checker's own post-probe cleanup SHALL use
settling rather than cancellation, so the cancelled flag keeps meaning "torn down" and cannot be
cleared by the checker itself.

#### Scenario: logout during a slow probe
- **WHEN** `Cancel()` is called while a probe is blocked and the probe then succeeds
- **THEN** the auth and connectivity states are left as they were and no connectivity report is sent

#### Scenario: cancel before a periodic check
- **WHEN** `Cancel()` lands and the periodic task then calls `Check()`
- **THEN** no probe runs and no state is restored until a `CheckNow()` resumes the checker
