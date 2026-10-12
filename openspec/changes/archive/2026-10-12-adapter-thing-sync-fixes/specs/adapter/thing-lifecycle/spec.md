## MODIFIED Requirements

### Requirement: Presence Reconciliation
`EnsureThings` SHALL make the set of things match the supplied seeds: every state record whose ID
is absent from the seeds is destroyed, and every seed with no state record is created. Seeds whose
ID already has a live thing SHALL be dropped without being touched — presence reconciliation never
re-announces or rebuilds a healthy thing. Seeds sharing an ID SHALL be applied once, the first one
winning, so a duplicate never strands a second thing at a second address. Destruction SHALL be
skipped while the adapter is not yet initialized, so a device sync racing `InitializeThings` cannot
drop persisted records before they have been announced; seeds whose IDs have no stored state SHALL
still be created. The whole pass SHALL run under the adapter write lock inside a single state batch,
and SHALL be best-effort per device with failures joined, so a non-nil error means the pass was
partially applied.

#### Scenario: seed with no record
- **WHEN** `EnsureThings` receives a seed whose ID has no stored state
- **THEN** a state record is created, an address is allocated unless the seed pins a `CustomAddress`,
  and a forced inclusion report is published

#### Scenario: duplicate seeds
- **WHEN** `EnsureThings` receives two seeds with the same ID and no `CustomAddress`
- **THEN** exactly one thing is created for that ID, from the first seed

#### Scenario: record with no seed
- **WHEN** a stored thing's ID is absent from the seeds on an initialized adapter
- **THEN** that thing is destroyed and an exclusion report is published for its address

#### Scenario: record with no seed before initialization
- **WHEN** `EnsureThings` runs while `IsInitialized` is false and a stored thing's ID is absent from
  the seeds — including when the seed set is empty
- **THEN** that record is left for `InitializeThings` and no exclusion is published

#### Scenario: one device fails
- **WHEN** creating one seed's thing fails
- **THEN** the remaining seeds are still processed and the joined error is returned

### Requirement: Generic Sync Flow
`SyncThings` SHALL fetch the available devices outside the adapter lock, build seeds from the
selection, exclude vanished devices, apply `EnsureThings`, and return the applied seeds together
with the IDs it excluded. If the fetch fails nothing SHALL be mutated and the error SHALL be
returned, so a network glitch can never wipe live things. A successful fetch SHALL be treated as
complete: every selected device absent from it is destroyed, including when the response is empty —
adapters must therefore make their client return an error rather than a truncated or empty slice on
a non-2xx, a rate limit or an unparsable body. Until the adapter is initialized that destroy is
deferred by `EnsureThings`, so a racing boot sync cannot wipe records `InitializeThings` has not
announced yet. A seed function returning nil means the item cannot be mapped now: it gets no seed,
and since its ID is unknown no unseeded selected device can be told apart from it, so while any
fetched item maps to nil `SyncThings` SHALL exclude no vanished device, SHALL keep every live thing
of a selected device that has no seed, and SHALL NOT trigger the stale sweep. `SyncThings` SHALL NOT
be atomic: it takes and releases the adapter lock per operation, so adapters that also handle
`cmd.thing.delete` or configuration writes must serialise those against it with a shared
`router.MessageHandlerLocker`.

#### Scenario: fetch fails
- **WHEN** the fetch function returns an error
- **THEN** no thing is created or destroyed and the error is returned with nil seeds

#### Scenario: empty response
- **WHEN** the fetch succeeds and returns no devices on an initialized adapter
- **THEN** every existing thing is destroyed

#### Scenario: empty response before initialization
- **WHEN** the fetch succeeds and returns no devices while `IsInitialized` is false
- **THEN** persisted records are left for `InitializeThings`

#### Scenario: seeds are reusable
- **WHEN** `SyncThings` completes, with or without per-device errors
- **THEN** the applied seeds are returned so the caller can pass them to `RebuildChangedThings`
  without fetching again

#### Scenario: an owned device cannot be mapped
- **WHEN** the seed function returns nil for a fetched, selected device the adapter has a live thing for
- **THEN** that thing is neither destroyed nor rebuilt, and no seed for it is returned

#### Scenario: a device the adapter does not own cannot be mapped
- **WHEN** the seed function returns nil for a fetched, selected device the adapter owns no thing for
- **THEN** no exclusion is published for its ID and it is not returned in `excludedIDs`
