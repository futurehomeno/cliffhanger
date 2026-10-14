## MODIFIED Requirements

### Requirement: Generic Sync Flow
`SyncThings` SHALL fetch the available devices outside the adapter lock, build seeds from the
selection, exclude vanished devices, apply `EnsureThings`, and return the applied seeds together
with the IDs it excluded. If the fetch fails nothing SHALL be mutated and the error SHALL be
returned, so a network glitch can never wipe live things. A successful fetch SHALL be treated as
complete: every selected device absent from it is destroyed, including when the response is empty
unless an item maps to a nil seed —
adapters must therefore make their client return an error rather than a truncated or empty slice on
a non-2xx, a rate limit or an unparsable body. Until the adapter is initialized that destroy is
deferred by `EnsureThings`, so a racing boot sync cannot wipe records `InitializeThings` has not
announced yet. A seed function returning nil means the item cannot be mapped now: it gets no seed,
and since its ID is unknown no unseeded selected device can be told apart from it, so while any
fetched item maps to nil `SyncThings` SHALL exclude no vanished device, SHALL keep every live thing
and ghost record of a selected device that has no seed, and SHALL return an error wrapping `ErrUnmappedDevice`, so
callers do not prune their selection and the stale sweep is skipped. Devices the adapter never
supports must be filtered out in the fetch, not by a nil seed, or no sync would clean up again.
`SyncThings` SHALL NOT
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
- **THEN** that thing is neither destroyed nor rebuilt, no seed for it is returned, and the error wraps
  `ErrUnmappedDevice`

#### Scenario: a device the adapter does not own cannot be mapped
- **WHEN** the seed function returns nil for a fetched, selected device the adapter owns no thing for
- **THEN** no exclusion is published for its ID and it is not returned in `excludedIDs`

#### Scenario: a ghost of an owned device cannot be mapped
- **WHEN** the seed function returns nil for a fetched item while a selected device without a seed has
  a state record but no live thing
- **THEN** its record is kept and no exclusion is published for it
