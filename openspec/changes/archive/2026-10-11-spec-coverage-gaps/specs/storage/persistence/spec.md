## ADDED Requirements

### Requirement: Invalid Defaults File
When the defaults file exists but cannot be read or unmarshalled, `Load` SHALL return that error
without consulting the data file or the backup. A defaults-backed `Reset` hitting the same failure SHALL return
the error after the data and backup files were removed, leaving the model zeroed.

#### Scenario: Corrupted defaults file
- **WHEN** the defaults file contains invalid JSON and a valid data file exists
- **THEN** `Load` returns an error

### Requirement: Interrupted Database Recovery Resumes
When `<filename>.db` is missing but `<filename>.db.recovered` exists — a crash between renaming the
corrupted file away and renaming the recovered one into place — opening the database SHALL first
rename the recovered file to `<filename>.db` and open it, so the salvaged data is not lost.

#### Scenario: Crash between the recovery renames
- **WHEN** the database is opened with only `<filename>.db.recovered` on disk
- **THEN** the database opens with the recovered entries and `<filename>.db.recovered` no longer
  exists

### Requirement: Undecodable Stored Value
`Get` SHALL return `ok` false and an error when the stored JSON cannot be unmarshalled into the
caller's value, so a type mismatch is never mistaken for a missing key nor for a successful read.

#### Scenario: Reading into the wrong type
- **WHEN** a string was stored and `Get` is called with a pointer to an integer
- **THEN** it returns `ok` false and a non-nil error

## MODIFIED Requirements

### Requirement: Value Encryption
`Encrypt` SHALL interpret the key as a hex string, encrypt with AES-GCM using a freshly generated
random nonce, prepend that nonce to the ciphertext and return the result hex-encoded. The decoded key
length SHALL select AES-128, AES-192 or AES-256 (16, 24 or 32 bytes); any other length SHALL make
both `Encrypt` and `Decrypt` return an error. `Decrypt` SHALL be the exact inverse of `Encrypt` and
SHALL fail rather than panic on malformed input: a non-hex key, a non-hex ciphertext, a ciphertext
shorter than the GCM nonce, or a payload that fails authentication SHALL all return an error.
`Encrypt` SHALL likewise return an error for a non-hex key.

#### Scenario: round trip
- **WHEN** a string is encrypted and then decrypted with the same key
- **THEN** the original string is returned

#### Scenario: nonces are not reused
- **WHEN** the same plaintext is encrypted twice with the same key
- **THEN** the two ciphertexts differ

#### Scenario: truncated ciphertext
- **WHEN** `Decrypt` is given a hex string shorter than the nonce
- **THEN** it returns an error naming the required nonce length

#### Scenario: key of an unsupported length
- **WHEN** `Encrypt` is given a valid hex key that decodes to 10 bytes
- **THEN** it returns an error
