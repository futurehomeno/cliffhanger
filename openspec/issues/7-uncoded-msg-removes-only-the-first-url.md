# 7. An uncoded msg loses only the first URL of its chain

## What happened

The `messaging/routing` delta of change `error-report-policy` says that `msg` of an `internal` error is
the error chain "with every URL removed". `design.md` and the code (`router/error.go`, `errorReport`)
rewrite only the first `*url.Error` found with `errors.As`; a second `*url.Error` (for example in an
`errors.Join`) and a URL an adapter writes into its own error text stay in `msg`.

## How to reproduce

Return `errors.Join(&url.Error{Op: "Get", URL: "https://a", Err: e1}, &url.Error{Op: "Get", URL: "https://b", Err: e2})`
from a message processor: `msg` still holds `https://b`.

## What would close it

Either the delta says "the URL of the first `*url.Error` in the chain" (what the design decided), or the
router walks every `*url.Error` in the chain. A URL in an adapter's own text stays the adapter's to
remove (adapter plan).
