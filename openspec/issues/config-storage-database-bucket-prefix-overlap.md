# database: bucket enumeration matches nested and glob-like buckets

`Keys` (database/database.go:267) enumerates with the buntdb glob pattern `<bucket>:*`, and
`KeysFrom` ranges up to `<bucket>:\xff`. Neither stops at the next `:`, so `Keys("dev")` on a plain
database also returns the keys of a `NewDomainDatabase("dev", db)` (as `bucket:key`), and a bucket
name containing `*` or `?` matches other buckets. The spec's "confined to the requested bucket"
holds only when no bucket name is a `:`-prefix of another and names contain no glob characters.

Close by either documenting the restriction on bucket/domain names in the Key Enumeration
requirement, or escaping the pattern and filtering out keys with a further `:` after the prefix.
