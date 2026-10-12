# Storage and config fixes

## Why

Four open issues in `database`, `security` and `config`: a non-idempotent delete, bucket enumeration
that treats the bucket as a glob, a key file that could be overwritten or served invalid, and a
telemetry getter that leaks aliased state.

## What Changes

- `database.Delete` of a missing key returns nil.
- `Keys`/`KeysFrom` scan the `<bucket>:` key range instead of a glob, so `*`/`?` in a bucket are literal.
  Keys may contain `:`, so a domain name must not be reused as a plain bucket on the same database
  (documented, not filtered: virtual meter keys are topics with `:`).
- `security.GetKey` generates a key only when the file does not exist, returns read/chmod errors and
  rejects a stored key that is not hex for 16/24/32 bytes.
- `DefaultStore.Telemetry` returns a deep clone.

## Impact

`database/database.go`, `security/encryption.go`, `config/default.go`.
