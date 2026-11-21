## ADDED Requirements

### Requirement: Prospective Build State
When `RebuildChangedThings` builds a prospective thing, the `ThingState` it passes to
`ThingFactory.Create` SHALL be an in-memory stand-in carrying the seed's ID, the live thing's address
and the seed's `Info` (round-tripped through JSON, a no-op when the seed has none). It SHALL report
no persisted state and a zero inclusion checksum, and SHALL silently discard `SetState` and
`SetInclusionChecksum`, so the drift check never touches `adapter.json`. A service topology derived
from persisted per-thing state rather than from `Info` is therefore seen as changed by the drift
check.

#### Scenario: factory reads the seed info during a drift check
- **WHEN** `RebuildChangedThings` calls `Create` for a seed whose `Info` lists an extra capability
- **THEN** `Info` on the passed state yields the seed's new info and `Address` the live address
- **AND** `State` leaves the model untouched and nothing is written to the state file

### Requirement: Stale Hub Node Sweep Once
The adapter SHALL sweep hub nodes it no longer owns, clearing nodes left behind by a state store
that was lost or restored from an older backup. `ExcludeStaleNodes` SHALL ask the hub which devices
it attributes to this adapter and SHALL destroy every address the adapter holds no thing for,
best-effort per address: failures SHALL be joined and the addresses that were excluded SHALL still
be returned. When the hub request itself fails it SHALL exclude nothing and return an error wrapping
`ErrStaleNodeFetch`; a per-address failure SHALL NOT wrap it. Attribution SHALL be read from a
service topic's resource name rather than from `device.FIMP.Adapter`, which carries the service name
and differs for technologies such as zwave. Hub devices with an empty FIMP address SHALL be ignored,
and since several hub devices MAY share one thing address, addresses SHALL be deduplicated.

The automatic sweep SHALL run only from `SyncThings` on an adapter built by `NewAdapter` with a
non-nil MQTT transport, and only when the fetch, the vanished-device exclusion and `EnsureThings` all
succeeded — that is the only moment the adapter's things are known to match the service's, so a node
the sync is about to recreate can never be excluded first. It SHALL additionally require the adapter
to be initialized: until then no thing is registered, so every hub node would be classified stale and
the whole fleet excluded. The predicate set by `WithStaleNodeExclusion` (the sweep is enabled by
default) SHALL be evaluated on every such sync. Neither the initialization check nor a false
predicate SHALL spend the once-per-process claim, so a sync that arrives too early or while the
sweep is disabled defers it, and a runtime configuration flag can enable it later.

Once claimed, the sweep SHALL run asynchronously so it cannot hold up the sync, its failures SHALL be
logged rather than returned, and the Vinculum request SHALL be bounded by a timeout. A sweep whose
hub request failed SHALL release the claim so a later successful sync retries it; any other outcome,
including partial per-address failures, SHALL keep the claim, so the sweep is not repeated for the
lifetime of the process. The claim SHALL also keep two sweeps from overlapping.

#### Scenario: state restored from an older backup
- **WHEN** a sync completes without errors and the hub attributes a node to this adapter that the
  adapter owns no thing for
- **THEN** that address is destroyed and its exclusion announced, asynchronously

#### Scenario: a device the sync is about to recreate
- **WHEN** the device fetch or `EnsureThings` reported an error during the sync
- **THEN** no sweep is triggered, so a node awaiting recreation is not excluded first

#### Scenario: a sync races initialization at startup
- **WHEN** a device sync completes before `InitializeThings` has run, so the adapter reports success
  with no things registered
- **THEN** no sweep is started, and the first sync after initialization still performs one

#### Scenario: a disabled predicate does not spend the once
- **WHEN** the predicate returns false on the first successful sync and true on a later one
- **THEN** the first sync skips the sweep without spending the claim and
  the later sync performs it

#### Scenario: the hub cannot be reached
- **WHEN** the Vinculum request fails or times out
- **THEN** the error is logged, the sync that triggered the sweep is unaffected, and the claim is
  released so the next successful sync sweeps again

#### Scenario: one stale node fails to be excluded
- **WHEN** the hub answers and excluding one of the stale addresses fails
- **THEN** the other addresses are still excluded, the failure is logged, and no later sync sweeps
  again

## MODIFIED Requirements

### Requirement: Seeds From Selection
`SeedsFromSelection` SHALL build one seed per available device whose ID appears in the selection.
The seed function MAY return nil to skip an available item; no seed is produced for it. A `nil`
selection SHALL select every available device; a non-`nil` empty selection SHALL select none.
Duplicate IDs — in the available list or the selection — SHALL yield a single seed, because two
seeds with the same ID would create two things with the second stranded at a second address.

#### Scenario: missing selection key means all devices
- **WHEN** the selection is `nil` (the value produced by unmarshalling a missing JSON key)
- **THEN** a seed is produced for every available device

#### Scenario: explicit empty selection means none
- **WHEN** the selection is a non-`nil` slice of length zero
- **THEN** no seeds are produced, and `EnsureThings` destroys every existing thing

#### Scenario: duplicate device in the available list
- **WHEN** two available items map to the same seed ID
- **THEN** exactly one seed for that ID is returned

#### Scenario: seed function skips an item
- **WHEN** the seed function returns nil for an available item
- **THEN** no seed is produced for it and the remaining items are still seeded

## REMOVED Requirements

### Requirement: Stale Hub Node Sweep
**Reason**: its "once is consumed even when disabled" scenario and the fetch-failure claim were wrong against the code.
**Migration**: replaced by "Stale Hub Node Sweep Once".
