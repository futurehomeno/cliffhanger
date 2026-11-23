# Adapter route tests failed once under make test

The first `make test` on branch `fix/stop-unsubscribe-disconnected` failed
`TestRouteAdapter_ThingDelete`, `TestRouteAdapter_ThingDeleteErrorPaths` and
`TestRouteAdapter_NetworkReset` in `adapter`. Three reruns of `./adapter` and a second full
`make test` passed. The change touches only `root.doStop`; the failures look like timing against the
shared local broker. Not investigated further.

## 2026-11-23: also `TestSyncThings_FetchFailureKeepsThings`

On untouched `develop` (5937f52), `go test -race -count=1 ./adapter -run TestSyncThings` failed 2 of
6 runs in `TestSyncThings_FetchFailureKeepsThings/a_failed_fetch_destroys_nothing`: the "never"
expectation saw two `evt.thing.inclusion_report` messages on `rn:test_adapter/ad:1`. Other sessions
were running tests against the same local broker at the time, so messages of another test run on
the same topic are the likely cause. Closes when the tests use a topic or broker of their own.
