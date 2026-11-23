## ADDED Requirements

### Requirement: Failed Stale Sweep Retried On A Timer
A failed automatic sweep SHALL, besides releasing its claim, be tried again after 5 minutes without
waiting for another sync, because an adapter that reconciles only on device drift may never sync
again in the lifetime of the process. The retry SHALL pass the same initialization check, predicate
and claim as a sweep started by a sync, so the two never overlap. A retry that fails SHALL arm the
next one, and at most one retry SHALL be pending at a time.

The retry SHALL sweep only while the things still match the last fetched list: the latest
`SyncThings` whose fetch succeeded finished without errors. Otherwise it SHALL be dropped, and the
next sync without errors sweeps as before.

#### Scenario: the hub answers only after the adapter started
- **WHEN** the sweep's hub request fails and no further sync runs
- **THEN** the sweep is attempted again 5 minutes later, and again after every further failure

#### Scenario: sweeps fail while a retry is already pending
- **WHEN** a sweep started by a sync fails while the retry of an earlier failure is still pending
- **THEN** no second retry is armed

#### Scenario: a later sync did not apply cleanly
- **WHEN** a sweep failed and, before its retry is due, a sync fetched its list and reported an error
- **THEN** the retry sweeps nothing, and the next sync without errors performs the sweep
