## MODIFIED Requirements

### Requirement: Configuration Mixin
The package SHALL provide a `Devices` struct with a single field serialised as `selected_devices`,
to be embedded in an application's public configuration, and a `Selectable` interface satisfied by
any model embedding it. `Devices.Selection()` SHALL return a pointer to the embedded field, so a
write through it changes the configuration model itself. The JSON round trip SHALL preserve the
nil/empty distinction in both directions: an absent key or `null` unmarshals to nil and marshals
back to `null`, while `[]` unmarshals to a non-nil empty selection and marshals back to `[]`.

#### Scenario: absent key survives a rewrite
- **WHEN** a configuration containing no `selected_devices` key is unmarshalled and marshalled again
- **THEN** the output contains `"selected_devices":null`
- **AND** the selection still reports `IncludeAll()`

#### Scenario: empty list survives a rewrite
- **WHEN** a configuration containing `"selected_devices":[]` is unmarshalled and marshalled again
- **THEN** the output contains `"selected_devices":[]`

#### Scenario: writing through the selectable pointer
- **WHEN** `*cfg.Selection() = selection.Selection{"a"}` is assigned on a model embedding `Devices`
- **THEN** the model's `SelectedDevices` is `["a"]`

### Requirement: Selection Removal On Device Deletion
`Store.Remove(id)` SHALL drop every occurrence of a device from the selection and SHALL be a no-op —
no write, no configuration-time stamp — when the device is not selected or the selection includes
every device. An error from the store's `set` function SHALL be returned unchanged, so the caller
can keep the device and retry. Its read-then-write is not atomic, so it SHALL be invoked only under
the same handler lock as the configuration writes it can race with.

#### Scenario: deleting an unselected device
- **WHEN** `Remove("z")` is called and `z` is not in the selection
- **THEN** no write reaches the configuration service and nil is returned

#### Scenario: deleting under an include-all selection
- **WHEN** `Remove("a")` is called on a nil selection
- **THEN** nothing is written and nil is returned, and the next sync will recreate the device

#### Scenario: configuration write fails
- **WHEN** `Remove("a")` is called on a selection containing `a` and `set` returns an error
- **THEN** that error is returned
