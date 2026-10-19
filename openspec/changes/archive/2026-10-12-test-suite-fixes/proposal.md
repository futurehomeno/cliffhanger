# test/suite fixes

## Why

Four issues in `test/suite` and `test/helper/adapter`: a busy-spinning assertion loop, a data race on a shared expectation, an unspecified upper bound for `ExactlyOnce`/`AtMostOnce`, and a teardown callback that drifts on a second run.

## What Changes

- `Router.AssertExpectations` polls on a short ticker instead of spinning.
- `processMessage` evaluates `PublishFn`/`ReplyFn` into locals instead of writing the shared `Expectation`.
- Spec: upper-bounded occurrences are checked only until the node's expectations are satisfied; `Never` asserts absence over the timeout. Behaviour unchanged.
- `TearDownAdapter`'s callback no longer mutates its captured path, so it can run any number of times.

## Impact

`test/suite/router.go`, `test/helper/adapter/helpers.go`, spec `support/test-suite`.
