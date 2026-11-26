# The content of a secrets file never reaches an error or a log line

## Why

When a secrets file does not unmarshal, `loadFile` wraps the `encoding/json` error with `%w`, and
`loadData` logs that error when it falls back to the backup. Most `encoding/json` errors name only a
character or a type, but some quote the value they failed on:

- a number that does not fit an integer or float field (`cannot unmarshal number 98765432109876543210
  into ... of type int`),
- a field tagged `,string` (`invalid use of ,string struct tag, trying to unmarshal "..." into int`),
- a type with its own `UnmarshalJSON` or `UnmarshalText` that quotes its input, such as `time.Time`
  (`parsing time "..." as ...`).

A value of a secrets file that lands in such a field is then printed in the error an adapter logs.
A file cut short by a power cut gives a syntax error, which quotes nothing (measured with Go 1.26.1).

## What Changes

- A secrets store (`NewSecrets`, `NewCanonicalSecrets`) that cannot unmarshal its data or backup file
  returns an error naming the path, the kind of failure (not valid JSON, or a value that does not fit
  the model) and the file size in bytes. The text of the `encoding/json` error is dropped.
- The backup fallback log line carries that same error, so it is clean too.
- Configuration and state stores are unchanged.

## Impact

- Downstream edge adapters: changed behaviour, no API change. Every adapter that loads a secrets
  store (cliffhanger v1.3.3 and later: easee, hoiax, hue, mill, netatmo, sensibo, somfy, sonos,
  tibber, zaptec, and CloudBridge, ecollector, energy-guard) logs the shorter error after a bump.
- The returned error no longer wraps the `encoding/json` error, so `errors.As` for
  `*json.SyntaxError` or `*json.UnmarshalTypeError` on a secrets store's `Load` error no longer
  matches. No adapter checks a storage `Load` error that way.
- Code: `storage/storage.go`, spec `storage/persistence`.
