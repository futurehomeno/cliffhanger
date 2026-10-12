# Adapter thing-sync fixes

## Why

Three open issues in the adapter's thing sync: an all-nodes report with no things carries JSON `null`,
duplicate seeds strand a live thing in `EnsureThings`, and a nil seed makes `SyncThings` destroy or
exclude a selected device the vendor still lists.

## What Changes

- `evt.network.all_nodes_report` carries `[]` when no thing is registered.
- `EnsureThings` dedupes seeds by ID, first wins.
- A nil seed means "cannot map now". The item has no known ID, so it cannot be told apart from a
  vanished device: while any fetched item maps to nil, `SyncThings` excludes no vanished
  device, keeps the live things of unseeded selected devices and returns an error wrapping
  `ErrUnmappedDevice`, so callers do not prune their selection and the stale sweep is skipped.

## Impact

`adapter/adapter.go`, `adapter/sync.go`.
