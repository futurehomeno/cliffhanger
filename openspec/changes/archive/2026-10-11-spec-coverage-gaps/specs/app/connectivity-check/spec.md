## ADDED Requirements

### Requirement: Cancelled Checker Never Probes Or Re-Arms
The checker SHALL re-read the cancelled flag immediately before calling the probe, so a `Cancel`
landing after a periodic `Check` passed its skip test, or after a recheck timer fired and released
its lock, never reaches the third party API. `schedule` SHALL refuse to arm a recheck timer while
the checker is cancelled, so a failing or rate-limited probe whose result was applied just before a
`Cancel` cannot resume a torn-down checker; only `CheckNow` SHALL resume it.

#### Scenario: cancel before the probe starts
- **WHEN** `Cancel()` lands between the pending-timer check and the probe call
- **THEN** the probe is not called and no state is applied

#### Scenario: schedule after cancel
- **WHEN** a recheck would be scheduled after `Cancel()` has run
- **THEN** no timer is armed and no further probe runs until `CheckNow()`
