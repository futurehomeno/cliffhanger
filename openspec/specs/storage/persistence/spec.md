# Persistence Specification

## Purpose
Three packages carry everything an adapter keeps across restarts. `storage` persists a JSON document
— configuration, credentials or opaque state — with a defaults file, a rolling backup and mode-aware
writes. `database` is an in-memory key-value store with append-only persistence to disk, used for
high-churn data where rewriting a whole JSON file per change would be wasteful. `security` provides
the AES-GCM primitives and the on-disk key an adapter uses to encrypt values it must not store in
clear text.

## Requirements

### Requirement: Storage Layout Variants
`Storage[T]` SHALL expose `Model`, `Load`, `Save` and `Reset` over a caller-owned model, which is
expected to be a pointer. The constructor chosen SHALL determine the file layout and whether a
defaults file exists:

- `New` — Thingsplex layout: data at `<workDir>/data/<name>`, defaults at `<workDir>/defaults/<name>`.
- `NewCanonical` — data at `<workDir>/<name>`, defaults at `<defaultsDir>/<name>`.
- `NewState` and `NewCanonicalState` — the same two data paths with **no** defaults file.
- `NewSecrets` and `NewCanonicalSecrets` — the same two data paths, no defaults file, and secret file
  permissions.

The backup path SHALL always be the data path with the `.bak` suffix appended. Every public method
SHALL be serialized by the storage's own mutex.

#### Scenario: Thingsplex layout
- **WHEN** `New(model, workDir, "config.json")` is saved
- **THEN** the document is written to `<workDir>/data/config.json`
- **AND** its defaults are read from `<workDir>/defaults/config.json`

### Requirement: File Permissions
A configuration or state store SHALL write its data file with mode `0644`; a secrets store created by
`NewSecrets` or `NewCanonicalSecrets` SHALL write it with `0640`. `Save` SHALL enforce the mode on
every write regardless of the umask and regardless of the mode a pre-existing file carried, and SHALL
re-apply the mode to the backup file after renaming the data file over it — otherwise a secrets file
predating the tightened mode would leave credentials world-readable in the `.bak`.

#### Scenario: pre-existing loose permissions are tightened
- **WHEN** an existing data file has been chmodded to `0666` and `Save` is called
- **THEN** the file is left at `0644` for a configuration store and `0640` for a secrets store

#### Scenario: backup inherits the tightened mode
- **WHEN** a secrets store saves twice over a file created with looser permissions
- **THEN** the resulting `.bak` file carries mode `0640`

### Requirement: Load Precedence
`Load` SHALL apply the defaults file first and then unmarshal the data file over it, so a data file
that omits a field inherits the default. When the data file cannot be read or unmarshalled and a
defaults file exists, `Load` SHALL log the failure, keep the defaults and return nil. When no
defaults file exists, that failure SHALL be returned.

#### Scenario: partial data file
- **WHEN** the defaults declare three settings and the data file declares one
- **THEN** the loaded model carries the data file's value for that one setting and the defaults for
  the other two

#### Scenario: unreadable data file with defaults present
- **WHEN** the data file contains invalid JSON and a defaults file exists
- **THEN** `Load` returns nil and the model holds the default values

### Requirement: Load Falls Back To The Backup
When the data file is missing, `Load` SHALL treat an existing backup file as the data file, and when
the data file exists but fails to load, `Load` SHALL retry from the backup and return successfully if
the backup loads. This covers a crash between the backup rename and the rewrite that follows it in
`Save`.

#### Scenario: only the backup survives
- **WHEN** a store with no defaults file has its data file removed but its `.bak` intact
- **THEN** `Load` succeeds and the model holds the values from the backup

### Requirement: Load With Nothing Present
When a store has a defaults path configured and neither the data file, the backup nor the defaults
file exists, `Load` SHALL return an error naming both the data and the defaults path. When a store
has no defaults path — a state or secrets store — and no data file exists, `Load` SHALL succeed and
leave the model untouched.

#### Scenario: first start of a secrets store
- **WHEN** `Load` is called on a secrets store before anything was ever saved
- **THEN** it returns nil

#### Scenario: missing configuration entirely
- **WHEN** `Load` is called on a defaults-backed store with no files on disk at all
- **THEN** it returns an error listing the data path and the defaults path

### Requirement: Atomic-Enough Save
`Save` SHALL create the data directory (mode `0755`) if missing, marshal the model as tab-indented
JSON, rename any existing data file to the backup path rather than copying it, and then write and
`fsync` the new data file. A failure to produce the backup SHALL abort the save before the data file
is rewritten.

#### Scenario: backup rotation
- **WHEN** a store is saved twice
- **THEN** the `.bak` file holds the contents written by the first save

### Requirement: Reset Semantics
`Reset` SHALL remove both the data file and the backup file, and SHALL then restore the in-memory
model according to the store kind:

- A defaults-backed store SHALL zero the model and reload the defaults file, so fields present in the
  removed data but absent from the defaults do not survive. It SHALL fail before deleting anything if
  the configured defaults file is missing.
- A secrets store SHALL zero the model, so a logout does not keep serving stale credentials.
- A store with no defaults and no secret flag SHALL keep its in-memory model, because there is
  nothing to reload and callers hold on to it past the reset.

Zeroing SHALL clear the pointed-to struct in place rather than nil the pointer, so cached pointers are
wiped and the model stays a valid target for a later `Load` or mutation.

#### Scenario: defaults replace rather than merge
- **WHEN** a store whose defaults declare only `SettingA` is reset after loading a data file with
  `SettingA`, `SettingB` and `SettingC`
- **THEN** the model holds the default `SettingA` and zero values for `SettingB` and `SettingC`

#### Scenario: secrets reset then re-login
- **WHEN** a secrets store is reset and the caller then assigns a fresh token to `Model()` and saves
- **THEN** the assignment does not panic, the save succeeds and a subsequent `Load` returns the fresh
  token

#### Scenario: state store keeps its model
- **WHEN** `NewCanonicalState` is reset
- **THEN** the persisted file is removed but `Model()` still holds its values

### Requirement: Key-Value Database
`Database` SHALL be an in-memory key-value store backed by buntdb with append-only persistence,
opened at `<workdir>/<filename>.db` with the work directory created if missing. It SHALL use the
`Always` sync policy and enable background compaction. Unless overridden by options, the filename
SHALL be `data`, the compaction size threshold SHALL be 2 MiB and the compaction percentage SHALL be
100 — a percentage of 0 would degenerate into rewriting the whole database after any write. The
`workdir` argument SHALL take precedence over a `WithWorkdir` option passed by the caller. Values
SHALL be stored JSON-encoded under the composite key `<bucket>:<key>`. `Start` SHALL be a no-op and
`Stop` SHALL close the underlying file.

#### Scenario: round trip
- **WHEN** a value is set in a bucket and read back through `Get`
- **THEN** `Get` reports `ok` true and unmarshals the stored JSON into the caller's value

#### Scenario: missing key
- **WHEN** `Get` is called for a key that does not exist
- **THEN** it returns `ok` false and a nil error

### Requirement: Expiry
`Set` SHALL store a value without expiry. `SetWithExpiry` SHALL attach a TTL only when the supplied
duration is greater than zero; a zero or negative duration SHALL behave exactly like `Set`.

#### Scenario: expired entry disappears
- **WHEN** a value is written with a TTL and read after that TTL has elapsed
- **THEN** `Get` reports `ok` false

### Requirement: Key Enumeration
`Keys` SHALL return every key of a bucket in ascending order with the `<bucket>:` prefix stripped.
`KeysFrom` SHALL return the keys from the given key inclusive to the end of the bucket. `KeysBetween`
SHALL return the keys from `from` inclusive to `to` exclusive. All three SHALL be confined to the
requested bucket.

#### Scenario: bounded range
- **WHEN** a bucket holds `test_key1` through `test_key6` and `KeysBetween(bucket, "test_key2",
  "test_key5")` is called
- **THEN** it returns `test_key2`, `test_key3` and `test_key4`

#### Scenario: open-ended range does not leak into a sibling bucket
- **WHEN** `KeysFrom` is called on `test_bucket_a` while `test_bucket_b` also holds keys
- **THEN** only keys of `test_bucket_a` are returned

### Requirement: Corrupted File Recovery
When the data file cannot be opened because buntdb reports it invalid or truncated, the database
SHALL recover rather than fail: it SHALL load what it can from the corrupted file into a temporary
in-memory database (logging, not failing, on a partial load), write the salvaged contents to
`<filename>.db.recovered` truncating any stale file of that name, delete a previous
`<filename>.db.corrupted`, rename the corrupted file to `<filename>.db.corrupted`, rename the
recovered file into place and reopen it.

#### Scenario: truncated tail
- **WHEN** the last record of a data file holding six entries is truncated and the database is opened
- **THEN** the database opens successfully with the five intact entries
- **AND** the original file is preserved as `<filename>.db.corrupted`

#### Scenario: stale recovery artefact
- **WHEN** a larger `<filename>.db.recovered` left by an earlier failed recovery is present
- **THEN** its trailing bytes do not appear in the recovered database

### Requirement: Domain Namespacing And Reset
`NewDomainDatabase(domain, db)` SHALL wrap a database so that every bucket is transparently prefixed
with `<domain>:` for reads, writes, deletes and key enumeration, letting several components share one
file without colliding. `Reset` SHALL delete all entries of the underlying database and shrink the
file afterwards; it is not scoped to a domain.

#### Scenario: namespaced access
- **WHEN** two domain databases with different domains write the same bucket and key
- **THEN** each reads back its own value

#### Scenario: reset empties the store
- **WHEN** `Reset` is called after several writes
- **THEN** `Keys` returns nothing for the previously populated buckets

### Requirement: Encryption Key Management
`GetKey(path)` SHALL return the key stored at the given path, and SHALL generate, persist and return
a fresh 32-byte (64 hex character) key when the file cannot be read. In both cases the key file SHALL
be chmodded to `0600`. `GenerateKey(size)` SHALL return `size` cryptographically random bytes encoded
as a lowercase hex string.

#### Scenario: first run
- **WHEN** `GetKey` is called for a path that does not exist
- **THEN** a new key is written to that path with mode `0600` and returned

#### Scenario: subsequent runs
- **WHEN** `GetKey` is called again for the same path
- **THEN** the previously generated key is returned unchanged

### Requirement: Value Encryption
`Encrypt` SHALL interpret the key as a hex string, encrypt with AES-GCM using a freshly generated
random nonce, prepend that nonce to the ciphertext and return the result hex-encoded. `Decrypt` SHALL
be its exact inverse and SHALL fail rather than panic on malformed input: a non-hex key, a non-hex
ciphertext, a ciphertext shorter than the GCM nonce, or a payload that fails authentication SHALL all
return an error.

#### Scenario: round trip
- **WHEN** a string is encrypted and then decrypted with the same key
- **THEN** the original string is returned

#### Scenario: nonces are not reused
- **WHEN** the same plaintext is encrypted twice with the same key
- **THEN** the two ciphertexts differ

#### Scenario: truncated ciphertext
- **WHEN** `Decrypt` is given a hex string shorter than the nonce
- **THEN** it returns an error naming the required nonce length
