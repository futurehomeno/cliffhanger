## MODIFIED Requirements

### Requirement: Error Reporting Contract
When the `MessageProcessor` returns an error, the handler SHALL reply with exactly one
`evt.error.report` message. The reply SHALL carry the request's service, value type `string` and,
as value, the error code found in the error chain (requirement "Coded Errors"), or `internal` when
the chain carries no code. It SHALL carry the properties `msg`, `cmd_topic`, `cmd_service` and
`cmd_type`, the last three always set from the request's topic, service and interface. `msg` of a
coded error SHALL be the error's own plain sentence, with no URL, identifier, key or response body.
`msg` of an `internal` error SHALL be the error chain's text with the URL of the first `*url.Error`
in the chain removed: that error SHALL be rendered as its operation and its cause, without its URL.
A URL written into an error's own text stays the adapter's to remove. The reply's
correlation ID SHALL be the request's UID. The reply SHALL be published to the request's
`resp_to` topic when it has one, and otherwise to the request's address (or the handler's default
address) with message type `evt`, as requirement "Reply Address Resolution" states. An error reply
addressed to a device resource SHALL be marked with the skip storage strategy so device errors are
not persisted by the hub. `WithSilentErrors()` SHALL suppress the reply while keeping the log; it is
meant for handlers of events, which answer nobody. A handler SHALL NOT send a report that looks like
success in place of the error report; a state report showing the real state MAY be published
before it.

#### Scenario: processor returns an error
- **WHEN** a message processor returns an error and silent errors are not enabled
- **THEN** an `evt.error.report` is replied with `msg`, `cmd_topic`, `cmd_service` and `cmd_type` properties
- **AND** its correlation ID is the request's UID

#### Scenario: error without a code
- **WHEN** a message processor returns `errors.New("boom")`
- **THEN** the reply's value is `internal` and its `msg` property is `boom`

#### Scenario: coded error
- **WHEN** a message processor returns an error coded `out_of_range` with the sentence "Current must not exceed 16 A."
- **THEN** the reply's value is `out_of_range` and its `msg` property is "Current must not exceed 16 A."

#### Scenario: URL removed from an uncoded chain
- **WHEN** a message processor returns an uncoded error wrapping a `*url.Error` for `Get "https://10.0.0.5/api/KEY/lights/1"` with cause `context deadline exceeded`
- **THEN** the reply's value is `internal` and its `msg` property contains `Get` and `context deadline exceeded` but not `https://` and not `KEY`

#### Scenario: request carries a response topic
- **WHEN** a failing request sets `resp_to`
- **THEN** the error report is published to the `resp_to` address and still carries the request's UID as correlation ID

#### Scenario: state not reported yet
- **WHEN** a message processor returns an error wrapping `router.ErrNotReported`
- **THEN** the error is logged at warning level and the `evt.error.report` is still replied, with the value `unavailable`

#### Scenario: device error not stored
- **WHEN** the error reply resolves to a `rt:dev` address and the payload has no storage strategy
- **THEN** the payload is tagged with the skip storage strategy before publishing

## ADDED Requirements

### Requirement: Coded Errors
An error returned by a handler MAY carry an error code and a plain sentence; the router SHALL find
them anywhere in the error chain. The codes SHALL be the lower_snake values of fimpgo's
`fimptype.ErrorCodeT`:

| code | meaning |
| --- | --- |
| `bad_request` | the payload is missing, of the wrong value type or malformed |
| `out_of_range` | a number is outside the range the service advertises |
| `not_supported` | a value is not in the list the service advertises, or the device lacks the capability |
| `not_found` | no service, thing or item exists under the given address or identifier |
| `busy` | another operation is running; the same command may succeed later |
| `unauthorized` | the vendor cloud refused the adapter's credentials |
| `device_refused` | the device or vendor cloud received the command and refused it |
| `unavailable` | the device, the vendor cloud or the requested state cannot be reached or is not known yet |
| `internal` | anything else, and every error that carries no code |

An adapter MAY use its own lower_snake code for a refusal none of these names, for example
`eco_mode`. A handler whose lock is already held SHALL answer `busy` with the sentence "another
operation is already running, skipping message". A payload that cannot be read as the command's
value type SHALL be answered with `bad_request`, an address under which no service or thing is
registered with `not_found`, and an error wrapping `router.ErrNotReported` with `unavailable`.

#### Scenario: busy handler lock
- **WHEN** a second message reaches a handler whose shared locker is already held
- **THEN** the processor is not invoked and the `evt.error.report` carries the value `busy`

#### Scenario: code found deep in the chain
- **WHEN** a processor returns `fmt.Errorf("set level: %w", e)` where `e` is coded `device_refused`
- **THEN** the reply's value is `device_refused` and its `msg` is the sentence of `e`

#### Scenario: adapter's own code
- **WHEN** a processor returns an error coded `eco_mode`
- **THEN** the reply's value is `eco_mode`

#### Scenario: unknown address
- **WHEN** a command arrives on a topic no service is registered under
- **THEN** the reply's value is `not_found`

### Requirement: A Late Failure Is Reported With The Command's Correlation Id
A failure found after the handler accepted a command, for example a deferred write that the vendor
cloud refuses later, SHALL be answered with one `evt.error.report` built from the original request
by the same rules as requirement "Error Reporting Contract": the request's service, topic resolution,
`cmd_*` properties, code, `msg` and the request's UID as correlation ID. The framework SHALL provide
one function that builds this report from the stored request and the error, so the adapter only
publishes it. No `evt.error.report` SHALL be published without a correlation ID, and a late failure
SHALL NOT be answered with a report that looks like success.

#### Scenario: deferred write refused
- **WHEN** a handler accepted `cmd.max_current.set` and, after it returned, the vendor cloud refuses the write
- **THEN** one `evt.error.report` is published whose correlation ID is the UID of that `cmd.max_current.set`
- **AND** its `cmd_topic`, `cmd_service` and `cmd_type` are those of that command

#### Scenario: late failure with a response topic
- **WHEN** the stored request set `resp_to`
- **THEN** the late error report is published to the `resp_to` address

### Requirement: One Log Line Per Failure
A failure answered with `evt.error.report` SHALL be logged exactly once, by the router or by the
late-failure function, with the request topic, service, type, the code and the full error chain,
URLs included. A handler, service or controller SHALL return the error, wrapped with what it was
doing, and SHALL NOT log it as well. A failure with a code, an adapter's own code and an error
wrapping `router.ErrNotReported` included, SHALL be logged at warning level; a failure without a
code (`internal`) SHALL be logged at error level. An adapter MAY limit the log line of a refusal
that repeats to once per period; the `evt.error.report` is still sent for every failed command.

#### Scenario: coded failure is a warning
- **WHEN** a processor returns an error coded `unavailable` or `unauthorized`
- **THEN** exactly one log line is written for it, at warning level, carrying the full error chain

#### Scenario: uncoded failure is an error
- **WHEN** a processor returns an error that carries no code
- **THEN** exactly one log line is written for it, at error level, carrying the full error chain

#### Scenario: repeated refusal
- **WHEN** an adapter receives the same refusal for three commands within its log period
- **THEN** three `evt.error.report` messages are published, each with its command's correlation ID, and one log line is written
