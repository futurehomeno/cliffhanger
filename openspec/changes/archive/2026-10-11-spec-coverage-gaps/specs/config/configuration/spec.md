## ADDED Requirements

### Requirement: Default Settings Constructors
`config.NewDefault(workDir)` SHALL return a `Default` whose `WorkDir` and `ConfigDir` both hold
`workDir`, for the Thingsplex layout where configuration and data share one directory.
`config.NewCanonicalDefault(cfgDir, workDir)` SHALL set `ConfigDir` to `cfgDir` and `WorkDir` to
`workDir`. Every other field SHALL be left at its zero value for the storage to load.

#### Scenario: Canonical layout
- **WHEN** `NewCanonicalDefault("/etc/app", "/var/lib/app")` is called
- **THEN** `ConfigDir` is `/etc/app` and `WorkDir` is `/var/lib/app`

### Requirement: Restart Counter Save Failure
`DefaultStore.IncrementRestartsCount` SHALL increment the in-memory `RestartsCount` under the write
lock, save, and return the new count. When the save fails it SHALL return zero together with the
save error; the in-memory count SHALL stay incremented, so a later successful save persists it.

#### Scenario: Save fails while counting a restart
- **WHEN** `IncrementRestartsCount` is called and the save function returns an error
- **THEN** zero and that error are returned
- **AND** `RestartsCount()` reports the incremented value

### Requirement: Configuration Route Value Decoding
After the value-type check, a set route SHALL decode the payload value into the setter's Go type
through a JSON round trip — the raw object for object routes — and SHALL return the decoding error
without calling the setter when the value does not fit that type. The reply SHALL carry the decoded
value as received, not a value re-read through a getter.

#### Scenario: Object of the wrong shape
- **WHEN** an object route whose setter takes a struct with an integer field receives an object
  carrying a string in that field
- **THEN** an error is returned and the setter is not called
