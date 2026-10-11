# Close spec and test coverage gaps

## Why

The main specs were written from the code in one pass; an audit against `develop` found
functionality no requirement describes, a few requirements that no longer match the code, and
behaviour that matters but has no test.

## What Changes

- Spec deltas: ADDED requirements for undocumented behaviour, MODIFIED where a requirement
  contradicted the code. One production fix: `event.Manager.Subscribe` ignores a nil
  filter instead of panicking.
- New tests for the untested behaviour that matters (error paths, dedup/rollback/concurrency
  invariants, spec scenarios without a test).
- Bugs and doubts found on the way are parked in `openspec/issues/`.

## Impact

- Downstream edge adapters: none (docs, tests and the nil-filter guard in `event.Manager.Subscribe`).
