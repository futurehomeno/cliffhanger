# A configuration store's load error can still quote a value

`secrets-not-in-log` cleans only the load error of a secrets store (`NewSecrets`,
`NewCanonicalSecrets`). A configuration or state store still wraps the `encoding/json` error, which
quotes the value for a number too large for its field, a `,string` field, or a type whose
`UnmarshalJSON` quotes its input (`time.Time`). Adapters that kept tokens or passwords in
`config.json` before they moved them to `secrets.json` load that file through `storage.New` or
`NewCanonical` until the migration has run, and the error is logged by `load` (the defaults fallback)
and by the adapter.

Reproduce: write `{"Expiry": "FAKE-SECRET"}` to `data/config.json` of a `storage.New` store whose
model has `Expiry time.Time`, call `Load` without a defaults file; the error contains `FAKE-SECRET`.

Open question: apply the same error to every store (loses the field name and json detail that help
when a user edits `config.json` by hand), or keep the detail for configuration stores.

Closes when that is decided and, if so, done.
