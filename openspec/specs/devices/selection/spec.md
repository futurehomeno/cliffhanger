# Device Selection Specification

## Purpose
A vendor account usually exposes more devices than the user wants on the hub, so the `selection`
package models the user-chosen subset: its storage in the application configuration under
`selected_devices`, its rendering as a multi-select in the app manifest, and its maintenance when
the hub deletes a device. Its central invariant is that "never configured" and "deselected
everything" are different states and must stay distinguishable through JSON, through copies and
through the manifest.

## Requirements

### Requirement: Nil And Empty Selections Are Distinct
A `Selection` SHALL be a `[]string` of device IDs in which a nil value includes every available
device and a non-nil zero-length value includes none. Every copy the package produces SHALL preserve
that distinction. `Clone` SHALL be the copying primitive, because both `slices.Clone` and
`append([]string(nil), s...)` collapse an empty selection to nil.

#### Scenario: unconfigured application
- **WHEN** a `Selection` is nil
- **THEN** `IncludeAll()` returns true
- **AND** `Contains(id)` returns true for any ID

#### Scenario: everything deselected
- **WHEN** a `Selection` is non-nil and empty
- **THEN** `IncludeAll()` returns false and `Contains(id)` returns false for every ID
- **AND** `Clone()` returns a non-nil empty selection, not nil

### Requirement: Selection Mutation
`Without(id)` SHALL return a copy with the device removed together with `true`, and SHALL return the
receiver unchanged together with `false` when the ID is absent. A nil selection SHALL always be
returned unchanged with `false`: "every device" cannot express an exclusion, so an application that
supports device deletion must keep an explicit selection.

#### Scenario: removing a selected device
- **WHEN** `Without("b")` is called on `["a", "b", "c"]`
- **THEN** `["a", "c"]` and `true` are returned
- **AND** the receiver is not modified

#### Scenario: removing from an include-all selection
- **WHEN** `Without("b")` is called on a nil selection
- **THEN** nil and `false` are returned

### Requirement: Configuration Mixin
The package SHALL provide a `Devices` struct with a single field serialised as `selected_devices`,
to be embedded in an application's public configuration, and a `Selectable` interface satisfied by
any model embedding it. The JSON round trip SHALL preserve the nil/empty distinction in both
directions: an absent key or `null` unmarshals to nil and marshals back to `null`, while `[]`
unmarshals to a non-nil empty selection and marshals back to `[]`.

#### Scenario: absent key survives a rewrite
- **WHEN** a configuration containing no `selected_devices` key is unmarshalled and marshalled again
- **THEN** the output contains `"selected_devices":null`
- **AND** the selection still reports `IncludeAll()`

#### Scenario: empty list survives a rewrite
- **WHEN** a configuration containing `"selected_devices":[]` is unmarshalled and marshalled again
- **THEN** the output contains `"selected_devices":[]`

### Requirement: Selection Store
`NewStore(get, set)` SHALL wire a `Store` to a configuration service's read and write paths rather
than introducing a second lock over the same model: `get` must read a copy under the configuration
service's read lock and `set` must apply the value under its write lock, stamp the configuration
time and save. `Get` SHALL return the stored selection preserving the nil/empty distinction. `Set`
SHALL store a `Clone` of the supplied selection so a caller mutating the slice it handed over cannot
alter the stored configuration.

#### Scenario: caller mutates the slice it passed
- **WHEN** `Set(sel)` is called and the caller then modifies `sel` in place
- **THEN** the stored configuration is unaffected

### Requirement: Selection Removal On Device Deletion
`Store.Remove(id)` SHALL drop a device from the selection and SHALL be a no-op — no write, no
configuration-time stamp — when the device is not selected or the selection includes every device.
Its read-then-write is not atomic, so it SHALL be invoked only under the same handler lock as the
configuration writes it can race with.

#### Scenario: deleting an unselected device
- **WHEN** `Remove("z")` is called and `z` is not in the selection
- **THEN** no write reaches the configuration service and nil is returned

#### Scenario: deleting under an include-all selection
- **WHEN** `Remove("a")` is called on a nil selection
- **THEN** nothing is written and nil is returned, and the next sync will recreate the device

### Requirement: Adapter Deletion Integration
`adapter.WithSelection(remover, locker)` SHALL wire a `*selection.Store` into the adapter's
`cmd.thing.delete` handler and SHALL panic at wiring time if either argument is nil, rather than
returning a configuration that silently loses updates. The locker SHALL be the same
`router.MessageHandlerLocker` passed to `app.RouteApp`, so a delete cannot interleave with
`cmd.config.extended_set` rewriting the selection. The handler SHALL resolve the device ID with
`ExchangeAddress` before destroying the thing — the destroy drops the address-to-ID record — and
SHALL deselect before destroying, so a failed deselect leaves the thing intact and retryable
instead of leaving a deleted device selected for the next sync to recreate. When no thing state maps
the address to an ID, the address itself SHALL be used as the ID.

#### Scenario: deselect fails
- **WHEN** the selection store returns an error during `cmd.thing.delete`
- **THEN** the thing is not destroyed and the handler returns an error

#### Scenario: node the adapter never registered
- **WHEN** `cmd.thing.delete` names an address `ExchangeAddress` cannot resolve
- **THEN** the address is passed to the remover as the device ID

### Requirement: Manifest Selector Rendering
`PrepareManifest` SHALL render the device selector into the manifest in one of three states, driven
by a `Block` naming the UI block and the application config that host it. When the application is
not ready to list devices it SHALL set the block text to `NotReady` and hide the config. When the
fetch fails it SHALL set the block text to `Failed`, leave the selector exactly as the template
defines it so a stale option list is never presented as current, and return the wrapped error for
the caller to log. When the fetch succeeds it SHALL replace the config's select options with one
option per device and clear the config's hidden flag. Block texts SHALL be written under the `en`
language key.

#### Scenario: user not logged in
- **WHEN** `PrepareManifest` is called with `ready` false
- **THEN** the UI block's `en` text becomes `NotReady`, the config is hidden and nil is returned
- **AND** the device fetch is not attempted

#### Scenario: device fetch fails
- **WHEN** the fetch function returns an error
- **THEN** the UI block's `en` text becomes `Failed`, the config's options are left untouched and a
  `selection: fetch available devices` error is returned

#### Scenario: devices listed
- **WHEN** the fetch returns three devices
- **THEN** the config carries three select options built by the supplied `option` function and is
  not hidden

### Requirement: Manifest Rendering Tolerates A Renamed Template
`PrepareManifest` SHALL treat a manifest template that does not declare the referenced UI block or
application config as a no-op for the missing element rather than dereferencing it, so renaming
either in packaging cannot panic the application. A block with an empty text for the state being
rendered SHALL be left untouched.

#### Scenario: block missing from the template
- **WHEN** the manifest has no UI block with the configured ID
- **THEN** no text is set and the remaining steps still run

#### Scenario: config missing from the template
- **WHEN** the manifest has no application config with the configured ID and `ready` is true
- **THEN** the fetch still runs and its error is still returned, but no options are written
