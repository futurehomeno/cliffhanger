## 1. Fix

- [x] 1.1 Failing test: an error-path state report wrapping `adapter.ErrNotReported` logs nothing, any other error logs at error level
- [x] 1.2 `sendStateReportAfterError` logs only when `adapter.ReportFailed`
- [x] 1.3 Test: a connector reporting connectivity from Connect and Disconnect does not deadlock (fails when either takes `connectorLock`); reword the `connectorLock` comment
- [x] 1.4 Reword the stale sweep comments: retried until it succeeds once per process
- [x] 1.5 Spec `adapter/thing-lifecycle`: any failed sweep releases the claim, as the code and `TestExcludeStaleNodes_FailedExclusionReleasesClaim` do since #276
- [x] 1.6 `go build`, `go vet`, `golangci-lint run`, affected tests
