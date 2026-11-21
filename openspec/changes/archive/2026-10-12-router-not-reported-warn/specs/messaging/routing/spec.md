## MODIFIED Requirements

### Requirement: Error Reporting Contract
When the `MessageProcessor` returns an error, the handler SHALL log it with the request topic,
service and type, and SHALL reply with an `evt.error.report` message carrying the request's service,
a string value `"failed to process incoming message"` and the properties `msg` (the error text),
`cmd_topic`, `cmd_service` and `cmd_type` taken from the request. An error wrapping
`router.ErrNotReported` (the same value as `adapter.ErrNotReported`) SHALL be logged at warning
level, any other error at error level. `WithSilentErrors()` SHALL suppress the reply while keeping
the log. An error reply addressed to a device resource SHALL be marked with the skip storage
strategy so device errors are not persisted by the hub.

#### Scenario: processor returns an error
- **WHEN** a message processor returns an error and silent errors are not enabled
- **THEN** an `evt.error.report` is replied with `msg`, `cmd_topic`, `cmd_service` and `cmd_type` properties

#### Scenario: state not reported yet
- **WHEN** a message processor returns an error wrapping `router.ErrNotReported`
- **THEN** the error is logged at warning level and the `evt.error.report` is still replied

#### Scenario: device error not stored
- **WHEN** the error reply resolves to a `rt:dev` address and the payload has no storage strategy
- **THEN** the payload is tagged with the skip storage strategy before publishing
