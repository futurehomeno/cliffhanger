# EnsureThings strands a live thing when given duplicate seeds

`EnsureThings` (adapter/adapter.go:280, `ensureThings` seed loop at :341) does not dedupe its seeds.
Two seeds with the same ID and no `CustomAddress` call `createThing` twice: the first registers a
thing at address 1, the second overwrites the state record with a freshly allocated address 2 and
registers a second thing. The thing at address 1 stays live and announced but has no state record,
so no later sync or destroy-by-ID can reach it until a restart (and its hub node is never excluded).
`SeedsFromSelection` dedupes, so `SyncThings` is safe; adapters building seeds by hand are not.
Close it by deduplicating seeds by ID in `ensureThings` (first wins, mirroring `SeedsFromSelection`)
with a test, or by documenting the precondition on `EnsureThings`.
