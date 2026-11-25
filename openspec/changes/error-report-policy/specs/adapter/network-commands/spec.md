## MODIFIED Requirements

### Requirement: Command Error Reporting
Any adapter command handler that returns an error SHALL cause the router to log it and reply with an
`evt.error.report` string message as requirement "Error Reporting Contract" of `messaging/routing`
states: addressed to the request's `resp_to` topic when set, otherwise to the request's address with
message type `evt`, carrying the error code as value, the request's UID as correlation ID, and the
`msg`, `cmd_topic`, `cmd_service` and `cmd_type` properties identifying the failed request. A
malformed payload SHALL be reported the same way as a failed operation, with the code `bad_request`.

#### Scenario: Malformed payload
- **WHEN** `cmd.thing.delete` arrives with a payload that is not a string map
- **THEN** an `evt.error.report` with the value `bad_request` is returned whose `msg` property explains that the provided address has an incorrect format
