# manifest.AuthResponse is dead code

`manifest/manifest.go:155` declares `AuthResponse {status, error_text, error_code}`, which nothing
in cliffhanger or in the edge adapters under `~/proj/go/edge-*-adapter` uses; the auth handlers
reply with `app.AuthenticationReport` (`app/model.go:18`) instead. Keeping two shapes for the same
`evt.auth.status_report` payload invites an adapter to use the stale one (it lacks the FHX
`errors` field).

Close it by deprecating it (`// Deprecated: use app.AuthenticationReport`) and removing it in the
next breaking release.
