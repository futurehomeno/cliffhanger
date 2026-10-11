# A nil seed makes SyncThings destroy a selected, still-available device

`SeedsFromSelection` (adapter/seeds.go:14) skips items whose seed function returns nil. `SyncThings`
(adapter/sync.go:44) then treats a selected ID with no seed as vanished: `excludeVanishedDevices`
publishes an exclusion at the ID when the adapter does not own it, and `EnsureThings` destroys the
thing when it does. So a seed function that returns nil for a device it temporarily cannot map
(missing field, unknown model) deletes a device the vendor still lists and the user selected.
Unclear whether that is intended ("nil = unsupported, remove it") or an accident. Close it by
documenting the contract on `SeedsFromSelection`/`SyncThings` (and adding it to the thing-lifecycle
spec), or by excluding skipped-but-available IDs from the vanished/destroy passes.
