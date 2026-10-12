# App Service Contract Specification

## Purpose
Defines the FIMP-facing contract every cliffhanger application exposes on its own app service:
installation and configuration (`cmd.config.extended_set`, `cmd.app.uninstall`), introspection
(`cmd.app.get_manifest`, `cmd.app.get_state`, `cmd.config.get_extended_report`, `cmd.app.get_diag`)
and authentication (`cmd.auth.login`, `cmd.auth.logout`, `cmd.auth.set_tokens`). The hub UI drives an
adapter exclusively through these commands and the app manifest, so the reply event names, the
report payload shapes and the manifest model are a hard compatibility surface. It also covers the
optional capability interfaces an application may implement and what `app.RouteApp` does when one is
absent.

## Requirements

### Requirement: Mandatory Application Contract
An application SHALL implement `app.App`: `GetManifest`, `Configure(config any)`, `Uninstall` and —
through the embedded `LogProvider` — `ErrorsReport`. `Configure` SHALL accept the value produced by
the `configFactory` passed to `RouteApp`, which is the type the incoming `cmd.config.extended_set`
payload was unmarshalled into. `GetManifest` SHALL return a manifest instance the caller may mutate,
because the handler writes `AppState` and `ConfigState` into it before replying.

#### Scenario: diagnostics are not optional
- **WHEN** an application type is written without an `ErrorsReport` method
- **THEN** it does not satisfy `app.App` and cannot be passed to `RouteApp` at all

### Requirement: Optional Capability Interfaces
`RouteApp` SHALL type-assert the application against `ResettableApp`, `LogginableApp`,
`AuthorizableApp` and `LogoutableApp` and SHALL register the matching routing only for those it
implements. A command whose interface is not implemented SHALL have no routing, so the message is
ignored and no reply of any kind is published. `cmd.auth.logout` SHALL be registered exactly once
from the `LogoutableApp` assertion, never additionally from `LogginableApp` or `AuthorizableApp`:
the router dispatches a message to every matching routing, so a second registration would log out
twice and publish two contradictory status reports.

#### Scenario: application without a reset
- **WHEN** the application does not implement `ResettableApp` and `cmd.app.reset` arrives
- **THEN** no routing votes for the message and no `evt.app.config_action_report` is published

#### Scenario: application implementing both authentication interfaces
- **WHEN** the application implements both `LogginableApp` and `AuthorizableApp`
- **THEN** `RouteApp` returns exactly one `cmd.auth.logout` routing, alongside one `cmd.auth.login`
  and one `cmd.auth.set_tokens` routing

#### Scenario: authorizable application starts unauthenticated
- **WHEN** `RouteApp` asserts `AuthorizableApp` and the lifecycle auth state is still `NA`
- **THEN** the auth state is set to `NOT_AUTHENTICATED` while the routing is being built

### Requirement: Command And Reply Event Pairs
Every app routing SHALL be filtered by the application's own service name and by message type, and
SHALL answer with the fixed reply event of its command: `cmd.app.get_manifest` →
`evt.app.manifest_report`, `cmd.app.get_state` → `evt.app.state_report`,
`cmd.config.get_extended_report` → `evt.config.extended_report`, `cmd.config.extended_set` →
`evt.app.config_report`, `cmd.app.uninstall` → `evt.app.uninstall_report`, `cmd.app.reset` →
`evt.app.config_action_report`, `cmd.app.get_diag` → `evt.app.diag_report`, and `cmd.auth.login`,
`cmd.auth.logout` and `cmd.auth.set_tokens` → `evt.auth.status_report`. All replies SHALL be object
messages carrying the request's correlation data. `RouteApp` SHALL always register the six routings
that need no capability assertion: get_state, get_extended_report, get_manifest, extended_set,
uninstall and get_diag.

#### Scenario: state request
- **WHEN** `cmd.app.get_state` is received on the app service
- **THEN** `evt.app.state_report` is published with the full lifecycle snapshot (health, connection,
  config and auth axes) as its object value

#### Scenario: foreign service
- **WHEN** a `cmd.app.get_state` message names a different service
- **THEN** the routing does not vote for it and nothing is published

### Requirement: Manifest Report Modes
The `cmd.app.get_manifest` handler SHALL treat a null-valued request as a request for the bare
manifest, and SHALL read a string value otherwise. Only the values `manifest_state` and `full` SHALL
cause the reply manifest's `app_state` to be filled with the lifecycle snapshot and its
`config_state` with the public configuration; any other string SHALL yield the manifest as loaded.
A value that is neither null nor a string, and a failing `GetManifest`, SHALL produce a processing
error instead of a manifest report.

#### Scenario: hub opens the app page
- **WHEN** `cmd.app.get_manifest` arrives with the string value `full`
- **THEN** `evt.app.manifest_report` carries the manifest with `app_state` and `config_state`
  populated

#### Scenario: malformed mode
- **WHEN** the request value is an object rather than a string
- **THEN** the handler returns the error "provided value has an incorrect format" and the router
  replies with an error report, not a manifest report

### Requirement: Public Configuration Redaction
Configuration exposed over FIMP SHALL be taken from `publicConfigState`: if the configuration
storage implements `PublicModeler` the value of `PublicModel()` SHALL be reported, otherwise the raw
`Model()`. This SHALL apply both to `evt.config.extended_report` and to the `config_state` field of
a state-carrying manifest report, so a storage that redacts secrets redacts them in both places.

#### Scenario: redacting config service
- **WHEN** the configuration storage is a `config.Service` with a redact function and
  `cmd.config.get_extended_report` arrives
- **THEN** the reply carries the redacted model, never the stored one

### Requirement: Configuration Reports Never Fail The Message
The `cmd.config.extended_set` and `cmd.app.uninstall` handlers SHALL always reply with a
`ConfigurationReport` — `{op_status, op_error, app_state}` — and SHALL NOT surface a processing
error. On success `op_status` SHALL be `ok`; on a payload unmarshalling failure or a failing
`Configure`/`Uninstall` it SHALL be `error` with `op_error` set to `configure the app err: <error>`.
The report SHALL always carry the current lifecycle snapshot in `app_state`, so the hub learns the
resulting state even when the operation failed.

#### Scenario: configuration rejected by the application
- **WHEN** `Configure` returns an error
- **THEN** `evt.app.config_report` is published with `op_status` `error`, a populated `op_error` and
  the current app state

#### Scenario: unparsable configuration payload
- **WHEN** the `cmd.config.extended_set` value cannot be unmarshalled into the configuration type
- **THEN** `Configure` is not called and the same `evt.app.config_report` error shape is published

### Requirement: Uninstall Excludes Things First
The `cmd.app.uninstall` handler SHALL invoke the `excludeAllThings` callback, when one was provided
to `RouteApp`, before calling `App.Uninstall`. A failure of that callback SHALL be logged and SHALL
NOT abort the uninstall or change the reply: the application's own `Uninstall` still runs and
decides `op_status`. A nil callback SHALL be skipped, which is the correct wiring for an application
that removes its things inside `Uninstall`.

#### Scenario: exclusion fails
- **WHEN** `excludeAllThings` returns an error
- **THEN** the error is logged, `App.Uninstall` is still called, and `evt.app.uninstall_report`
  reflects only the outcome of `Uninstall`

### Requirement: Config Action Commands
A config action command SHALL reply with `evt.app.config_action_report` carrying a
`manifest.ButtonActionResponse` — `{op, op_status, next, error_code, error_text}` — built by the
action function from the request's string value, which SHALL be passed as an empty string when the
value is not a string. `RouteConfigActionCommand` SHALL make this shape available for any
manifest-button command name. `cmd.app.reset` SHALL use it with `op` set to `cmd.app.reset`, and
SHALL report `op_status` `ok` with `next` `reload` on success, or `op_status` `error` with `next`
`ok` and `error_text` `failed to reset the application` when `Reset` fails.

#### Scenario: user presses the reset button
- **WHEN** `cmd.app.reset` is handled and `Reset` succeeds
- **THEN** `evt.app.config_action_report` carries `op_status` `ok` and `next` `reload`, telling the
  hub UI to reload the app page

#### Scenario: reset fails
- **WHEN** `Reset` returns an error
- **THEN** the error is logged and the response carries `op_status` `error`, `next` `ok` and the
  fixed error text

### Requirement: Authentication Status Reports
`cmd.auth.login`, `cmd.auth.set_tokens` and `cmd.auth.logout` SHALL reply with
`evt.auth.status_report` carrying an `AuthenticationReport` whose `status` is the lifecycle auth
state read *after* the operation. A failing `Login`, `Authorize` or `Logout` SHALL be logged and
SHALL set `error_text` to `failed to login`, `failed to authorize` or `failed to logout`
respectively, while still producing a status report rather than an error. A credentials payload that
cannot be unmarshalled — into `LoginCredentials` `{username, password, encrypted}` for login, into
`auth.OAuth2TokenResponse` for set_tokens — SHALL instead produce a processing error and no status
report.

#### Scenario: login rejected by the vendor
- **WHEN** `Login` returns an error
- **THEN** `evt.auth.status_report` carries `error_text` `failed to login` and the current auth state
  as `status`

#### Scenario: invalid credentials payload
- **WHEN** the `cmd.auth.login` value is not a credentials object
- **THEN** the handler returns "provided login credentials have an invalid format" and no
  `evt.auth.status_report` is published

### Requirement: Deprecated FHX Errors Field
The login handler SHALL set the deprecated `errors` field only when the attempt actually failed:
when `Login` returned an error, or when the resulting auth state is one of `NOT_AUTHENTICATED`,
`ERROR` or `LOST`. Any other state — including `IN_PROGRESS` for a login whose second step is still
running — SHALL leave `errors` empty so an unfinished flow is not reported as a failure. The field
SHALL be omitted from the JSON when empty, and SHALL NOT be set by the logout or set_tokens
handlers.

#### Scenario: asynchronous login still running
- **WHEN** `Login` returns no error and leaves the auth state `IN_PROGRESS`
- **THEN** the report carries `status` `IN_PROGRESS` and no `errors` field

#### Scenario: credentials refused
- **WHEN** `Login` returns no error but the auth state ends as `NOT_AUTHENTICATED`
- **THEN** `errors` is set to `failed to login`

### Requirement: Serialized Mutating Commands
The mutating handlers — extended_set, uninstall, reset, login, logout and set_tokens — SHALL be
wired with the external locker passed to `RouteApp`, so only one of them runs at a time. When the
lock is already held the handler SHALL NOT run the operation and SHALL reply with an error report
carrying "another operation is already running, skipping message". The read-only handlers —
get_state, get_extended_report, get_manifest and get_diag — SHALL NOT take the lock and SHALL stay
answerable while a long installation runs.

#### Scenario: configuration arrives during an uninstall
- **WHEN** `cmd.config.extended_set` is handled while `cmd.app.uninstall` still holds the lock
- **THEN** `Configure` is not called and an error report is published instead of
  `evt.app.config_report`

#### Scenario: state query during an install
- **WHEN** `cmd.app.get_state` arrives while the lock is held
- **THEN** `evt.app.state_report` is published normally

### Requirement: Diagnostic Report
The `cmd.app.get_diag` handler SHALL reply with `evt.app.diag_report` carrying a `DiagReport`:
`uptime` in whole seconds since process start, `restarts_count` from the lifecycle, and `errors` as
returned by the `LogProvider`. A `LogProvider` returning an error SHALL abort the reply with
"failed to retrieve error log entries", so the router publishes an error report and never a partial
diag report. A nil `LogProvider` SHALL NOT be passed to `RouteCmdAppDiagGetReport`: the handler
dereferences it unconditionally, and the resulting panic is contained by the router's per-message
recovery, leaving the command silently unanswered.

#### Scenario: hub requests diagnostics
- **WHEN** `cmd.app.get_diag` arrives and `ErrorsReport` returns two entries
- **THEN** `evt.app.diag_report` is published with both entries, the current uptime and the restart
  count

#### Scenario: log provider fails
- **WHEN** `ErrorsReport` returns an error
- **THEN** no `evt.app.diag_report` is published

### Requirement: Process-Wide Log Capture
`NewLogCapture` SHALL return a `LogProvider` backed by a single `formatters.ErrorHook` installed on
the global logrus logger, and SHALL install that hook at most once per process: later calls SHALL
return the same instance so repeated application construction does not stack duplicate hooks. The
hook SHALL capture only `warn`, `error`, `fatal` and `panic` entries, formatted without colors and
without the trailing newline, into a ring buffer of 64 entries; once full it SHALL overwrite the
oldest. `ErrorsReport` SHALL drop entries older than 30 days before returning the remainder in
chronological order.

#### Scenario: buffer overflows
- **WHEN** more than 64 warn-or-worse entries have been logged
- **THEN** `ErrorsReport` returns the most recent 64, oldest first

#### Scenario: info entries
- **WHEN** the application logs at info or debug level
- **THEN** nothing is added to the buffer and the diag report is unaffected

#### Scenario: two applications in one process
- **WHEN** `NewLogCapture` is called twice
- **THEN** both calls return the same hook and a single log entry appears once in the report

### Requirement: Manifest Model
A manifest SHALL consist of `configs`, `ui_blocks`, `ui_buttons`, `auth`, `init_flow`, `services`
plus the runtime-filled `app_state` and `config_state`. Labels, headers, texts and footers SHALL be
`MultilingualLabel` maps keyed by language code, and select options SHALL be `{val, label}` pairs.
`AppConfig`, `AppUBLock` and `UIButton` SHALL each expose `Hide()`/`Show()` toggling their `hidden`
flag, and `GetAppConfig`, `GetUIBlock` and `GetButton` SHALL return a pointer into the manifest's
own slice — so a `Hide()` on a returned element mutates the manifest that will be reported — or
`nil` when no element carries the requested ID.

#### Scenario: hiding a button for an unauthenticated app
- **WHEN** `GetButton("login")` is called on a loaded manifest and `Hide()` is invoked on the result
- **THEN** the button in the manifest's `ui_buttons` slice has `hidden` set and the following
  `evt.app.manifest_report` carries it hidden

#### Scenario: unknown identifier
- **WHEN** `GetUIBlock` is called with an ID no block declares
- **THEN** it returns `nil` and the caller must not dereference it

### Requirement: Manifest Loading
`NewLoader(workDir)` SHALL return a `Loader` that reads `app-manifest.json` through the standard
storage layout: `<workDir>/defaults/app-manifest.json` first, then `<workDir>/data/app-manifest.json`
overlaid on top of it. Each `Load` SHALL unmarshal into a freshly allocated `Manifest`, so a caller
mutating one loaded manifest — hiding blocks, filling `app_state` — cannot affect a later load. When
neither file exists `Load` SHALL fail with an error wrapped as "manifest loader: failed to load the
manifest".

#### Scenario: shipped defaults only
- **WHEN** only `defaults/app-manifest.json` exists
- **THEN** the manifest is loaded from it and no error is returned

#### Scenario: per-request mutation
- **WHEN** `GetManifest` loads the manifest and the application hides a UI block before replying
- **THEN** the next `Load` returns the block visible again

### Requirement: Lifecycle Capability Interfaces
`InitializableApp` and `CheckableApp` SHALL be asserted by `TaskApp`, not by `RouteApp`, and an
application implementing neither SHALL get no background tasks. `Initialize` SHALL run once at
startup and, if it returns an error, SHALL set the app health to startup error and SHALL be retried
every 10 minutes for as long as that state holds. `Check` SHALL run only while the app health is
running, at the interval returned by `CheckInterval()`, and SHALL fall back to 30 minutes when that
interval is 0. A failing `Check` SHALL only be logged.

#### Scenario: initialization fails at boot
- **WHEN** `Initialize` returns an error during startup
- **THEN** the app health becomes startup error and initialization is retried 10 minutes later

#### Scenario: default check interval
- **WHEN** the application implements `CheckableApp` and returns 0 from `CheckInterval`
- **THEN** `Check` is scheduled every 30 minutes while the app is running
