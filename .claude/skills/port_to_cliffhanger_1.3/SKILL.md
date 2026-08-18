---
name: port_to_cliffhanger_1.3
description: Upgrade a Futurehome edge adapter to cliffhanger v1.3.4, from either v1.2.10 (adopting the common blocks - ConnectivityChecker, config.Service, auth.Authenticator, secrets storage, stream.Supervisor, app flows, thing-sync helpers) or v1.3.3 (a short delta bump), or build a new adapter directly on 1.3.4. Use when asked to bump an adapter to cliffhanger 1.3 or 1.3.4, adopt the extracted-common-block architecture, or wire the new alarm_system service. For a from-scratch port off legacy fimpgo/v0.x, do the base port with port_to_cliffhanger_1.2 first, then apply this on top.
---

# Upgrade an edge adapter to cliffhanger v1.3.4

Target the **`v1.3.4` tag**: `go get github.com/futurehomeno/cliffhanger@v1.3.4`.

Read [references/upgrade-to-1.3.md](references/upgrade-to-1.3.md) first — it is the API catalog, the
per-adapter `Check()` fit table, the state/auth/config mapping, the **1.3.3 → 1.3.4 delta** (§9) and
the **`alarm_system` service** (§10). This file is the procedure.

## Pick your track

Read the adapter's `src/go.mod`:

| Pinned at | Track | What it is |
|---|---|---|
| v1.2.x or earlier | **A** | Adopt the common blocks. The big one. Ten steps, most of which are deletions. |
| v1.3.0 – v1.3.3 | **B** | A delta bump. Eight steps, no re-architecture. |
| legacy fimpgo / cliffhanger v0.x | — | Do `port_to_cliffhanger_1.2` first, then come back on track A. |

**Do not run track A on an adapter that is already on 1.3.** It has the common blocks; re-porting
them churns working code and risks the wire compatibility below.

## Golden rule: wire-compatibility is unchanged

None of the 1.3 primitives change FIMP identity. Resource name, service names, service addresses
(`ThingSeed.CustomAddress`), interfaces and value types must stay byte-identical across the bump —
the framework code produces the same inclusion/exclusion/evt messages your hand-rolled code did.
Diff a captured inclusion report + evt stream before/after the upgrade.

## Reference adapters

Sibling `edge-*-adapter` repos (here under `~/proj/go/`; adjust to your checkout). Their module file
is `src/go.mod`, not the repo root — that is where the pin is.

| Repo | cliffhanger | Use it for |
|---|---|---|
| **edge-mill-adapter** | 1.3.2 | **The complete cloud-OAuth template**: ConnectivityChecker, `CredentialsStore` over `storage.NewSecrets`, `config.Service[*Config]`, `selection.Store` + `adapter.WithSelection`, probe-folded thing sync. |
| edge-hoiax-adapter | 1.3.3 | Same shape, plus `auth.Authenticator` + `auth.Transport` over the hub auth proxy and a `config.Migration` moving tokens into secrets. |
| **edge-easee-adapter** | 1.3.3 | **The closest model for the common blocks** after mill: `auth.Authenticator`, `selection.Store` + `adapter.WithSelection` + `selection.PrepareManifest`, `adapter.SyncThings`, `bootstrap.EdgeTasks` + `thing.TaskCarCharger`. Also the model for *not* using a ConnectivityChecker — `Check()` is a no-op and a `RefreshToken` task drives connectivity. |
| **edge-zaptec-adapter** | 1.3.3 on its port branches (`develop`/`release_3.1` are still 1.2.10) | **The model for an adapter the primitives do not fit**: hand-rolled `Check()`, event-driven auth loss over an `AuthLost` event instead of `OnAuthLoss`, plain `EnsureThings` with no selection store, explicit per-service reporting tasks. Adopts `config.Service[C]` and `storage.NewSecrets` and stops there. |
| edge-sonos-adapter / edge-tibber-adapter / edge-hue-adapter | 1.3.3 | Further ports; sonos for GENA, hue for the virtual meter. |

`edge-sensibo-adapter`, `edge-netatmo-adapter` and `edge-adax-adapter` are still on v1.2.10 — they
are track A candidates, not templates. `core-energy-guard` is a **core service on v1.2.7**, not an
edge adapter and not a template.

**`stream.Supervisor` has no reference implementation in the fleet.** Both push-transport adapters
still hand-roll their reconnect loop — easee `src/internal/signalr/{client,manager}.go`, zaptec
`src/internal/zaptecapi/amqp/manager.go`. Treat it as an available block to adopt, not as a pattern
to copy from a sibling.

For the `alarm_system` service, the reference implementations are the `feat/alarm-system-service`
branches of edge-zaptec-adapter (pull) and edge-easee-adapter (push) — see refs §10.

---

# Track A — from v1.2.10 to v1.3.4

**1.3 is not a rewrite — it is the same architecture with the per-adapter boilerplate extracted into
the framework.** Every adapter hand-rolled the same `Check()` state machine, OAuth refresh, config
service, thing-sync wrapper, credential storage and push-transport supervisor. 1.3 ships these as
reusable primitives. Porting = **deleting your copies and wiring the framework ones**, without
changing any FIMP wire behaviour.

The 1.2 base architecture (package layout, packaging, tooling, FIMP compatibility) is unchanged —
see the `port_to_cliffhanger_1.2` skill for it; do **not** repeat that work.

1. **Bump the dependency and toolchain.** cliffhanger 1.3 requires **`go 1.26`** — first
   `go mod edit -go=1.26` and bump CI/runtime images to Go 1.26, else module resolution fails before
   any API work. Then `go get github.com/futurehomeno/cliffhanger@v1.3.4`, `go mod tidy`. Build; the
   compile errors map the API deltas. Mostly additive, but a handful **do** break a 1.2.10 adapter —
   they are listed with their fixes in refs §9.1. Expect at minimum: `app.App` now embeds
   `LogProvider` (embed `app.NewLogCapture()`), `telemetry.New` takes a trailing `version string`,
   `telemetry.Telemetry` now has `Start() error`/`Stop() error`, and `debug.Route` panics if
   `debug.InitializeLogger` never ran — which an empty `log_file` in a test fixture guarantees.

2. **Adopt the typed HTTP errors first** (everything else keys on them). Make the API client return
   `httpclient.ErrUnauthorized` (401/403), `httpclient.ErrTooManyRequests` /
   `*httpclient.TooManyRequestsError{RetryAfter}` (429), plain errors otherwise — via
   `httpclient.ErrorFromResponse(resp)`. This is the contract `ConnectivityChecker`, `Authenticator`
   and the sync guards all consume. (refs §2)

3. **Replace hand-rolled `Check()` with `app.ConnectivityChecker`** — the biggest deletion. Build it
   with a `probe func() error` (a cheap real API call returning the sentinels above), the lifecycle,
   the adapter as `ConnectivityReporter`, and a `CheckerConfig` (`Interval`, `RecheckBackoff`,
   `MaxRechecks`, `RateLimitDelay`). It maps probe outcome → lifecycle exactly like the old machine
   (success → Authenticated/Connected + repair; 401 → Lost/Disconnected; 429 → untouched +
   reschedule; transient → disconnect only after `MaxRechecks`). Wire `checker.CheckNow` as the
   `Authorize` callback and `checker.Cancel` on logout/reset. **Check the fit table first** (refs
   §3): sensibo/sonos/mill/netatmo/hoiax fit directly; zaptec/easee/tibber do **not** (event-driven
   / no-op / no-probe) — leave their `Check()` alone. Customization hooks (`AuthLossState`,
   `Authorized`) cover the Lost-vs-NotAuthenticated and teardown-guard nuances. If the probe runs
   through an `auth.Transport`, guard it against the auth sentinels (refs §3, §4) — otherwise a
   backoff window reports a bogus disconnection.

4. **Replace the OAuth refresh loop with `auth.Authenticator`** (username/password→JWT and
   authorization-code refresh). Provide a `CredentialsStore` — back it by the **secrets store** from
   step 5 (do step 5 first, or reorder, so credentials have a single source of truth), not the plain
   config — and a `TokenExchanger` (your client), set `RefreshLead`, `Backoff` (a `backoff.Stateful`,
   e.g. `backoff.NewTolerantFixed`), `UnauthorizedGrace`, and `OnAuthLoss`. Attach the token with
   `auth.Transport` (a `http.RoundTripper`) instead of a hand-written bearer injector; use
   `auth.TokenExpirationDate` for JWT expiry. (refs §4). Hub-proxied / plain-API-key auth
   (adax/sensibo) does **not** use this — keep it.

5. **Move credentials into secrets storage.** Split secrets out of the world-readable `config.json`
   into `storage.NewSecrets(...)` (0640 `data/secrets.json`) — edge adapters use this;
   `NewCanonicalSecrets(...)` is the core-application layout (`workDir/<name>`, no `data/`). Migrate
   the legacy token fields once into the secrets file. (refs §6)

6. **Adopt `config.Service[C]`** (generic, thread-safe) in place of the hand-rolled config service:
   `Update`/`Persist`/`Reset`/`Migrate`/`DefaultStore`/`PublicModel`, plus `config.Get` /
   `config.GetDuration` accessors and `config.BackoffConfig`. Keep your `Config` struct and its JSON
   shape identical. (refs §5)

7. **Collapse lifecycle bundles and thing-sync.** Use `lifecycle.MarkRunning()` /
   `MarkNotConfigured()` (cloud-auth adapters only; apps at `AuthStateNA`/`ConnStateNA` set states
   individually) and `SetConnAndAuthState` for atomic auth-loss bundles instead of four separate
   setters. Replace the fetch→filter→map→`EnsureThings` wrapper with `adapter.SeedsFromSelection` +
   `adapter.SyncThings` (the fetch is owned by the sync, so a failed fetch mutates nothing); replace
   a hand-rolled capability-drift rebuild with `adapter.RebuildChangedThings`; store the user's
   device selection in `selection.Devices` and wire `adapter.WithSelection` so `cmd.thing.delete`
   deselects the device it deletes. (refs §7, worked recipe and its two hazards in §7.1)

8. **Adopt the remaining blocks where they apply:** `stream.Supervisor` for a push transport's
   reconnect/backoff lifecycle (SignalR, GENA, AMQP) via a `root.Service` — no adapter has adopted
   it yet, so port against the framework tests, not a sibling;
   `app.Reset`/`app.Logout`/`app.Authorize`/`app.ConfigModel` flow helpers; `bootstrap.EdgeRouting`
   /`EdgeTasks` bundles; `router.DefaultLogStats(prefixes...)` for the stats callback;
   `utils.Throttle` for log throttling, `utils.Timestamped` for last-value+staleness and
   `utils.Normalize` for case-insensitive unit/mode lookups. (refs §8)

9. **Delete the replaced code** — the old `Check()`/`scheduleRecheck`/`cancelRecheck`, the bespoke
   authenticator/transport, the hand-rolled config service, the sync wrapper, the
   credential-in-config plumbing. Less code is the point of this bump.

10. **Read refs §9 anyway.** Several 1.3.4 behaviour changes bite a freshly ported adapter as hard
    as an existing one — the failed-start rollback calling `Stop()` on partially started services,
    `storage.Reset()` clearing fields absent from `defaults/config.json`, and the batched thing-sync
    writes. Then **verify**: `go vet ./...`, `golangci-lint run`, `make test` (coverage gate 75%,
    target >80%), `make build-arm`, `make deb-arm`, and **prove wire-compatibility** by diffing the
    inclusion report + a representative evt stream against the pre-bump capture. Finally, review
    logs — invoke the `improve_logs` skill if available, else check them by hand.

---

# Track B — from v1.3.3 to v1.3.4

A delta bump. The architecture does not change; **almost nothing breaks at compile time, and that is
the hazard** — most of the risk is in behaviour that still compiles. Refs §9 is the full list with
symptoms and fixes; this is the order to work it in.

1. **Bump the pin.** `go get github.com/futurehomeno/cliffhanger@v1.3.4`, `go mod tidy`, build.
   Go stays at 1.26. The only compile break to expect is a stored function value of one of the
   routing constructors that gained `options ...config.RoutingOption`.

2. **Grep for `interface{ Stop() }`.** `telemetry.Telemetry` gained `Start() error`/`Stop() error`,
   so the 1.3.3 type-assertion workaround now silently fails and telemetry is never stopped. And
   `telemetry.New` no longer starts the cloud config poll: with `root.Builder.WithTelemetry` the app
   drives it, but an adapter constructing telemetry outside the root app must call `Start()`/`Stop()`
   itself. Regenerate any `telemetry.Telemetry` mock, or use `test/mocks/telemetry`.

3. **Audit every `root.Service.Stop()`.** A partly failed `Start` is now unwound: services that
   started before the failing one get `Stop()`ed even though the app never reached steady state.
   `Stop()` must be safe on a half-started service. `doStop` is also best-effort now and joins its
   errors, so returning an error no longer aborts the rest of the shutdown.

4. **Guard the ConnectivityChecker probe against the new auth sentinels.**
   `auth.ErrRefreshSuspended` does **not** wrap `httpclient.ErrUnauthorized`, so a probe firing
   during a backoff window lands in the generic-failure bucket and reports a disconnection that is
   not real. `auth.ErrReloginRequired` is terminal but only chains `ErrUnauthorized` on one of its
   three paths. In the probe: skip on `ErrRefreshSuspended`/`ErrRefreshDeferred`, treat
   `ErrReloginRequired` as auth loss. (refs §4)

5. **Check the reset and thing-sync paths.** `storage.Reset()` now zeroes the model before reloading
   defaults, so config fields absent from `defaults/config.json` are genuinely cleared and `json:"-"`
   fields must be restored by the caller — `config.Service.Reset()` handles `WorkDir`/`ConfigDir`,
   a hand-rolled reset must replicate it. `InitializeThings`/`EnsureThings` now batch their
   `adapter.json` writes and a displaced thing is `Disconnect()`ed, so `Disconnect()` must be
   idempotent and safe on an instance that never fully connected.

6. **Service-specific checks, if you use them.** `outlvlswitch` now clamps `cmd.lvl.set` instead of
   forwarding out-of-range values (drop your own clamping; `start_lvl` error strings changed);
   `virtualmeter`'s garbage-collection argument is reinterpreted as a post-orphan grace period; a
   `router.MessageHandler` on a value receiver starts working after silently dropping every message.

7. **Optionally adopt `alarm_system`** if the adapter exposes a car charger (refs §10). Pure
   addition — no existing behaviour changes, and the event names match what the zigbee charger
   already advertises, so the hub renders the faults identically.

8. **Verify.** `go vet ./...`, `golangci-lint run`, `make test`, `make build-arm`, `make deb-arm`,
   and diff the inclusion report + evt stream against the pre-bump capture. Adding `alarm_system`
   changes the inclusion report by design — that one is expected.

---

## Output expectations

- Adapter compiles and runs on cliffhanger `v1.3.4`.
- Track A additionally: the hand-rolled Check/auth/config/sync/secrets code is replaced by framework
  primitives and net lines are **removed**.
- Identical FIMP behaviour: same addresses, same reports, no re-inclusion for existing users — with
  the deliberate exception of a newly added `alarm_system` service.
- Adapters that don't fit a primitive (zaptec/easee/tibber Check, hub-proxy auth) are left as-is
  with a one-line note why.
- CI green.
