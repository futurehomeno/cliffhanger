---
name: port_to_cliffhanger_1.3
description: Upgrade a Futurehome edge adapter from cliffhanger v1.2.10 to the v1.3.0 line (the `develop` branch), or build a new adapter directly on 1.3. Use when asked to bump an adapter to cliffhanger 1.3, adopt the new common blocks (ConnectivityChecker, config.Service, auth.Authenticator, secrets storage, stream.Supervisor, app flows, thing-sync helpers), or align an adapter with the extracted-common-block architecture. For a from-scratch port off legacy fimpgo/v0.x, do the base port with port_to_cliffhanger_1.2 first, then apply this on top.
---

# Upgrade an edge adapter from cliffhanger v1.2.10 to v1.3.0

**1.3.0 is not a rewrite — it is the same architecture with the per-adapter boilerplate
extracted into the framework.** Every adapter hand-rolled the same `Check()` state machine,
OAuth refresh, config service, thing-sync wrapper, credential storage and push-transport
supervisor. 1.3 (cliffhanger `develop`, waves 1–4 in #185/#186/#187/#189) ships these as
reusable primitives. Porting to 1.3 = **deleting your copies and wiring the framework ones**,
without changing any FIMP wire behaviour.

Read [references/upgrade-to-1.3.md](references/upgrade-to-1.3.md) first — it is the full API
catalog, the per-adapter Check() fit table, and the state/auth/config mapping. This file is the
procedure. The 1.2 base architecture (package layout, packaging, tooling, FIMP compatibility)
is unchanged — see the `port_to_cliffhanger_1.2` skill for it; do **not** repeat that work.

Cliffhanger source for 1.3 is the `develop` branch of the cliffhanger checkout (there is no
v1.3.0 tag yet; `git describe` shows `v1.2.10-N-g<sha>`). Reference adapters still on 1.2.x are
the sibling `edge-*-adapter` repos in this workspace (here under `~/proj/go/`; adjust to your
checkout); `core-energy-guard` is a **core service on v1.2.7**, not an edge adapter and not a
1.3 template.

## Golden rule: wire-compatibility is unchanged

None of the 1.3 primitives change FIMP identity. Resource name, service names, service
addresses (`ThingSeed.CustomAddress`), interfaces and value types must stay byte-identical
across the bump — the framework code produces the same inclusion/exclusion/evt messages your
hand-rolled code did. Diff a captured inclusion report + evt stream before/after the upgrade.

## Procedure

1. **Bump the dependency and toolchain.** cliffhanger `develop` requires **`go 1.26`** — first
   `go mod edit -go=1.26` and bump CI/runtime images to Go 1.26, else module resolution fails
   before any API work. Then `go get github.com/futurehomeno/cliffhanger@<develop-pseudo-version>`,
   `go mod tidy`. Build; the compile errors map the API deltas (mostly additive — the 1.2 surface
   is preserved, so nothing you already use breaks).

2. **Adopt the typed HTTP errors first** (everything else keys on them). Make the API client
   return `httpclient.ErrUnauthorized` (401/403), `httpclient.ErrTooManyRequests` /
   `*httpclient.TooManyRequestsError{RetryAfter}` (429), plain errors otherwise — via
   `httpclient.ErrorFromResponse(resp)`. This is the contract `ConnectivityChecker`,
   `Authenticator` and the sync guards all consume. (refs §2)

3. **Replace hand-rolled `Check()` with `app.ConnectivityChecker`** — the biggest deletion.
   Build it with a `probe func() error` (a cheap real API call returning the sentinels above),
   the lifecycle, the adapter as `ConnectivityReporter`, and a `CheckerConfig`
   (`Interval`, `RecheckBackoff`, `MaxRechecks`, `RateLimitDelay`). It maps probe outcome →
   lifecycle exactly like the old machine (success → Authenticated/Connected + repair;
   401 → Lost/Disconnected; 429 → untouched + reschedule; transient → disconnect only after
   `MaxRechecks`). Wire `checker.CheckNow` as the `Authorize` callback and `checker.Cancel` on
   logout/reset. **Check the fit table first** (refs §3): sensibo/sonos/mill/netatmo fit
   directly; zaptec/easee/tibber do **not** (event-driven / no-op / no-probe) — leave their
   `Check()` alone. Customization hooks (`AuthLossState`, `Authorized`) cover the
   Lost-vs-NotAuthenticated and teardown-guard nuances (refs §3).

4. **Replace the OAuth refresh loop with `auth.Authenticator`** (username/password→JWT and
   authorization-code refresh). Provide a `CredentialsStore` — back it by the **secrets
   store** from step 5 (do step 5 first, or reorder, so credentials have a single source of
   truth), not the plain config — and a
   `TokenExchanger` (your client), set `RefreshLead`, `Backoff` (a `backoff.Stateful`, e.g.
   `backoff.NewTolerantFixed`), `UnauthorizedGrace`, and `OnAuthLoss`. Attach the token with
   `auth.Transport` (a `http.RoundTripper`) instead of a hand-written bearer injector; use
   `auth.TokenExpirationDate` for JWT expiry. (refs §4). Hub-proxied / plain-API-key auth
   (adax/sensibo) does **not** use this — keep it.

5. **Move credentials into secrets storage.** Split secrets out of the world-readable
   `config.json` into `storage.NewSecrets(...)` / `NewCanonicalSecrets(...)` (0640
   `data/secrets.json`). Migrate the legacy token fields once into the secrets file. (refs §6)

6. **Adopt `config.Service[C]`** (generic, thread-safe) in place of the hand-rolled config
   service: `Update`/`Persist`/`Reset`/`Migrate`/`DefaultStore`/`PublicModel`, plus `config.Get`
   / `config.GetDuration` accessors and `config.BackoffConfig`. Keep your `Config` struct and its
   JSON shape identical. (refs §5)

7. **Collapse lifecycle bundles and thing-sync.** Use `lifecycle.MarkRunning()` /
   `MarkNotConfigured()` and `SetConnAndAuthState` for atomic auth-loss bundles instead of four
   separate setters. Replace the fetch→filter→map→`EnsureThings` wrapper with
   `adapter.SeedsFromSelection` + `adapter.SyncThings` (shared anti-wipe guard,
   `ErrIncompleteFetch`); replace a hand-rolled capability-drift rebuild (sensibo `reconcile.go`)
   with `adapter.RebuildChangedThings`. (refs §7)

8. **Adopt the remaining blocks where they apply:** `stream.Supervisor` for a push transport's
   reconnect/backoff lifecycle (easee SignalR, sonos GENA, zaptec AMQP) via a `root.Service`;
   `app.Reset`/`app.Logout`/`app.Authorize`/`app.ConfigModel` flow helpers; `bootstrap.EdgeRouting`
   /`EdgeTasks` bundles; `router.DefaultLogStats(prefixes...)` for the stats callback;
   `utils.Throttle` for log throttling and `utils.Timestamped` for last-value+staleness. (refs §8)

9. **Delete the replaced code** — the old `Check()`/`scheduleRecheck`/`cancelRecheck`, the
   bespoke authenticator/transport, the hand-rolled config service, the sync wrapper, the
   credential-in-config plumbing. Less code is the point of this bump.

10. **Verify.** `go vet ./...`, `golangci-lint run`, `make test` (coverage gate 75%, target
    >80%), `make build-arm`, `make deb-arm`. Then **prove wire-compatibility**: diff the
    inclusion report + a representative evt stream against the pre-bump capture. Finally, review
    logs — invoke the `improve_logs` skill if available, else check them by hand.

## Output expectations

- Adapter compiles and runs on cliffhanger 1.3 (`develop`), with the hand-rolled Check/auth/
  config/sync/secrets code replaced by framework primitives and net lines **removed**.
- Identical FIMP behaviour: same addresses, same reports, no re-inclusion for existing users.
- Adapters that don't fit a primitive (zaptec/easee/tibber Check, hub-proxy auth) are left as-is
  with a one-line note why.
- CI green.
