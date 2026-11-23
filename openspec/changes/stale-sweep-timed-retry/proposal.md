# A failed stale sweep is retried on a timer

## Why

A failed stale hub node sweep releases its claim "so a later successful sync retries it". An adapter
that reconciles only on device drift has no later sync: edge-adax-adapter calls `SyncThings` once
after start and then only when its devices change (its PR 42, issue 4). When the hub does not answer
the sweep's request within 10 s, for example while Vinculum is still starting after a hub boot, the
stale nodes stay until the process restarts. The sweep outcome is not visible to the adapter, and
syncing on every check would cost it one more cloud call per check.

## What Changes

- A failed automatic sweep is tried again after 5 minutes, and after every further failure, without
  waiting for another sync.
- The retry passes the same initialization check, predicate and claim as a sweep started by a sync.
- The retry is dropped when a `SyncThings` since then fetched its list and did not finish without
  errors: the things then no longer provably match the service, and the next clean sync sweeps as
  before.

## Impact

`adapter/stale.go`, `adapter/sync.go`, spec `adapter/thing-lifecycle`. No API change. Things changed
outside `SyncThings` (`cmd.thing.delete`, a direct `EnsureThings`) are not tracked, as they are not
during the sweep a sync starts.
