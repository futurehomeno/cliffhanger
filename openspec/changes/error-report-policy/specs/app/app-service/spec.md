## ADDED Requirements

### Requirement: A Failed Outcome Fills The Error Text
The typed app reports SHALL keep their event names and JSON field names exactly as they are:
`evt.app.config_report` and `evt.app.uninstall_report` carry `{op_status, op_error, app_state}`,
`evt.app.config_action_report` carries `{op, op_status, next, error_code, error_text}` and
`evt.auth.status_report` carries `{status, error_text, error_code, errors}`. No field SHALL be
renamed, removed or given another meaning. Whenever a command answered by one of these reports
fails, the report's error text field — `op_error` in the configuration and uninstall reports,
`error_text` in the others — SHALL be filled, and `op_status` SHALL be `error` where the report has
one. For `cmd.auth.login` and `cmd.auth.set_tokens` the outcome is a failure when the operation
returned an error, and also when it returned none but the auth state after it is
`NOT_AUTHENTICATED`, `ERROR` or `LOST`; `error_text` is then `failed to login` or
`failed to authorize`. An auth state of `IN_PROGRESS` is not a failure. No answer SHALL change
between a typed report and `evt.error.report`: `evt.error.report` stays the answer where the other
requirements of this capability give it today, among them a credentials payload that cannot be
read and a busy lock.

#### Scenario: check after authorizing fails
- **WHEN** `cmd.auth.set_tokens` is handled, `Authorize` returns no error and the auth state after it is `NOT_AUTHENTICATED`
- **THEN** `evt.auth.status_report` carries `status` `NOT_AUTHENTICATED` and `error_text` `failed to authorize`

#### Scenario: login refused without an error
- **WHEN** `Login` returns no error and the auth state after it is `ERROR`
- **THEN** `evt.auth.status_report` carries `error_text` `failed to login`, as well as the deprecated `errors` field

#### Scenario: login still running
- **WHEN** `Login` returns no error and leaves the auth state `IN_PROGRESS`
- **THEN** `evt.auth.status_report` carries an empty `error_text`

#### Scenario: report shape unchanged
- **WHEN** `Configure` returns an error for `cmd.config.extended_set`
- **THEN** `evt.app.config_report` is published with exactly the fields `op_status` `error`, a filled `op_error` and `app_state`, and no `evt.error.report` is published
