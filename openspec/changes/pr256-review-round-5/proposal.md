# PR #256 review, round 5

## Why

- A failed `cmd.charge.start` or `cmd.charge.stop` logs an ERROR when the forced state report fails,
  also when the controller has no state yet (`adapter.ErrNotReported`). The reporting tasks of the
  same service skip that error, so the failed command gets a second, misleading error line.
- A reviewer asked `thing.Connect()` and `thing.Disconnect()` to take `connectorLock`. That would
  deadlock real adapters: edge-easee-adapter's `Disconnect` waits for queued reports, one of which
  calls `thing.SendConnectivityReport`, which takes `connectorLock`. On `main` they also ran outside
  any thing lock. The field comment did not say so.
- The stale sweep comments still said the sweep runs at most once per process, but a failed sweep
  releases its claim and is retried by the next successful sync. Spec `adapter/thing-lifecycle` still
  kept the claim after a failed exclusion, which #276 changed in the code and its test
  (`TestExcludeStaleNodes_FailedExclusionReleasesClaim`).

## What Changes

- The error-path state report of `cmd.charge.start` and `cmd.charge.stop` uses `adapter.ReportFailed`,
  so a state not reported yet is not logged.
- The `connectorLock` comment names what it serialises and why Connect and Disconnect stay outside it;
  a regression test fails if either takes the lock.
- `adapter/stale.go` comments and spec `adapter/thing-lifecycle` match the retry after any failed
  sweep. No code change.

## Impact

`adapter/service/chargepoint` routing, spec `devices/fimp-services`; `adapter/thing.go` and
`adapter/stale.go` comments only, spec `adapter/thing-lifecycle`. No API change.
