Code starts only after the policy is accepted. Each task writes the failing test first, runs it and
sees it fail for the expected reason, then writes the code.

## 1. fimpgo

- [ ] 1.1 fimpgo change `error-codes` merged and released with `fimptype.ErrorCodeT`
- [ ] 1.2 `go.mod` takes that fimpgo release
  - For now `go.mod` pins the head of fimpgo branch `spec/error-codes` (pseudo-version); re-pin to the release tag before merge. The pinned head also carries `ErrorCodeRateLimited` (`rate_limited`).

## 2. Router

- [x] 2.1 Failing tests in `router/handler_test.go`: an uncoded error answers `internal` with `msg` `boom` (replaces the assertion at `:69`); a coded error answers its code and sentence; a code wrapped twice is found; an uncoded `*url.Error` loses its URL in `msg`; `ErrNotReported` answers `unavailable` at warning level; a busy lock answers `busy`; the reply's correlation ID is the request's UID
- [x] 2.2 `router.Error`, `router.NewError`; `handleError` finds the code with `errors.As`, builds `msg`, picks the log level by code; `Handle` returns a `busy` error on a held lock
- [x] 2.3 Failing test: exactly one log line per failure, at warning for a coded failure (`unauthorized` included) and at error for `internal`
- [x] 2.4 Failing tests for `router.NewErrorReport`: `corid` is the stored request's UID; `resp_to` wins over the request address; skip storage on a device resource; one log line
- [x] 2.5 `router.NewErrorReport`, sharing the payload builder with `handleError`

## 3. Services

- [x] 3.1 Failing tests per service package, checking the reply's value (the level limit of `outlvlswitch` and the step rounding of `waterheater` stay and keep their tests): unknown address `not_found`; wrong value type `bad_request`; missing capability, unlisted mode, phase mode, charging mode, unit, event, scene, playback `not_supported`; current above `sup_max_current`, setpoint outside `sup_range`, parameter outside `min`/`max` `out_of_range`
- [x] 3.2 Coded errors in `adapter/routing.go` and `adapter/service/*`
  - `mediaplayer` and `colorctrl` advertise no range for volume and colour, so they answer no `out_of_range` (issue 9).
- [x] 3.3 Failing test: `virtualmeter.Specification` declares `out evt.error.report string`
- [x] 3.4 `evt.error.report` in `virtualmeter` `requiredInterfaces`

## 4. App flows

- [x] 4.1 Failing tests: set_tokens and login with no error but a failed auth state fill `error_text`; `IN_PROGRESS` leaves it empty; report field names unchanged
- [x] 4.2 `error_text` from `failedAuthStates` in `HandleCmdAuthLogin` and `HandleCmdAuthSetTokens`

## 5. Release

- [ ] 5.1 `go build ./...`, `go vet ./...`, `golangci-lint run`, `make test`
  - 2026-11-25: build, vet and lint clean; `make test` failed only timing tests in `prime/observer`, `router`, `task` on a loaded machine (issue 2). Repeat on a quiet machine before the release.
- [ ] 5.2 Archive this change, release cliffhanger

## 6. Adapters (one change each, per the adapter plan in `design.md`)

- [x] 6.1 Shared conformance test: a failed command gets one `evt.error.report` with a code, the command's `corid` and no URL in `msg`
  - `suite.ExpectCodedError(topic, service, code)` in `test/suite`: the code, a correlation ID present, no URL in `msg`; each adapter uses it for its failing commands.
- [ ] 6.2 All ten adapters: codes and plain sentences for their main failures; no URL, serial or body in error text; failures logged once
- [ ] 6.3 zaptec, mill, hoiax, tibber, adax, sensibo: manifest declares `evt.error.report`
- [ ] 6.4 mill, hoiax, netatmo, tibber, sonos, sensibo: `error_text` filled when the check after authorizing fails
- [ ] 6.5 zaptec: hand-built reports through the router with `corid`; every write refused by eco mode or scheduled power management answered with its `corid` and code, the log line limited to once per period; current above the maximum rejected; own reports not at debug level
- [ ] 6.6 easee: a refused deferred write answered with the command's `corid`; HTTP failures logged once
- [ ] 6.7 mill, adax: thermostat setpoint out of range rejected, not clamped; sensibo: rounding to whole degrees stays, a setpoint out of range rejected
- [ ] 6.8 hoiax: response body in `msg` bounded
- [ ] 6.9 energy guard, easee, zaptec: the wrong "under 6 A" statement; energy guard's rule for a current above the maximum and its `get_schedule_report` failure
