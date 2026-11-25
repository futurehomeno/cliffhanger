## ADDED Requirements

### Requirement: Secrets Stay Out Of Errors And Logs
When a secrets store created by `NewSecrets` or `NewCanonicalSecrets` cannot unmarshal its data file
or its backup, the error returned by `Load` and any line the storage logs about that failure SHALL NOT
contain any part of the file's content. The error SHALL name the file path, the kind of failure — not
valid JSON, or a value that does not fit the model — and the file size in bytes, and nothing else.
Configuration and state stores are not affected.

#### Scenario: file cut short
- **WHEN** the secrets file holds valid JSON cut in half after a fake token
- **THEN** `Load` returns an error naming the path, "not valid JSON" and the size
- **AND** the error does not contain the fake token

#### Scenario: value that does not fit the model
- **WHEN** the secrets file is valid JSON but a fake secret sits in a field of another type, as a
  number too large for an integer field, a string in a `,string` integer field or a string in a
  `time.Time` field
- **THEN** `Load` returns an error naming the path, a value that does not fit the model and the size
- **AND** the error does not contain the fake secret

#### Scenario: fallback to the backup
- **WHEN** the secrets data file does not unmarshal and holds a fake secret, and its backup is valid
- **THEN** `Load` succeeds from the backup
- **AND** the line logged about the data file does not contain the fake secret
