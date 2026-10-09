## MODIFIED Requirements

### Requirement: Reporting Tasks Skip A State Not Reported Yet
A controller SHALL return an error wrapping `adapter.ErrNotReported` when the device has not reported
the requested state yet. The presence, numericsensor, scenectrl and battery periodic reporting tasks
SHALL skip such an error without logging it and SHALL keep logging every other error.

#### Scenario: state not reported yet
- **WHEN** a reporting task's controller returns an error wrapping `adapter.ErrNotReported`
- **THEN** no report is sent and nothing is logged

#### Scenario: other failure
- **WHEN** a reporting task's controller returns any other error
- **THEN** the task logs it as before
