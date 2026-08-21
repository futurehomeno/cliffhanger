# Adapter Addressing And State Specification

## Purpose
Defines the persistent identity layer beneath the thing lifecycle: how a vendor device ID is mapped
to a stable hub address, how that mapping and each thing's private state are stored in
`data/adapter.json`, and what the adapter guarantees across restarts, failed writes and a corrupt
file. Hub nodes, FIMP topics and every report an adapter publishes are keyed on these addresses, so
losing or reassigning one silently orphans a device in the hub.

## Requirements

### Requirement: Adapter Resource Identity
An adapter SHALL be constructed with a FIMP resource name and a resource address, and SHALL use them
as its identity for the lifetime of the process. Every adapter-level message — inclusion reports,
exclusion reports, `evt.network.all_nodes_report` — SHALL be published to the adapter topic built
from resource type `ad`, that resource name and that resource address, with the message service set
to the resource name. Service specification addresses SHALL embed both, in the form
`/rt:dev/rn:<resourceName>/ad:<resourceAddress>/sv:<serviceName>/ad:<thingAddress>`, so a thing's
address is the last segment of every one of its service topics.

#### Scenario: adapter announces a thing
- **WHEN** a thing publishes its inclusion report
- **THEN** the message is published to the adapter's `rt:ad/rn:<name>/ad:<address>` topic rather than
  to any device topic

#### Scenario: service topic derivation
- **WHEN** a thing at address `3` exposes a `chargepoint` service on adapter `test_adapter`/`1`
- **THEN** its topic is `/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:3`

### Requirement: Persistent State Model
The adapter state SHALL consist of a monotonic `address_index` and a map of thing records keyed by
device ID. Each record SHALL carry the device ID, the assigned hub address, the optional seed `info`
blob, the optional per-thing `state` blob and the `inclusion_checksum` of the last published
inclusion report. `info` and `state` SHALL be stored as raw JSON and omitted from the file when
empty.

#### Scenario: record written on creation
- **WHEN** a thing is created from a seed carrying an `Info` value
- **THEN** the record holds the seed ID, the assigned address and the marshalled info, with a zero
  inclusion checksum until the first report is published

### Requirement: Address Allocation
The adapter SHALL allocate a hub address only for a record whose seed carries no `CustomAddress`, by
incrementing `address_index` and using its decimal string as the address — the first allocation
yields `"1"`. A seed with a `CustomAddress` SHALL keep that value and SHALL NOT consume an index.
Allocation SHALL happen inside the same state write as the record itself, so creating a thing costs
one write rather than two. Addresses SHALL NOT be reused: removing a record never decrements the
index.

#### Scenario: first device
- **WHEN** the first thing is created on a fresh adapter from a seed with no `CustomAddress`
- **THEN** it is assigned address `"1"` and `address_index` becomes 1

#### Scenario: device removed and re-added
- **WHEN** a thing at address `"2"` is destroyed and a new device is later created
- **THEN** the new device is assigned the next unused index, not `"2"`

#### Scenario: adapter pins the device ID as the address
- **WHEN** a seed sets `CustomAddress` to the vendor device ID
- **THEN** that ID becomes the hub address and `address_index` is left unchanged

### Requirement: Stable Identifier Exchange
The state SHALL provide a bidirectional mapping between vendor device ID and hub address, exposed as
`Adapter.ExchangeID` (ID to address) and `Adapter.ExchangeAddress` (address to ID), both reporting
absence rather than an error. Lookup by ID SHALL be a map lookup on the record key; lookup by address
SHALL scan the records for a matching address. The mapping SHALL remain stable for as long as the
record exists, including across restarts and across a topology rebuild.

#### Scenario: address resolved for a known device
- **WHEN** `ExchangeID` is called with a device ID that has a state record
- **THEN** the record's address and `true` are returned

#### Scenario: unknown identifier
- **WHEN** `ExchangeID` or `ExchangeAddress` is called with a value no record matches
- **THEN** an empty string and `false` are returned

### Requirement: State File Location And Durability
The adapter state SHALL be persisted as JSON at `<workDir>/data/adapter.json` through
`storage.NewState`, with no defaults file. Each save SHALL create the data directory if needed,
marshal the whole model with tab indentation, rename any existing file to `adapter.json.bak`,
normalise the backup's permissions, then write the file with mode `0644`, truncating, and `fsync` it
before returning. Because every record lives in one file, a save always rewrites the whole model.

#### Scenario: save creates a backup
- **WHEN** the state is saved and a previous `adapter.json` exists
- **THEN** the previous file is renamed to `adapter.json.bak` and the new content is written and
  synced to `adapter.json`

#### Scenario: crash between rename and write
- **WHEN** the process dies after the rename and before the rewrite completes
- **THEN** only `adapter.json.bak` exists, and the next load reads the state from it

### Requirement: Missing Or Corrupt State
`NewState` SHALL load the state eagerly and SHALL return an error if it cannot. A missing state file
SHALL NOT be an error: the adapter starts with an empty model, index zero and no things. An
unparsable `adapter.json` SHALL fall back to `adapter.json.bak` when one exists, logging the failure
and continuing. When neither the file nor the backup can be read, `NewState` SHALL fail rather than
silently start with an empty fleet, since that would orphan every existing hub node.

#### Scenario: first boot
- **WHEN** `NewState` runs and no `adapter.json` or backup exists
- **THEN** it succeeds with an empty state and `InitializeThings` registers no things

#### Scenario: truncated file with a readable backup
- **WHEN** `adapter.json` is unparsable and `adapter.json.bak` parses
- **THEN** the backup's content becomes the loaded state and the failure is logged

#### Scenario: both copies unreadable
- **WHEN** neither `adapter.json` nor its backup can be parsed
- **THEN** `NewState` returns an error and the adapter is not constructed

### Requirement: Batched Writes
`State.batch` SHALL collapse every save made while its function runs into a single write at the end.
The deferral SHALL be lifted and the flush performed even when the function returns an error or
panics, so a partially applied best-effort pass still reaches the disk and a leftover deferral can
never turn later saves into silent no-ops. The deferral SHALL be lifted before the flush, so a save
landing in between writes for itself instead of being dropped. The flush SHALL hold the state write
lock, since it is the only save not already made from inside a lock-holding mutation. A failed flush
SHALL leave the dirty mark set so the next batch retries it.

#### Scenario: a reconciliation pass touching many things
- **WHEN** `EnsureThings` creates and destroys several things inside a batch
- **THEN** `adapter.json` is written and fsynced once, after the pass, not once per thing

#### Scenario: the pass fails partway
- **WHEN** the batched function returns a joined error
- **THEN** the in-memory changes made so far are still flushed to disk

#### Scenario: flush fails
- **WHEN** the single write at the end of a batch fails
- **THEN** the error is joined onto the function's error and the state stays dirty so the next batch
  writes it

### Requirement: Rollback Of Unpersisted Mutations
Outside a batch, every state mutation SHALL restore its in-memory value when the write fails, so
memory never diverges from disk until a restart. `add` SHALL restore the previous record — or delete
the new key — and restore `address_index`. `remove` SHALL put the deleted record back, so the next
sync retries the destroy instead of resurrecting the thing after a restart.
`SetInclusionChecksum` SHALL restore the previous checksum, since the skip-if-unchanged shortcut is
keyed on the in-memory value and would otherwise make every retry a no-op. Inside a batch these
rollbacks cannot fire, because a save only fails at the flush; the resulting records are ghosts that
`EnsureThings` heals.

#### Scenario: failed write while adding a record
- **WHEN** persisting a new record fails
- **THEN** the record is removed from the in-memory model and `address_index` is restored to its
  pre-call value

#### Scenario: failed write while removing a record
- **WHEN** persisting a removal fails
- **THEN** the record stays in the in-memory model and the destroy is retried by the next sync

### Requirement: Per-Thing Info And State
`ThingState.Info` SHALL unmarshal the seed information stored at creation into a caller-supplied
model and SHALL be a no-op returning `nil` when the record carries none; it is written only by thing
creation, never by the thing itself. `ThingState.State` and `SetState` SHALL give a thing a private
JSON blob within the same record, `SetState` persisting immediately (or into the enclosing batch).
Destroying a thing SHALL drop the whole record including this blob, so callers that recreate a thing
in place — ghost healing and topology rebuild — must capture and restore it themselves.

#### Scenario: reading info a thing was created with
- **WHEN** a factory calls `Info` on a state record created from a seed with no `Info`
- **THEN** the supplied model is left untouched and no error is returned

#### Scenario: state survives across restarts
- **WHEN** a thing calls `SetState` and the process is restarted
- **THEN** `InitializeThings` rebuilds the thing and `State` yields the same blob

### Requirement: Inclusion Checksum Persistence
The inclusion report checksum SHALL be stored per thing record and SHALL be updated only after the
report has been published. `SetInclusionChecksum` SHALL skip the write when the stored value already
equals the new one. Because the checksum is part of the record, destroying a thing clears it, and the
next creation of that ID publishes its report unconditionally.

#### Scenario: repeated identical report
- **WHEN** `SetInclusionChecksum` is called with the value already stored
- **THEN** no state write occurs and `nil` is returned

#### Scenario: restart with an unchanged fleet
- **WHEN** the adapter restarts and rebuilds things whose reports still match the stored checksums
- **THEN** no inclusion reports are published

### Requirement: Restart Continuity
A restart SHALL preserve, for every device, its hub address, its seed info, its private state blob
and its last inclusion checksum, plus the adapter-wide `address_index`. Nothing else SHALL survive:
the live thing objects, their services, connector state and connectivity reporting caches are
rebuilt by the factory from the record during `InitializeThings`.

#### Scenario: fleet after a restart
- **WHEN** the process restarts with an intact `adapter.json`
- **THEN** each thing is rebuilt at the address it had before, and `ExchangeID` returns the same
  mapping as before the restart

#### Scenario: capability change across versions
- **WHEN** an adapter upgrade makes the factory build a different service set for a stored record
- **THEN** the thing keeps its address and state, and the changed report is published because its
  checksum no longer matches the stored one
