# Cliffhanger v1.2.10 → v1.3.0 upgrade reference

1.3.0 (cliffhanger `develop`) extracts the boilerplate that every edge adapter re-implemented
into framework primitives. This doc catalogs the new APIs, maps each adapter's hand-rolled code
onto them, and lists the fixes made between 1.2.10 and 1.3.0. It assumes the 1.2 base
architecture (see the `port_to_cliffhanger_1.2` skill) — only the deltas are here.

The change landed in four "waves":
- **#185** (`44f6379`) — common blocks extracted from edge adapters (checker, httpclient errors, utils).
- **#186** (`cb03c87`) — lifecycle bundles, app flows, generic config service.
- **#187** (`db09e05`) — OAuth authenticator, bearer transport, seeds-selection helper.
- **#189** (`d4459bd`) — edge bundles, stream supervisor, secrets storage.
- Follow-up hardening: #197/#198/#200/#201/#202 (checker & auth race/edge fixes) and #203
  (thing-sync helpers + checker customization hooks). See §10.

---

## 1. New package/API map (what replaces what)

| New in 1.3 | Signature (key parts) | Replaces the hand-rolled… |
|---|---|---|
| `app.ConnectivityChecker` | `NewConnectivityChecker(probe func() error, lc, reporter ConnectivityReporter, cfg CheckerConfig)` → `Check()/CheckNow()/Cancel()/CheckInterval()` | `Check()` + `scheduleRecheck`/`cancelRecheck`/`checkFailed` |
| `app.Reset/Logout/Authorize/ConfigModel` | `Authorize(lc, persistCredentials, check func() error)`, `Reset(lc, ThingDestroyer, resetConfig, teardown...)`, `Logout(lc, clearCredentials, teardown...)`, `ConfigModel[T](any)(T,error)` | inline Login/Logout/Reset lifecycle sequences |
| `auth.Authenticator` | `NewAuthenticator(CredentialsStore, TokenExchanger, AuthenticatorConfig)` → `AccessToken()(string,error)` | bespoke token cache + refresh + backoff |
| `auth.Transport` | `http.RoundTripper` wrapping a `TokenSource` | hand-written bearer-header injector |
| `auth.TokenExpirationDate` | `(jwtToken string)(time.Time,error)` | manual JWT expiry parsing |
| `httpclient` errors + builder | `ErrUnauthorized`, `ErrTooManyRequests`, `*TooManyRequestsError{RetryAfter}`, `ErrorFromResponse(resp)`, `NewJSONRequest(ctx,method,url,body,headers)` | per-adapter 401/429 sentinels + request builders |
| `config.Service[C]` | `NewService[C](storage.Storage[C], defaults, redact)` → `Update/Persist/Reset/Migrate/DefaultStore/PublicModel`; `Get[C,V]`, `GetDuration[C]`; `config.BackoffConfig` | hand-rolled `Service` struct + RWMutex + setters |
| `lifecycle.MarkRunning/MarkNotConfigured` + `SetConnAndAuthState` | state-bundle helpers; atomic conn+auth in one event | four separate `Set*State` calls |
| `storage.NewSecrets/NewCanonicalSecrets` | `Storage[T]` at 0640 (`data/secrets.json`) | credentials living in world-readable `config.json` |
| `stream.Supervisor` | `NewSupervisor(connect Connection, b backoff.Stateful)` → `Start()/Stop()/TriggerReconnect()` | bespoke SignalR/GENA/AMQP reconnect loops |
| `adapter.SeedsFromSelection` + `SyncThings` + `RebuildChangedThings` | `SeedsFromSelection[T](available, selected, seed)`, `SyncThings[T](a, available, selected, seed) (ErrIncompleteFetch)`, `RebuildChangedThings(seeds)` | fetch→filter→map→EnsureThings wrapper, anti-wipe guard, sensibo `reconcile.go` |
| `bootstrap.EdgeRouting/EdgeTasks` | `EdgeRouting[C](...)`, `EdgeTasks(...)` bundles | repeated builder wiring |
| `router.DefaultLogStats` | `DefaultLogStats(redactedInterfacePrefixes ...string) func(Stats)` | per-adapter `LogStats` with `cmd.auth.` masking |
| `backoff.NewTolerantFixed` | `(tolerance uint32, delay time.Duration) Stateful` | fixed-after-N-tolerated backoff presets |
| `utils.Throttle`, `utils.Timestamped[T]` | `Throttle.Do(key, emit)/Reset()`; `Timestamped.Set/Get` | ad-hoc log throttling & last-value+staleness caches |

All additive: the 1.2.10 API surface (`EnsureThings`, `NewAdapter`, `storage.New`, `config.Default`,
`backoff.New/NewStateful`, cache strategies, `RouteApp`) is unchanged, so a bump breaks nothing you
already use — you delete your copies at your own pace.

---

## 2. httpclient — the error contract (adopt first)

`httpclient/errors.go` defines the sentinels the rest of 1.3 keys on:
- `ErrUnauthorized` — 401 **or** 403.
- `ErrTooManyRequests` — 429; `*TooManyRequestsError{RetryAfter time.Duration}` carries a parsed
  `Retry-After` and matches `ErrTooManyRequests` via `errors.Is`.
- `ErrorFromResponse(resp)` maps a status code to the right sentinel (with Retry-After).
- `NewJSONRequest(ctx, method, url, body, headers)` builds a JSON request.

Make the API client wrap these (`fmt.Errorf("...: %w", httpclient.ErrUnauthorized)` is fine —
matched with `errors.Is`). Everything downstream (`ConnectivityChecker` branches, `Authenticator`
grace, `SyncThings`) depends on this classification being correct.

---

## 3. ConnectivityChecker — the Check() replacement (the big one)

`app/check.go`. Construct once, expose its `Check()`/`CheckInterval()` as the app's `CheckableApp`,
and drive it from the task layer (`app.TaskApp`).

```go
checker := app.NewConnectivityChecker(
    func() error { return client.Ping() },   // probe: cheap real call → httpclient sentinels
    appLifecycle,
    adapter,                                  // ConnectivityReporter (adapter.Adapter satisfies it)
    app.CheckerConfig{
        Interval:       time.Hour,            // 0 → DefaultCheckInterval (task layer)
        RecheckBackoff: backoff.New(time.Minute, 5*time.Minute, 15*time.Minute, 1, 2),
        MaxRechecks:    1,                    // disconnect only after >N consecutive failures
        RateLimitDelay: 5 * time.Minute,
        // 1.3 customization hooks (PR #203):
        AuthLossState: func(cur lifecycle.State) lifecycle.State { // default: always Lost
            if cur == lifecycle.AuthStateAuthenticated { return lifecycle.AuthStateLost }
            return lifecycle.AuthStateNotAuthenticated              // stay silent if never authed
        },
        Authorized: func() bool { return cfg.GetCredentials().AccessToken != "" }, // gate restore vs teardown
    },
)
```

Outcome → lifecycle (identical to the old hand-rolled machine):
- **success** → `Authenticated` + `Connected`, then `repairAppHealth` (Running/Configured if it was
  stranded NOT_CONFIGURED); gated by `Authorized()`.
- **`ErrUnauthorized`** → `SetConnAndAuthState(Disconnected, AuthLossState(cur))` — one atomic event.
- **`ErrTooManyRequests`** → connectivity untouched, reschedule (honors `RetryAfter`).
- **other/transient** → throttled warn, disconnect only once `failures > MaxRechecks`, reschedule with backoff.

Wiring: `checker.CheckNow` is the `Authorize` callback (forces an immediate probe, resets the
failure streak); `checker.Cancel` on logout/reset (stops rechecks, discards an in-flight probe via
the cancelled/stale guard).

### Per-adapter fit table (checked across all edge adapters)

| Adapter | Probe today | Fits ConnectivityChecker? | Notes / hooks needed |
|---|---|---|---|
| **sensibo** | `client.Ping()` | **Yes** | `AuthLossState` (Lost only if was Authenticated) + `Authorized` (creds present). Move its checksum rebuild to `RebuildChangedThings`. |
| **sonos** | `GetHouseHolds()` | **Yes** | Keep its UPnP event-driven `SyncThings()`; the hourly Check becomes the checker's periodic probe. |
| **mill** | `AccessToken()`+`AllDevices()` | **Yes** | Fold token acquisition into the probe; drift self-heal → `RebuildChangedThings`. |
| **netatmo** | `client.Ping()` | **Yes** | `AuthLossState` keyed on empty refresh token (closure over cfg); teardown via `Authorized`. |
| **zaptec** | `restClient.Ping()` (ConnState only) | **No** | Auth-loss handled out-of-band via `rest.AuthLost` event; not even a `CheckableApp`. Leave as-is. |
| **easee** | `Check()` is a no-op | **No** | Real probe lives in the `RefreshToken` task; use `stream.Supervisor` for SignalR, keep the task. |
| **tibber** | none (local-state reconciler) | **No** | No network probe; pure `AuthState`/`ConnState` from local token presence. Leave as-is. |

Rule: if the adapter does an active periodic probe that maps outcome→lifecycle, adopt the checker.
If auth-loss is event-driven (zaptec), the work lives in a separate task (easee), or there's no
probe at all (tibber), **don't** force it — that would change behaviour.

---

## 4. auth — Authenticator, Transport, JWT

`auth/authenticator.go`. For username/password→JWT and authorization-code refresh:

```go
authr := auth.NewAuthenticator(cfgSecrets /*CredentialsStore*/, client /*TokenExchanger*/, auth.AuthenticatorConfig{
    RefreshLead:       5 * time.Minute,                       // refresh this early
    Backoff:           backoff.NewTolerantFixed(2, 5*time.Minute),
    UnauthorizedGrace: time.Hour,                             // tolerate 401 streak before concluding loss
    OnAuthLoss:        func(reason string) { appLifecycle.MarkNotConfigured() /* + notify */ },
})
token, err := authr.AccessToken()   // cached; refreshes on expiry; concludes auth loss past grace
```

- `CredentialsStore` = `{ Credentials() Credentials; SetCredentials(Credentials) error; ClearCredentials() error }`.
- `TokenExchanger` = `{ ExchangeRefreshToken(string) (*OAuth2TokenResponse, error) }`.
- `Credentials{AccessToken, RefreshToken, ExpiresAt, RefreshExpiresAt}`; `expired(lead)`; `Empty()`.
- **`OnAuthLoss` runs under the Authenticator lock** (#199) — its callback must not call back into
  the Authenticator (deadlock). Emit lifecycle/notifications only.
- `UnauthorizedGrace` tolerates transient 401s; a rejection streak that outlives the grace concludes
  auth loss even while backoff suppresses exchanges, and the streak is tied to the rejected refresh
  token so an out-of-band credential swap isn't concluded lost on a stale timestamp (#197/#201).

Attach the token with `auth.Transport` (`http.RoundTripper` over a `TokenSource`) rather than a
hand-written header injector. Use `auth.TokenExpirationDate(jwt)` to read expiry.

Auth models that do **not** use Authenticator: hub-proxied (adax — proxy + hub token, no refresh)
and plain API key (sensibo — `credentials.accessToken` used directly). Leave those.

---

## 5. config.Service[C] — generic config service

`config/service.go`. Replaces the per-adapter `Service` struct + RWMutex + stamped setters:

```go
cfgSrv := config.NewService[*Config](storageSvc, func(c *Config) *config.Default { return &c.Default }, redactFn)
cfgSrv.Update(func(c *Config) { c.PollingInterval = "5m" })   // mutate + stamp ConfiguredAt + Save
cfgSrv.Persist(fn); cfgSrv.Reset(); cfgSrv.Migrate(migrations...)
interval := config.GetDuration(cfgSrv, func(c *Config) string { return c.PollingInterval }, time.Hour)
public := cfgSrv.PublicModel()   // redact-returning; hand it a copy for pointer configs (see caveat)
```

- Keep your `Config` struct + JSON shape identical — this is a mechanical swap of the service layer.
- `config.BackoffConfig{...}.Stateful(def)` builds a `backoff.Stateful` from persisted config.
- **`PublicModel()` caveat**: for a pointer-backed config with no redactor it aliases the live model;
  pass a copy-returning `redact` func (mirrors the `Get` helper caveat, #197). `Update`'s lock is
  not reentrant — callbacks must not call back into the Service.

---

## 6. Secrets storage

`storage/secrets.go`. Move credentials out of the world-readable `config.json` (0644) into a
0640 `data/secrets.json`:

```go
secrets := storage.NewSecrets[*Credentials](&Credentials{}, workDir, "secrets.json")
// or NewCanonicalSecrets(...) for the core-application canonical layout
```

Migrate once: on first 1.3 boot, if the legacy config still carries tokens, copy them into the
secrets store and blank them in config. The `Authenticator`'s `CredentialsStore` is typically
backed by this secrets storage.

---

## 7. lifecycle bundles + thing-sync helpers

- **`lifecycle.MarkRunning()`** = Running+Configured+Connected+Authenticated;
  **`MarkNotConfigured()`** = NotConfigured+Disconnected+NotAuthenticated (uninstall/reset/logout).
  Only for cloud adapters with auth; apps at `*NA` set states individually.
- **`SetConnAndAuthState(conn, auth)`** emits a single atomic auth event with both states applied —
  use it on auth-loss so a watcher never sees `LOST` with a stale connection state (#198/#200). The
  ConnectivityChecker already uses it internally.
- **`adapter.SeedsFromSelection[T](available, selected, seed)`** builds `ThingSeeds` from a fetched
  list filtered to the user's selection.
- **`adapter.SyncThings[T](a, available, selected, seed)`** = SeedsFromSelection + `EnsureThings`
  with a shared anti-wipe guard: returns `ErrIncompleteFetch` (non-fatal; log-and-retry) instead of
  destroying live things when a registered selected device is missing from a partial fetch. Replaces
  the guard hand-rolled four different ways (sensibo/sonos/mill/zaptec).
- **`adapter.RebuildChangedThings(seeds)`** rebuilds a registered thing whose seed yields a different
  service topology (picks up new capabilities), preserving its address **and** persisted per-thing
  state, build-before-destroy. Replaces sensibo `reconcile.go`. `EnsureThings` still handles presence
  only; call both (SyncThings for presence, then RebuildChangedThings for drift).

---

## 8. Remaining blocks

- **`stream.Supervisor`** (`stream/supervisor.go`): wrap a push transport's connect loop.
  `NewSupervisor(func(ctx, connected func()) error {...}, backoff.Stateful)` → `Start/Stop/
  TriggerReconnect`. Register as a `root.Service` so its lifecycle follows the app. Use for easee
  SignalR, sonos GENA, zaptec AMQP — replaces the bespoke reconnect/backoff goroutine (and fixes the
  Stop/Start races, §10).
- **`app` flow helpers** (`app/flow.go`): `Authorize(lc, persist, check)`, `Reset(lc, destroyer,
  resetConfig, teardown...)`, `Logout(lc, clear, teardown...)`, `ConfigModel[T](any)`. Use in the
  app's Login/Logout/Reset/Configure so the lifecycle sequence isn't hand-written.
- **`bootstrap.EdgeRouting[C]` / `EdgeTasks`**: bundle the common routing/task wiring; adopt if it
  reduces builder boilerplate for your adapter.
- **`router.DefaultLogStats(prefixes...)`**: the stats callback with credential-prefix redaction
  (`"cmd.auth."`) — pass to `router.WithStatsCallback`. Replaces the per-adapter `LogStats`.
- **`utils.Throttle`**: `Do(key, emit)` collapses repeated log lines (the checker uses it for the
  transient-error warn); `Reset()` on recovery. **`utils.Timestamped[T]`**: last-value + timestamp +
  staleness, for "report only if newer" and connectivity-detail caches.

---

## 9. go.mod & migration checklist

1. `go get github.com/futurehomeno/cliffhanger@<develop>` + `go mod tidy`; build to surface deltas.
2. httpclient error contract in the client (§2).
3. Replace `Check()` with `ConnectivityChecker` **iff** the adapter is in the "Yes" fit set (§3).
4. `Authenticator` + `Transport` for OAuth refresh (§4); skip for hub-proxy/API-key.
5. Secrets storage for credentials + one-time migration (§6).
6. `config.Service[C]` swap, same Config JSON (§5).
7. Lifecycle `Mark*`/`SetConnAndAuthState`; `SyncThings`/`RebuildChangedThings` for thing-sync (§7).
8. `stream.Supervisor`, flow helpers, `DefaultLogStats`, utils where they apply (§8).
9. Delete the replaced hand-rolled code.
10. `go vet` / `golangci-lint` / `make test` (gate 75%, >80%) / `build-arm` / `deb-arm`.
11. **Prove FIMP wire-compat**: diff inclusion report + evt stream vs the pre-bump capture.
12. `/improve_logs`.

---

## 10. Fixes made between 1.2.10 and 1.3.0 (behaviours to inherit, not re-break)

These landed while the common blocks were hardened; a from-scratch reimplementation must not
regress them:

- **Checker timer race** (#198): a `Cancel()` racing a firing recheck timer is caught by a fused
  `{timer, cancelled}` commit or discarded via `stale()`. Don't "simplify" the checker's mutex to
  atomics — the fusion is load-bearing.
- **Auth-loss bundle** (#198/#200): auth-loss sets connection+auth atomically
  (`SetConnAndAuthState`) so a watcher never reports `LOST` with an outdated connection state.
- **CheckNow freshness** (#198/#201): manual/authorize checks use `CheckNow` (clears pending
  backoff + resets the failure streak) so fresh credentials get a clean slate and aren't tripped by
  a stale streak.
- **Backoff vs grace** (#197/#201): an expired token whose 401 streak outlives `UnauthorizedGrace`
  concludes auth loss even while backoff suppresses exchanges; the streak is tied to the rejected
  refresh token (an out-of-band credential swap is not concluded lost on a stale timestamp).
- **repairAppHealth guard** (#201/#202): a successful probe only repairs a *stranded* app
  (NOT_CONFIGURED → Running/Configured); an already-RUNNING app is left untouched.
- **Rebuild correctness** (#203): `RebuildChangedThings` preserves the thing's address (pins
  `CustomAddress` to the live address) **and** its persisted per-thing state across the
  destroy/create, runs under the adapter lock (atomic like `EnsureThings`), and guards a nil
  inclusion report.
- **SyncThings anti-wipe** (#203): a partial/empty fetch returns `ErrIncompleteFetch` rather than
  destroying live things.
- **Stream supervisor Stop/Start** (#189 hardening): fd-leak on Chmod failure and Stop/Start races
  fixed — use the framework supervisor rather than a bespoke loop.
- **OnAuthLoss under lock** (#199): the callback runs holding the Authenticator lock; keep it to
  lifecycle/notification side effects only.

The checker/auth test suites (`app/check_test.go`, `auth/authenticator_test.go`) and the
thing-sync tests (`adapter/drift_test.go`, `adapter/sync_test.go`) are the executable spec for all
of the above — read them when in doubt about an edge case.
