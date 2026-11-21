## MODIFIED Requirements

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
logged rather than returned, and the Vinculum request SHALL be bounded by a timeout. A failed
sweep, whether the hub request failed or an address could not be excluded, SHALL release the claim so
a later successful sync retries it. A sweep without errors SHALL keep the claim, so the sweep is not
repeated for the lifetime of the process. The claim SHALL also keep two sweeps from overlapping.

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
- **THEN** the other addresses are still excluded, the failure is logged, and the claim is
  released so the next successful sync sweeps again
