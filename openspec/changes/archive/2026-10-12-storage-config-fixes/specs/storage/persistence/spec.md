## MODIFIED Requirements

### Requirement: Key-Value Database
`Database` SHALL be an in-memory key-value store backed by buntdb with append-only persistence,
opened at `<workdir>/<filename>.db` with the work directory created if missing. It SHALL use the
`Always` sync policy and enable background compaction. Unless overridden by options, the filename
SHALL be `data`, the compaction size threshold SHALL be 2 MiB and the compaction percentage SHALL be
100 — a percentage of 0 would degenerate into rewriting the whole database after any write. The
`workdir` argument SHALL take precedence over a `WithWorkdir` option passed by the caller. Values
SHALL be stored JSON-encoded under the composite key `<bucket>:<key>`. `Delete` SHALL be idempotent:
deleting a key that does not exist SHALL return nil. `Start` SHALL be a no-op and `Stop` SHALL close
the underlying file.

#### Scenario: round trip
- **WHEN** a value is set in a bucket and read back through `Get`
- **THEN** `Get` reports `ok` true and unmarshals the stored JSON into the caller's value

#### Scenario: missing key
- **WHEN** `Get` is called for a key that does not exist
- **THEN** it returns `ok` false and a nil error

#### Scenario: deleting a missing key
- **WHEN** `Delete` is called for a key that does not exist
- **THEN** it returns nil

### Requirement: Key Enumeration
`Keys` SHALL return every key of a bucket in ascending order with the `<bucket>:` prefix stripped.
`KeysFrom` SHALL return the keys from the given key inclusive to the end of the bucket. `KeysBetween`
SHALL return the keys from `from` inclusive to `to` exclusive. All three SHALL be confined to the keys
starting with `<bucket>:`, matching the bucket name literally — `*` and `?` in a bucket name have no
special meaning. Keys MAY contain `:`, so a plain bucket also enumerates the keys of a
`NewDomainDatabase` whose domain equals that bucket name; a domain name SHALL NOT be reused as a plain
bucket name on the same database.

#### Scenario: bounded range
- **WHEN** a bucket holds `test_key1` through `test_key6` and `KeysBetween(bucket, "test_key2",
  "test_key5")` is called
- **THEN** it returns `test_key2`, `test_key3` and `test_key4`

#### Scenario: open-ended range does not leak into a sibling bucket
- **WHEN** `KeysFrom` is called on `test_bucket_a` while `test_bucket_b` also holds keys
- **THEN** only keys of `test_bucket_a` are returned

#### Scenario: glob characters in a bucket name
- **WHEN** buckets `a*`, `a?` and `ab` hold keys and `Keys("a*")` is called
- **THEN** only the keys of bucket `a*` are returned

#### Scenario: key containing a colon
- **WHEN** a bucket holds the key `pt:j1/ad:1`
- **THEN** `Keys` returns `pt:j1/ad:1` unchanged

### Requirement: Encryption Key Management
`GetKey(path)` SHALL return the key stored at the given path, and SHALL generate, persist and return
a fresh 32-byte (64 hex character) key only when the file does not exist. In both cases the key file
SHALL be chmodded to `0600`. A new key SHALL be written to a temporary file and linked into place, so
the key file never holds a partial key and concurrent first runs all return the key that was linked.
Any other read error, a chmod failure, or a stored key that is not hex
decoding to 16, 24 or 32 bytes SHALL be returned as an error, leaving the file untouched.
`GenerateKey(size)` SHALL return `size` cryptographically random bytes encoded as a lowercase hex
string.

#### Scenario: first run
- **WHEN** `GetKey` is called for a path that does not exist
- **THEN** a new key is written to that path with mode `0600` and returned

#### Scenario: subsequent runs
- **WHEN** `GetKey` is called again for the same path
- **THEN** the previously generated key is returned unchanged

#### Scenario: invalid stored key
- **WHEN** the key file is empty, has a trailing newline, is not hex or decodes to a length other
  than 16, 24 or 32 bytes
- **THEN** `GetKey` returns an error and the file is not overwritten

#### Scenario: unreadable key file
- **WHEN** the key file exists but cannot be read
- **THEN** `GetKey` returns an error and the file is not overwritten

#### Scenario: concurrent first run
- **WHEN** several callers run `GetKey` at once for a path that does not exist
- **THEN** they all return the key stored in the file
