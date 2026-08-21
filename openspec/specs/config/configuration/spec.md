# Configuration Management Specification

## Purpose
Every edge adapter keeps a JSON configuration file that the hub reads and writes at runtime over
FIMP, while the adapter itself reads the same values from many goroutines. This capability provides
the shared baseline settings every adapter inherits, a generic thread-safe service wrapping the
adapter's own configuration model, versioned migrations for upgrading a file written by an older
release, validators and typed FIMP routes for exposing individual settings, and the change events
other components subscribe to.

## Requirements

### Requirement: Single Non-Reentrant Configuration Lock
`config.Service[C]` and its `DefaultStore` SHALL share one `sync.RWMutex`, because both mutate the
same underlying model. Mutating operations SHALL take the write lock and reads SHALL take the read
lock, so a concurrent writer can never observe or produce a torn model. The lock SHALL NOT be
reentrant: callbacks passed to `Update`, `Persist` and `Migrate` and accessors passed to `Get`
SHALL NOT call back into `Service` or `DefaultStore` methods.

#### Scenario: Concurrent writers
- **WHEN** two goroutines call `Update` at the same time
- **THEN** the second callback runs only after the first has returned and its save completed

#### Scenario: Callback re-enters the service
- **WHEN** a callback passed to `Update` calls another `Service` or `DefaultStore` method
- **THEN** the call deadlocks — the contract forbids it

### Requirement: Default Settings Store
`config.Default` SHALL carry the settings shared by every adapter — `WorkDir`, `ConfigDir`,
`ConfigVersion`, the `MQTT*` connection settings, the `Log*` settings (`LogFile`, `LogLevel`,
`LogFormat`, `LogRevertTimeout`, `LogFlushInterval`, `LogRevertAt`), `InfoFile`, `RestartsCount`,
`Telemetry` and `ConfiguredAt` — and `DefaultStore` SHALL expose the runtime-mutable ones through
lock-guarded accessors. `WorkDir` and `ConfigDir` SHALL be tagged `json:"-"` and therefore
never persisted: they come from the command line. Every setter and `Save` SHALL stamp `ConfiguredAt`
with the current time formatted as RFC 3339 with nanoseconds before writing, so the hub can tell
when the configuration last changed. `IncrementRestartsCount` SHALL be the exception: it saves
without stamping, since a restart is not a configuration change.

#### Scenario: Setting the log level
- **WHEN** `SetLevel` is called
- **THEN** the model's `LogLevel` is updated, `ConfiguredAt` is stamped and the storage is saved

#### Scenario: Counting a restart
- **WHEN** `IncrementRestartsCount` is called
- **THEN** `RestartsCount` is incremented and persisted
- **AND** `ConfiguredAt` is left unchanged

#### Scenario: Reloaded configuration file
- **WHEN** the configuration is written and read back
- **THEN** `WorkDir` and `ConfigDir` are absent from the file

### Requirement: Deep-Copied Default Snapshot
`DefaultStore.Default` SHALL return a deep copy taken under the read lock, safe to use after the
lock is released — including for deferred JSON marshaling such as a FIMP report payload. The
embedded `*TelemetryConfig`, its `Suppressed` entry and that entry's `Domains` and `Events` slices
SHALL be cloned so a caller cannot mutate shared state.

#### Scenario: Caller mutates the snapshot
- **WHEN** a caller appends to `Default().Telemetry.Suppressed.Domains`
- **THEN** the stored configuration is unaffected

#### Scenario: Snapshot marshaled after the lock is released
- **WHEN** the returned snapshot is marshaled while another goroutine calls a setter
- **THEN** the marshaled output is consistent and no data race occurs

### Requirement: Telemetry Configuration Storage
`DefaultStore.SetTelemetry` SHALL store a clone of the supplied configuration, including cloned
`Suppressed.Domains` and `Suppressed.Events` slices, so the caller's value cannot mutate the stored
one afterwards. `DefaultStore.Telemetry` SHALL return an error when no telemetry configuration is
present, whereas `Default.GetTelemetry` SHALL return the zero `TelemetryConfig` and no error in
that case.

#### Scenario: Telemetry never configured
- **WHEN** `DefaultStore.Telemetry` is called and the model has no telemetry configuration
- **THEN** an error is returned

#### Scenario: Caller mutates the value it stored
- **WHEN** a caller mutates the slices of a `TelemetryConfig` it passed to `SetTelemetry`
- **THEN** the stored configuration keeps the values as of the call

### Requirement: Guarded Model Updates
`Service.Update` SHALL apply the caller's function to the model under the write lock, stamp
`ConfiguredAt` and save. `Service.Persist` SHALL apply the function under the same lock and save
*without* stamping, for caches and internal state that are not user configuration. Both SHALL
return the save error.

#### Scenario: User changes a setting
- **WHEN** a handler calls `Update` to change a setting
- **THEN** the model is saved and `ConfiguredAt` reflects the moment of the change

#### Scenario: Adapter persists a cache
- **WHEN** the adapter calls `Persist` to store internal state
- **THEN** the model is saved and `ConfiguredAt` is unchanged

### Requirement: Reset Preserves Runtime Directories
`Service.Reset` SHALL restore the configuration from the defaults file under the write lock and
SHALL restore `WorkDir` and `ConfigDir` to the values the process was started with, on the success
path and on the error path alike. Without this the application — which only asks for a reload after
a reset — would resolve every later path against the current working directory, because those two
fields are not persisted and `Storage.Reset` zeroes the model before reloading.

#### Scenario: Reset with a readable defaults file
- **WHEN** `Reset` succeeds
- **THEN** the model holds the default settings
- **AND** `WorkDir` and `ConfigDir` still hold the values the process started with

#### Scenario: Reset with an unreadable defaults file
- **WHEN** `Storage.Reset` fails after zeroing the model
- **THEN** the error is returned
- **AND** `WorkDir` and `ConfigDir` are still restored

### Requirement: Migration Execution
`Service.Migrate` SHALL run the supplied migrations under the write lock and SHALL save the model
only when at least one step was applied. A migration error SHALL be returned without saving.

#### Scenario: Configuration already current
- **WHEN** `Migrate` is called and no migration's `From` matches `ConfigVersion`
- **THEN** no step runs and the storage is not written

#### Scenario: Migration step fails
- **WHEN** a migration's `Do` returns an error
- **THEN** `Migrate` returns that error and does not save

### Requirement: Migration Chain Semantics
`Default.Migrate` SHALL repeatedly select the first migration whose `From` equals the current
`ConfigVersion`, run its `Do` (a nil `Do` is a version bump only), and only then set `ConfigVersion`
to its `To`, continuing until no step matches. It SHALL return the number of applied steps. A step
whose `To` equals its `From` SHALL be rejected as not advancing the version, and a chain revisiting
an already-visited version SHALL be rejected as a cycle; in both cases the count of steps applied so
far is returned alongside the error. `Migrate` SHALL only mutate the in-memory model — persisting
the result is the caller's responsibility.

#### Scenario: Chained upgrade
- **WHEN** `ConfigVersion` is 0 and migrations 0→1 and 1→2 are supplied
- **THEN** both steps run in order, `ConfigVersion` becomes 2 and the applied count is 2

#### Scenario: Cyclic chain
- **WHEN** migrations 1→2 and 2→1 are supplied and `ConfigVersion` is 1
- **THEN** an error naming the cycle is returned after the steps already applied

#### Scenario: Failed step leaves the version behind
- **WHEN** a step's `Do` returns an error
- **THEN** `ConfigVersion` is not advanced past that step and earlier steps remain applied in memory

### Requirement: Redacted Public Model
`Service.PublicModel` SHALL return the model passed through the configured redact function, or the
model itself when no redact function was given, read under the read lock. It satisfies the
application's public-model contract used for reporting the configuration to the hub. For a
reference-type model the no-redactor path aliases the live configuration once the lock is released,
so adapters SHOULD supply a redact function returning a copy even when nothing needs hiding.

#### Scenario: Credentials in the configuration
- **WHEN** `PublicModel` is called on a service constructed with a redact function
- **THEN** the redacted value is returned instead of the live model

### Requirement: Locked Setting Reads
`config.Get` SHALL evaluate the caller's accessor under the read lock and return its result; the
accessor SHOULD return a value or a copy, since a returned reference type aliases the live model
after the lock is released. `config.GetDuration` SHALL read a duration persisted as a string and
fall back to the supplied default whenever the string is empty or not parsable by
`time.ParseDuration`.

#### Scenario: Unparsable duration setting
- **WHEN** `GetDuration` reads a setting holding `"soon"`
- **THEN** the supplied default duration is returned

#### Scenario: Valid duration setting
- **WHEN** `GetDuration` reads a setting holding `"30s"`
- **THEN** 30 seconds is returned

### Requirement: Setting Validators
`config.Validate` SHALL wrap a setter with an ordered list of validators, returning the first
validation error wrapped as a configuration validation failure and invoking the setter only when
all validators pass. `Within` SHALL reject a value absent from the allowed list, skipping the check
for a zero value when marked optional. `Between` SHALL reject values outside the inclusive
minimum/maximum range. `Lesser` SHALL reject values greater than *or equal to* its bound and
`Greater` SHALL reject values lesser than *or equal to* its bound. `Contains` SHALL report slice
membership and `Deduplicate` SHALL return the distinct elements of a slice in unspecified order.

#### Scenario: Rejected value
- **WHEN** a validated setter receives a value outside the allowed range
- **THEN** the wrapped setter is not called and a validation error is returned

#### Scenario: Optional setting left empty
- **WHEN** a `Within` validator marked optional receives the zero value
- **THEN** the value is accepted

#### Scenario: Deduplicating a list
- **WHEN** `Deduplicate` is given a slice with repeats
- **THEN** each distinct element appears exactly once, in no guaranteed order

### Requirement: Persisted Backoff Configuration
`config.BackoffConfig` SHALL persist a backoff specification as three duration strings and two
counts, and `Stateful(def)` SHALL build a `backoff.Stateful` from it, substituting the corresponding
field of `def` for every duration that is empty or unparsable and for every count that is zero. A
duration unset in both the configuration and the defaults SHALL become zero.

#### Scenario: Partially configured backoff
- **WHEN** only `Initial` is set and the defaults supply the rest
- **THEN** the built backoff uses the configured initial delay and the defaults for the remaining
  delays and counts

### Requirement: Typed Configuration Routes
The `RouteCmdConfigGet*` and `RouteCmdConfigSet*` helpers SHALL expose one setting each on a FIMP
service, listening on `cmd.config.get_<setting>` and `cmd.config.set_<setting>` and replying with
`evt.config.<setting>_report` carrying the value type the helper was built for.
`RouteCmdConfigGetReport` SHALL answer `cmd.config.get_report` with `evt.config.report` as an
object. A set route SHALL reject a message whose payload value type does not match the route's
value type before calling the setter, and SHALL return the setter's error without replying when the
setter fails. Durations SHALL travel as strings: the get route emits `time.Duration.String()` and
the set route parses the string with `time.ParseDuration`, rejecting an unparsable value.

#### Scenario: Wrong value type
- **WHEN** an integer setting receives `cmd.config.set_<setting>` with a string payload
- **THEN** an error is returned and the setter is not called

#### Scenario: Successful set
- **WHEN** a valid `cmd.config.set_<setting>` is handled
- **THEN** the setter is called with the decoded value and `evt.config.<setting>_report` echoes it
  back to the requester

#### Scenario: Duration setting
- **WHEN** `cmd.config.set_<setting>` for a duration route carries `"1h30m"`
- **THEN** the setter receives 90 minutes

### Requirement: Configuration Change Events
When a route is built with `config.WithConfigurationChangeEvent`, the set route SHALL publish a
`config` / `configuration_change` event naming the service and the setting on the supplied event
manager, and SHALL do so only after the setter succeeded. `config.PublishConfigurationChange` SHALL
publish the same event for hand-written routes given the same routing options, and SHALL do nothing
when no event manager was supplied. `config.PublishConfigurationChanges` SHALL emit one event per
named setting, and `config.WaitForConfigurationUpdate` SHALL build the event filter matching a
single service/setting pair.

#### Scenario: Setter fails
- **WHEN** a set route configured with an event manager runs a setter that returns an error
- **THEN** no configuration change event is published

#### Scenario: No event manager configured
- **WHEN** `PublishConfigurationChange` is called without `WithConfigurationChangeEvent`
- **THEN** nothing is published and no error occurs

#### Scenario: Subscriber waiting for a setting
- **WHEN** a component waits with `WaitForConfigurationUpdate` for a service and setting
- **THEN** it is woken only by a change event carrying that exact service and setting
