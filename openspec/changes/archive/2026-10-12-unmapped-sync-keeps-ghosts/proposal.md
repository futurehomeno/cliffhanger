## Why

A sync with an unmapped item pinned only the live things of unseeded selected devices, so a ghost record (state with no live thing) of such a device was destroyed and an exclusion published, removing the device from the hub even though no vanished device may be excluded while an item is unmapped.

## What Changes

- While any item is unmapped, `SyncThings` keeps every thing and every ghost record of a selected device that has no seed.

## Impact

`adapter/sync.go`.
