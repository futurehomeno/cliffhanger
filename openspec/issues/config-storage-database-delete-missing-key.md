# database: Delete of a missing key returns an error

`database.Delete` (database/database.go:238) passes buntdb's `ErrNotFound` through as an error when
the key does not exist, so deleting an absent key fails. Callers such as
`virtualmeter.Storage.DeleteDevice` (adapter/service/virtualmeter/storage.go:76) then report a
failure for an idempotent operation, and the only way to tell "not found" apart is to import
`github.com/tidwall/buntdb` and check `buntdb.ErrNotFound` — a third-party sentinel leaking through
the package's API. No spec requirement or test covers it.

Close by deciding the contract: either make `Delete` idempotent (treat `ErrNotFound` as success) or
export a `database.ErrNotFound`; then spec it under Key-Value Database and add a test.
