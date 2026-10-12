# The refresher's backoff error carries the last refresh error

## Why

While backoff is in effect `cache.Refresher` returns a bare `refresher: backoff is in effect`, so
adapters log the backoff without the real cause (30 of 42 hoiax 2.0.0 beta hubs, e.g.
`E [waterheater] Send state report. err: ... controller: failed to refresh points: refresher: backoff is in effect`).

## What Changes

- New exported sentinel `cache.ErrBackoff`. The backoff error matches it with `errors.Is` and also
  wraps the last refresh error, which appears in the message.

## Impact

- `adapter/cache/refresher.go`; spec `adapter/reporting`. Additive: the message keeps its prefix.
