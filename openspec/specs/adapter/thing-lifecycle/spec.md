# Thing Lifecycle Specification

## Purpose
Defines how an `adapter.Adapter` brings things into existence, keeps them matching the devices a
user selected in a vendor cloud, and removes them again. It covers the `ThingFactory` contract,
seed-driven presence reconciliation (`EnsureThings`), service-topology drift repair
(`RebuildChangedThings`), the generic `SyncThings` flow, and the announcement of inclusion and
exclusion reports to the hub. Every adapter in the fleet depends on these invariants, so a
violation shows up as duplicated, ghosted or silently stale hub nodes.

## Requirements

### Requirement: Thing Factory Contract
`ThingFactory.Create` SHALL be side-effect-free beyond the `Thing` it returns, and SHALL NOT call
back into the `Adapter` it is handed. `RebuildChangedThings` invokes it speculatively on a
prospective thing that is discarded when the topology turns out unchanged, so any effect meant to
outlive the returned value — registering the thing with an external manager, writing storage keyed
by its topic — is never undone. `Create` is always called with the adapter's write lock held on a
non-reentrant mutex; the adapter reference is passed only so it can be handed to the thing being
built.

#### Scenario: prospective build is discarded
- **WHEN** `RebuildChangedThings` calls `Create` and the resulting inclusion report has the same
  topology checksum as the live thing
- **THEN** the returned thing is dropped without being registered, connected or announced
- **AND** any external registration the factory performed is left behind uncorrected

#### Scenario: factory calls back into the adapter
- **WHEN** `Create` calls `ThingByAddress`, `Things`, `ServiceByTopic` or any other adapter method
- **THEN** the adapter deadlocks, because the write lock is already held by the caller

### Requirement: Seeds From Selection
`SeedsFromSelection` SHALL build one seed per available device whose ID appears in the selection.
A `nil` selection SHALL select every available device; a non-`nil` empty selection SHALL select
none. Duplicate IDs — in the available list or the selection — SHALL yield a single seed, because
two seeds with the same ID would create two things with the second stranded at a second address.

#### Scenario: missing selection key means all devices
- **WHEN** the selection is `nil` (the value produced by unmarshalling a missing JSON key)
- **THEN** a seed is produced for every available device

#### Scenario: explicit empty selection means none
- **WHEN** the selection is a non-`nil` slice of length zero
- **THEN** no seeds are produced, and `EnsureThings` destroys every existing thing

#### Scenario: duplicate device in the available list
- **WHEN** two available items map to the same seed ID
- **THEN** exactly one seed for that ID is returned

### Requirement: Thing Creation
`CreateThing` SHALL persist a state record, build the thing through the factory, send a forced
inclusion report, and only then register and connect the thing. A seed without a `CustomAddress`
SHALL be assigned a newly allocated address; a seed with a `CustomAddress` SHALL keep it. If any
step fails the adapter SHALL roll the state record back to the record that existed before the
call — restoring the previous record when there was one, removing the record when there was not —
so a failed creation leaves no ghost.

#### Scenario: creation fails after the state write
- **WHEN** the factory or the inclusion report fails for a seed that had no prior state record
- **THEN** the newly written record is removed from the state
- **AND** the thing is neither registered nor connected

#### Scenario: creation over an existing record fails
- **WHEN** creation fails for a seed whose ID already had a state record
- **THEN** the previous record — with its address and persisted state — is written back

### Requirement: Thing Destruction
Destroying a thing SHALL remove its state record, unregister and disconnect it, and publish an
`evt.thing.exclusion_report` carrying its address. All three steps SHALL be attempted regardless of
earlier failures and their errors joined, so a failed state write never skips the disconnect.
`DestroyThingByID` SHALL be a no-op for an unknown ID, since there is no address to announce.
`DestroyThingByAddress` SHALL publish the exclusion even for an address the adapter holds no record
for, clearing a stale hub node. `DestroyAllThings` SHALL be best-effort per device but SHALL always
leave the adapter's thing map empty.

#### Scenario: unknown ID
- **WHEN** `DestroyThingByID` is called with an ID absent from the state
- **THEN** nothing is published and `nil` is returned

#### Scenario: unknown address
- **WHEN** `DestroyThingByAddress` is called with an address absent from the state
- **THEN** an exclusion report for that address is still published

#### Scenario: state write fails during destroy
- **WHEN** removing the state record fails
- **THEN** the thing is still unregistered and disconnected and the exclusion is still published
- **AND** the record survives, so the next sync retries the destroy or recreates a live thing over it

### Requirement: Thing Initialization
`InitializeThings` SHALL rebuild and announce every thing held in the persistent state exactly
once per process, and SHALL return immediately once the adapter is initialized. It SHALL skip any
address that already has a live thing, because the router runs before the initialization task and
an inbound command can already have created one. Each thing SHALL be announced with
`SendInclusionReport(false)` before being registered, and registered as it goes, so a failure
partway leaves the successfully announced things registered and the failed one unregistered for the
next attempt. The adapter SHALL be marked initialized only after the whole pass succeeds.

#### Scenario: a command created a thing first
- **WHEN** initialization reaches a state record whose address already has a live thing
- **THEN** that record is skipped, the live instance is kept, and no second inclusion report is sent

#### Scenario: initialization fails partway
- **WHEN** the factory fails for the third of five records
- **THEN** the error is returned, `IsInitialized` stays false, and the first two things remain
  registered and announced

### Requirement: Presence Reconciliation
`EnsureThings` SHALL make the set of things match the supplied seeds: every state record whose ID
is absent from the seeds is destroyed, and every seed with no state record is created. Seeds whose
ID already has a live thing SHALL be dropped without being touched — presence reconciliation never
re-announces or rebuilds a healthy thing. The whole pass SHALL run under the adapter write lock
inside a single state batch, and SHALL be best-effort per device with failures joined, so a non-nil
error means the pass was partially applied.

#### Scenario: seed with no record
- **WHEN** `EnsureThings` receives a seed whose ID has no stored state
- **THEN** a state record is created, an address is allocated unless the seed pins a `CustomAddress`,
  and a forced inclusion report is published

#### Scenario: record with no seed
- **WHEN** a stored thing's ID is absent from the seeds
- **THEN** that thing is destroyed and an exclusion report is published for its address

#### Scenario: one device fails
- **WHEN** creating one seed's thing fails
- **THEN** the remaining seeds are still processed and the joined error is returned

### Requirement: Ghost Record Healing
`EnsureThings` SHALL treat a state record that is still seeded but has no live thing as a ghost left
by a partly failed destroy or rebuild, and SHALL recreate it. The heal SHALL preserve the ghost's
address by pinning it as the seed's `CustomAddress`, and SHALL restore the ghost's persisted
per-thing state onto the new record after creation — otherwise the recreated thing would take a
fresh address and start with no state. Healing SHALL be skipped while the adapter is not yet
initialized, where every record legitimately has no live thing.

#### Scenario: ghost is healed
- **WHEN** `EnsureThings` runs on an initialized adapter and finds a seeded record whose address has
  no live thing
- **THEN** a thing is recreated at the ghost's original address, its persisted state is written back,
  and an inclusion report re-announces it

#### Scenario: before initialization
- **WHEN** `EnsureThings` runs while `IsInitialized` is false
- **THEN** existing records are left for `InitializeThings` and no fleet-wide recreation happens

### Requirement: Topology Drift Rebuild
`RebuildChangedThings` SHALL rebuild every already-registered thing whose seed would produce a
different service topology than the live thing, and SHALL ignore seeds for IDs with no state record
or no live thing — presence is `EnsureThings`' job. Drift SHALL be detected by comparing
`topologyChecksum` — a CRC32 over the inclusion report's address, groups and full service
specifications — of a prospective thing built at the live address against the live thing's. The
prospective thing SHALL be built before the destroy, so a factory error costs nothing. The whole
pass SHALL run under the adapter write lock and SHALL be best-effort per seed with errors joined.

#### Scenario: device gains a capability
- **WHEN** a seed's info now yields an extra service and the checksums differ
- **THEN** the thing is destroyed and recreated at the same address with the new service set

#### Scenario: topology unchanged
- **WHEN** the prospective and live checksums match
- **THEN** the live thing is left registered and connected and nothing is published

#### Scenario: seed for an unregistered device
- **WHEN** a seed's ID has no state record, or its address has no live thing
- **THEN** the seed is skipped without calling the factory

### Requirement: Rebuild Preserves Identity And State
A rebuild SHALL reuse the live thing's address as the recreated seed's `CustomAddress`, so the
thing keeps its topic identity, and SHALL capture the persisted per-thing state before the destroy
and write it back after the recreation. If the destroy reports errors the rebuild SHALL log and
recreate anyway, because the subsequent `state.add` overwrites whatever record survived; aborting
would leave the device excluded or ghosted until the next sync and discard the saved state. If the
recreation itself fails the error SHALL be returned and the device stays excluded until the next
restart or sync.

#### Scenario: state survives a rebuild
- **WHEN** a thing with persisted per-thing state is rebuilt
- **THEN** the recreated thing has the same address and the same persisted state

#### Scenario: destroy reports an error
- **WHEN** the exclusion report or the state removal fails during a rebuild
- **THEN** a warning is logged and the recreation still proceeds

### Requirement: Generic Sync Flow
`SyncThings` SHALL fetch the available devices outside the adapter lock, build seeds from the
selection, exclude vanished devices, apply `EnsureThings`, and return the applied seeds together
with the IDs it excluded. If the fetch fails nothing SHALL be mutated and the error SHALL be
returned, so a network glitch can never wipe live things. A successful fetch SHALL be treated as
complete: every selected device absent from it is destroyed, including when the response is empty —
adapters must therefore make their client return an error rather than a truncated or empty slice on
a non-2xx, a rate limit or an unparsable body. `SyncThings` SHALL NOT be atomic: it takes and
releases the adapter lock per operation, so adapters that also handle `cmd.thing.delete` or
configuration writes must serialise those against it with a shared `router.MessageHandlerLocker`.

#### Scenario: fetch fails
- **WHEN** the fetch function returns an error
- **THEN** no thing is created or destroyed and the error is returned with nil seeds

#### Scenario: empty response
- **WHEN** the fetch succeeds and returns no devices
- **THEN** every existing thing is destroyed

#### Scenario: seeds are reusable
- **WHEN** `SyncThings` completes, with or without per-device errors
- **THEN** the applied seeds are returned so the caller can pass them to `RebuildChangedThings`
  without fetching again

### Requirement: Exclusion Of Vanished Devices
Before `EnsureThings` runs, `SyncThings` SHALL publish an exclusion report for every selected ID
that the fetch no longer lists and that the adapter owns no thing for, using the device ID itself as
the report address, and SHALL return the IDs it successfully excluded. Duplicate IDs in the
selection SHALL be announced once. An ID that resolves through `ExchangeID` to a thing the adapter
owns, or through `ExchangeAddress` to another thing's address, SHALL NOT be excluded. The pass SHALL
run before `EnsureThings`, since afterwards every legitimately destroyed device would look orphaned
and be excluded a second time at the wrong address.

#### Scenario: stale hub node left by an older adapter version
- **WHEN** a selected ID is missing from the fetch and neither `ExchangeID` nor `ExchangeAddress`
  resolves it
- **THEN** an exclusion report addressed with that ID is published and the ID is returned in
  `excludedIDs` so the caller can drop it from its persisted selection

#### Scenario: the adapter owns the device
- **WHEN** a selected ID missing from the fetch resolves through `ExchangeID`
- **THEN** no exclusion is published here, and `EnsureThings` destroys it announcing its real address

#### Scenario: the ID collides with another thing's address
- **WHEN** a selected ID missing from the fetch matches another thing's address
- **THEN** no exclusion is published, so the unrelated thing is not killed

### Requirement: Inclusion Report Deduplication
`Thing.SendInclusionReport(false)` SHALL publish `evt.thing.inclusion_report` only when the CRC32 of
the marshalled report differs from the checksum persisted in the thing state, and SHALL return
whether it published. With `force` true it SHALL always publish. After a successful publish the
thing SHALL emit an in-process inclusion-report-sent event and persist the new checksum.
`CreateThing` SHALL force the report; `InitializeThings` SHALL NOT, so a restart does not re-announce
an unchanged fleet.

#### Scenario: unchanged report on restart
- **WHEN** `InitializeThings` rebuilds a thing whose report matches the persisted checksum
- **THEN** no message is published and `false` is returned

#### Scenario: hub asks for the report
- **WHEN** `cmd.thing.get_inclusion_report` is handled
- **THEN** the report is sent with `force` true regardless of the stored checksum

### Requirement: In-Place Service Updates
`Thing.Update` SHALL apply the supplied `ThingUpdate` functions under the thing's write lock and
SHALL NOT publish anything by itself. `ThingUpdateAddService` SHALL add the service and append its
specification to the inclusion report only when no service with the same topic is registered, so it
is idempotent. `ThingUpdateRemoveService` SHALL drop the service from the index and rebuild the
inclusion report's service list without the entry whose address equals the removed service's topic.
The caller SHALL send an inclusion report afterwards for the hub to observe the change.

#### Scenario: adding a service twice
- **WHEN** `ThingUpdateAddService` is applied twice with the same service topic
- **THEN** the service appears once in the index and once in the inclusion report

#### Scenario: update alone does not announce
- **WHEN** services are added or removed via `Update`
- **THEN** nothing is published until `SendInclusionReport` is called

### Requirement: Registry Lookups And Concurrency
The adapter SHALL guard its thing map with a single non-reentrant read-write mutex: lookups
(`Things`, `ThingByAddress`, `ThingByID`, `ThingByTopic`, `ServiceByTopic`, `Services`,
`ExchangeID`, `ExchangeAddress`, `IsInitialized`) take the read lock and every mutating operation
takes the write lock for its whole pass. Registering a thing at an address already occupied by a
different instance SHALL disconnect the displaced instance, so its connector is not leaked.
Unregistering SHALL disconnect the thing. Service lookup by topic SHALL match on the service address
suffix starting at `/rt:dev/rn:`, so an inbound message topic with a `pt:j1/mt:cmd` prefix resolves
to the same service.

#### Scenario: an instance is displaced
- **WHEN** `registerThing` stores a thing at an address held by a different live instance
- **THEN** the previous instance is disconnected before the new one is stored and connected

#### Scenario: topic lookup from an inbound message
- **WHEN** `ServiceByTopic` is given the full MQTT topic of an inbound command
- **THEN** the prefix ahead of `/rt:dev/rn:` is cut and the matching service is returned
