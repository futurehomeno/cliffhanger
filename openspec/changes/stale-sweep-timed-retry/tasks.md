## 1. Retry

- [x] 1.1 Failing test: a failed sweep reaches the sweep predicate again after the retry delay
- [x] 1.2 Failing test: the retry is dropped while the latest sync did not finish without errors
- [x] 1.3 Failing test: `SyncThings` records a clean sync, and a failed one after it
- [x] 1.3a Failing test: two failed sweeps arm one retry
- [x] 1.3b Failing test: the retry delays are 1, 5, 5, then 15 minutes
- [x] 1.3c Failing test: a sync that starts applying during the hub request excludes nothing
- [x] 1.4 `excludeStaleNodes` arms the retry; `SyncThings` records whether the things match
- [x] 1.5 `go build`, `go vet`, `golangci-lint run`, affected tests
