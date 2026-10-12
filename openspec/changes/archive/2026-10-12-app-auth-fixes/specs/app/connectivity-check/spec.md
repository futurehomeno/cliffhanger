## MODIFIED Requirements

### Requirement: Successful Probe Restores Running State
A probe returning `nil` SHALL first settle the checker — clearing the failure streak, the pending
recheck timer and the throttled warning memory — and then, only if the `Authorized` gate returns
true, set `AUTH_STATE=AUTHENTICATED` and `CONN_STATE=CONNECTED` and repair the app health. The
`Authorized` gate exists so a `Logout`/`Reset` that cleared credentials mid-probe cannot be undone by
a success landing afterwards. The restore SHALL re-check cancellation under the lock `Cancel` takes
and hold it across the lifecycle writes, so a `Cancel` landing after the gate discards the success.

#### Scenario: successful probe while gate denies
- **WHEN** the probe succeeds but `Authorized()` returns false
- **THEN** no lifecycle state is changed and no connectivity report is sent
- **AND** the failure streak and any pending recheck are still cleared

#### Scenario: logout lands after the gate
- **WHEN** the probe succeeds, `Authorized()` returns true and a logout calls `Cancel` before the restore
- **THEN** the lifecycle is left as the logout set it and no connectivity report is sent
