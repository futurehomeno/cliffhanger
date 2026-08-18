# Cliffhanger v1.3.4 upgrade reference

The 1.3 line extracts the boilerplate that every edge adapter re-implemented into framework
primitives. This doc catalogs the APIs, maps each adapter's hand-rolled code onto them, and lists
the behaviours a reimplementation must not re-break.

**Target `v1.3.4`.** It assumes the 1.2 base architecture (see the `port_to_cliffhanger_1.2` skill)
— only the deltas are here.

Two entry paths, and they use different parts of this document:

- **From v1.2.10** — §1–§8 are the work, §9.1 lists the compile/boot breakages, §11 is the checklist.
- **From v1.3.3** — §9 is the whole job. §10 if you want the new `alarm_system` service.

---

## 1. New package/API map (what replaces what)

| Primitive | Signature (key parts) | Replaces the hand-rolled… |
|---|---|---|
| `app.ConnectivityChecker` | `NewConnectivityChecker(probe func() error, lc, reporter ConnectivityReporter, cfg CheckerConfig)` → `Check()/CheckNow()/Cancel()/CheckInterval()` | `Check()` + `scheduleRecheck`/`cancelRecheck`/`checkFailed` |
| `app.Reset/Logout/Authorize/ConfigModel` | `Authorize(lc, persistCredentials, check func() error)`, `Reset(lc, ThingDestroyer, resetConfig, teardown...)`, `Logout(lc, clearCredentials, teardown...)`, `ConfigModel[T](any)(T,error)` | inline Login/Logout/Reset lifecycle sequences |
| `app.NewLogCapture()` | `LogProvider` over a shared warn+ ring buffer | a hand-wired `formatters.NewErrorHook()` |
| `auth.Authenticator` | `NewAuthenticator(CredentialsStore, TokenExchanger, AuthenticatorConfig)` → `AccessToken()(string,error)` | bespoke token cache + refresh + backoff |
| `auth.Transport` | `http.RoundTripper` wrapping a `TokenSource` | hand-written bearer-header injector |
| `auth.TokenExpirationDate` | `(jwtToken string)(time.Time,error)` | manual JWT expiry parsing |
| `httpclient` errors + builder | `ErrUnauthorized`, `ErrTooManyRequests`, `*TooManyRequestsError{RetryAfter}`, `ErrorFromResponse(resp)`, `NewJSONRequest(ctx,method,url,body,headers)` | per-adapter 401/429 sentinels + request builders |
| `config.Service[C]` | `NewService[C](storage.Storage[C], defaults, redact)` → `Update/Persist/Reset/Migrate/DefaultStore/PublicModel`; `Get[C,V]`, `GetDuration[C]`; `config.BackoffConfig` | hand-rolled `Service` struct + RWMutex + setters |
| `config.PublishConfigurationChange` | `(serviceName fimptype.ServiceNameT, setting string, options ...RoutingOption)` | a hand-written setting handler that emitted no change event |
| `lifecycle.MarkRunning/MarkNotConfigured` + `SetConnAndAuthState` | state-bundle helpers; atomic conn+auth in one event | four separate `Set*State` calls |
| `storage.NewSecrets/NewCanonicalSecrets` | `Storage[T]` at 0640 (edge: `data/secrets.json`; canonical/core: `workDir/<name>`) | credentials in a world-readable `config.json` |
| `stream.Supervisor` | `NewSupervisor(connect Connection, b backoff.Stateful)` → `Start()/Stop()/TriggerReconnect()` | bespoke SignalR/GENA/AMQP reconnect loops |
| `adapter.SeedsFromSelection` + `SyncThings` + `RebuildChangedThings`, `selection` | see §7 | fetch→filter→map→EnsureThings wrapper, per-adapter `selected_devices` plumbing and `cmd.thing.delete` route |
| `bootstrap.EdgeRouting/EdgeTasks` | `EdgeRouting[C](...)`, `EdgeTasks(...)` bundles | repeated builder wiring |
| `router.DefaultLogStats` | `DefaultLogStats(redactedInterfacePrefixes ...string) func(Stats)` | per-adapter `LogStats` with `cmd.auth.` masking |
| `backoff.NewTolerantFixed` | `(tolerance uint32, delay time.Duration) Stateful` | fixed-after-N-tolerated backoff presets |
| `event.Bus[T]` | `NewBus[T]()` → `Subscribe(subID, buffer, filters...) chan T` / `Unsubscribe(subID)` / `Publish(value, dropped func(subID string))` | a per-package "map of ID to channels" fan-out |
| `utils.Throttle`, `utils.Timestamped[T]`, `utils.Normalize` | `Throttle.Do(key, emit)/Reset()`; `Timestamped.Set/Get`; `Normalize[T ~string](value T, allowed []T) (T, bool)` | ad-hoc log throttling, last-value caches, copy-pasted case-insensitive lookups |
| `adapter/service/alarm` | see §10 | nothing — a new capability, car chargers only |

The 1.2.10 API surface (`EnsureThings`, `NewAdapter`, `storage.New`, `config.Default`,
`backoff.New/NewStateful`, cache strategies, `RouteApp`) is preserved, so a bump breaks almost
nothing you already use — you delete your copies at your own pace. The exceptions are in §9.1.

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
    probe,                                    // see below
    appLifecycle,
    adapter,                                  // ConnectivityReporter (adapter.Adapter satisfies it)
    app.CheckerConfig{
        Interval:       time.Hour,            // 0 → DefaultCheckInterval (task layer)
        RecheckBackoff: backoff.New(time.Minute, 5*time.Minute, 15*time.Minute, 1, 2),
        MaxRechecks:    1,                    // disconnect only after >N consecutive failures
        RateLimitDelay: 5 * time.Minute,
        AuthLossState: func(cur lifecycle.State) lifecycle.State { // default: always Lost
            if cur == lifecycle.AuthStateAuthenticated { return lifecycle.AuthStateLost }
            return lifecycle.AuthStateNotAuthenticated              // stay silent if never authed
        },
        // Read creds through the credStore's synchronized accessor, NOT secrets.Model() directly:
        // storage.Model() returns the live model unlocked, so an unguarded read here races with
        // login/logout writes (the checker calls Authorized off the config lock).
        Authorized: func() bool { return credStore.Credentials().AccessToken != "" },
    },
)
```

Outcome → lifecycle (identical to the old hand-rolled machine):

- **success** → `Authenticated` + `Connected`, then `repairAppHealth` (Running/Configured if it was
  stranded NOT_CONFIGURED — an already-RUNNING app is left untouched); gated by `Authorized()`.
- **`ErrUnauthorized`** → `SetConnAndAuthState(Disconnected, AuthLossState(cur))` — one atomic event.
- **`ErrTooManyRequests`** → connectivity untouched, reschedule (honors `RetryAfter`).
- **other/transient** → throttled warn, disconnect only once `failures > MaxRechecks`, reschedule
  with backoff.

Wiring: `checker.CheckNow` is the `Authorize` callback (forces an immediate probe, clears pending
backoff and resets the failure streak); `checker.Cancel` on logout/reset.

### The probe must filter the auth sentinels

If the probe's HTTP client runs through an `auth.Transport`, the `Authenticator`'s errors reach the
checker verbatim (`http.Client` boxes them in `*url.Error`, which unwraps, so `errors.Is` works
end to end). Two of them are misclassified by the checker's default branching — see the taxonomy in
§4. Filter them in the probe:

```go
probe := func() error {
    err := client.Ping()
    switch {
    // Transient: a backoff window or a tolerated 401 inside UnauthorizedGrace. Neither wraps
    // ErrUnauthorized, so without this they fall into the generic-failure bucket and the checker
    // reports a disconnection that is not real.
    case errors.Is(err, auth.ErrRefreshSuspended), errors.Is(err, auth.ErrRefreshDeferred):
        return nil
    // Terminal, but only one of its three paths chains ErrUnauthorized. The Authenticator has
    // already driven OnAuthLoss; say so explicitly rather than letting it read as a disconnect.
    case errors.Is(err, auth.ErrReloginRequired):
        return fmt.Errorf("%w: %w", httpclient.ErrUnauthorized, err)
    }
    return err
}
```

### Per-adapter fit table

| Adapter | Probe today | Fits ConnectivityChecker? | Notes / hooks needed |
|---|---|---|---|
| **sensibo** | `client.Ping()` | **Yes** | `AuthLossState` (Lost only if was Authenticated) + `Authorized` (creds present). Move its checksum rebuild to `RebuildChangedThings`. |
| **sonos** | `GetHouseHolds()` | **Yes** | Keep its UPnP event-driven `SyncThings()`; the hourly Check becomes the checker's periodic probe. |
| **mill** | `AccessToken()`+`AllDevices()` | **Yes** | Done. Token acquisition folded into the probe; drift self-heal → `RebuildChangedThings`. |
| **netatmo** | `client.Ping()` | **Yes** | `AuthLossState` keyed on empty refresh token (closure over cfg); teardown via `Authorized`. |
| **hoiax** | `client.Ping(ctx)` | **Yes** | Done. `AuthLossState`/`Authorized` both keyed on the secrets store's refresh token, so a 401 with no session stays `NotAuthenticated`. Probe folds in the `SyncThings` pass. |
| **zaptec** | `restClient.Ping()` (ConnState only) | **No** | Auth-loss handled out-of-band via an `AuthLost` event bridged to `lc.SetAuthState`. Leave as-is. |
| **easee** | `Check()` is a no-op | **No** | The real probe lives in the `RefreshToken` task; auth loss comes from `auth.Authenticator`'s `OnAuthLoss`. Keep the task. |
| **tibber** | none (local-state reconciler) | **No** | No network probe; pure `AuthState`/`ConnState` from local token presence. Leave as-is. |

Rule: if the adapter does an active periodic probe that maps outcome→lifecycle, adopt the checker.
If auth-loss is event-driven (zaptec), the work lives in a separate task (easee), or there's no
probe at all (tibber), **don't** force it — that would change behaviour.

---

## 4. auth — Authenticator, Transport, JWT

`auth/authenticator.go`. For username/password→JWT and authorization-code refresh:

```go
// credStore is YOUR thin adapter over the secrets store implementing auth.CredentialsStore —
// storage.Storage[T] (Model/Load/Save/Reset) is NOT a CredentialsStore, so a wrapper is required.
// storage.Model() returns the live model UNLOCKED, so guard the wrapper with its own mutex around
// every read/write (login/logout/reset and the checker's Authorized read run on different
// goroutines):
//   func (s *credStore) Credentials() auth.Credentials { s.mu.RLock(); defer s.mu.RUnlock(); ... }
//   func (s *credStore) SetCredentials(c auth.Credentials) error { s.mu.Lock(); defer s.mu.Unlock(); ... }
//   func (s *credStore) ClearCredentials() error { s.mu.Lock(); defer s.mu.Unlock(); return s.secrets.Reset() }
authr := auth.NewAuthenticator(credStore, client /*TokenExchanger*/, auth.AuthenticatorConfig{
    RefreshLead:       5 * time.Minute,                       // refresh this early
    Backoff:           backoff.NewTolerantFixed(2, 5*time.Minute),
    UnauthorizedGrace: time.Hour,                             // tolerate a 401 streak before concluding loss
    // Auth loss = previously authenticated, now rejected → Lost (not MarkNotConfigured, which is
    // logout/reset). SetConnAndAuthState so watchers see LOST with a consistent conn state:
    OnAuthLoss: func(reason string) {
        appLifecycle.SetConnAndAuthState(lifecycle.ConnStateDisconnected, lifecycle.AuthStateLost)
    },
})
token, err := authr.AccessToken()
```

- `CredentialsStore` = `{ Credentials() Credentials; SetCredentials(Credentials) error; ClearCredentials() error }`
  — implement it on the adapter side over the secrets store (no cliffhanger type satisfies it).
- `TokenExchanger` = `{ ExchangeRefreshToken(string) (*OAuth2TokenResponse, error) }`.
  `auth.ProxyClient` already satisfies it and returns typed `httpclient` sentinels, so any
  per-adapter `isAuthRejection`/`isRateLimit` string matching on `"received status code: 401"` is
  dead code — delete it.
- `Credentials{AccessToken, RefreshToken, ExpiresAt, RefreshExpiresAt}`; `expired(lead)`; `Empty()`.
- **`OnAuthLoss` runs under the Authenticator lock** — its callback must not call back into the
  Authenticator (deadlock). Emit lifecycle/notifications only, and hand off anything that might
  block (closing a stream client that is itself waiting on `AccessToken`) to a goroutine.
- **`UnauthorizedGrace` is a behaviour change, not a freebie.** Non-zero grace means a genuinely
  revoked token is not reported as `AUTH_STATE_LOST` until the window elapses. Set it to `0` when
  the pre-1.3 client concluded auth loss on the first rejected refresh and the API is not known for
  spurious 401s — that keeps the bump wire-compatible.

### `AccessToken()` error taxonomy

The full contract. `errors.Is` against these instead of string-matching; the strings changed.

| Error | Wraps `ErrUnauthorized`? | Fires `OnAuthLoss`? | Meaning | What your probe should do |
|---|---|---|---|---|
| `ErrNotLoggedIn` | no | no | the store is empty, nothing to refresh | nothing — not a connectivity fault |
| `ErrRefreshDeferred` | **deliberately no** | no | a 401 being tolerated inside `UnauthorizedGrace` | treat as transient, skip |
| `ErrRefreshSuspended` | no | no | backoff is suppressing the exchange; returns on **every** call for the whole window | treat as transient, skip |
| `ErrReloginRequired` (refresh token expired) | no | **yes** | past `RefreshExpiresAt`, access token also expired | report auth loss |
| `ErrReloginRequired` (rejected past grace, during backoff) | no | **yes** | 401 streak outlived the grace while backoff suppressed retries | report auth loss |
| `ErrReloginRequired` (live 401/403 past grace) | **yes** | **yes** | the exchange itself was rejected | report auth loss (handled by default) |
| anything else | passthrough | no | transient network/5xx | let the checker count it |

Hierarchy to code against: `ErrNotLoggedIn` (nothing to do) < `ErrRefreshDeferred` /
`ErrRefreshSuspended` (transient, wait) < `ErrReloginRequired` (terminal, only a fresh login clears
it). `ErrRefreshSuspended` in particular fires on every call for the whole backoff window — match it
and downgrade the log level, or a five-minute window produces one error line per request.

Attach the token with `auth.Transport` (`http.RoundTripper` over a `TokenSource`) rather than a
hand-written header injector. Use `auth.TokenExpirationDate(jwt)` to read expiry.

Auth models that do **not** use Authenticator: hub-proxied (adax — proxy + hub token, no refresh),
plain API key (sensibo — `credentials.accessToken` used directly), and event-driven auth loss
(zaptec — its REST client publishes an `AuthLost` event that a handler maps to
`lc.SetAuthState(AuthStateLost)`). Leave those.

---

## 5. config.Service[C] — generic config service

`config/service.go`. Replaces the per-adapter `Service` struct + RWMutex + stamped setters:

```go
cfgSrv := config.NewService[*Config](storageSvc, func(c *Config) *config.Default { return &c.Default }, redactFn)
cfgSrv.Update(func(c *Config) { c.PollingInterval = "5m" })   // mutate + stamp ConfiguredAt + Save
cfgSrv.Persist(fn); cfgSrv.Reset(); cfgSrv.Migrate(migrations...)
interval := config.GetDuration(cfgSrv, func(c *Config) string { return c.PollingInterval }, time.Hour)
public := cfgSrv.PublicModel()
```

Both zaptec and easee embed it as `type Service struct { *config.Service[*Config] }` and add typed
accessors — that is the idiomatic shape.

- Keep your `Config` struct + JSON shape identical — this is a mechanical swap of the service layer.
- `config.BackoffConfig{...}.Stateful(def)` builds a `backoff.Stateful` from persisted config.
- It caches one `DefaultStore` behind a single shared lock, so delete any per-adapter
  `DefaultStore()` bridge that allocated a fresh wrapper (and its own mutex) per call.
- **`PublicModel()` caveat**: for a pointer-backed config with no redactor it aliases the live
  model; pass a copy-returning `redact` func. `Update`'s lock is not reentrant — callbacks must not
  call back into the Service.
- **`Update` no-rollback caveat**: `Update` mutates the live model *then* saves and does **not** roll
  it back if the save fails — a persistence error leaves the new value in memory while disk holds
  the old one. Adapters needing strict config/disk consistency must snapshot and restore themselves.
- **`Reset()` and `json:"-"` fields**: the underlying `storage.Reset()` zeroes the model before
  reloading defaults, so fields the defaults file cannot carry come back empty. `config.Service`
  restores `Default.WorkDir` and `Default.ConfigDir` for you (from a `defer`, so they survive the
  error path too). **Any other `json:"-"` field on your config is yours to restore**, and any field
  absent from `defaults/config.json` is now genuinely cleared by a reset rather than surviving it.

---

## 6. Secrets storage

`storage/storage.go` (`NewSecrets`/`NewCanonicalSecrets`). Move credentials out of the
world-readable `config.json` (0644) into a 0640 `data/secrets.json`:

```go
secrets := storage.NewSecrets[*Credentials](&Credentials{}, workDir, "secrets.json")
// or NewCanonicalSecrets(...) for the core-application canonical layout
```

Migrate once: on first 1.3 boot, if the legacy config still carries tokens, copy them into the
secrets store and blank them in config. Do it as a **`config.Migration` step** (`From: N, To: N+1`)
whose `Do` copies and blanks — it runs under the config service lock, so it only mutates the model
and `Service.Migrate` persists it once together with the version bump. Make it **idempotent**: if
the secrets store is already non-empty, only clear the stale config copy (an earlier run may have
written secrets and then failed to persist the blanked config). Keep the legacy JSON keys on
`Config` as read-only `,omitempty` fields so a pre-1.3 `config.json` still parses.

The storage layer sets the modes: config `0644`, secrets `0640`. A `Save()` renames the data file to
`.bak` rather than copying it, then chmods the backup — so an upgrade from a world-readable secrets
file does not leave the credentials behind in a 0644 `.bak`. `Load()` falls back to the `.bak` when
the data file is missing, which is the state a crash between rename and rewrite leaves behind.

---

## 7. lifecycle bundles + thing-sync helpers

- **`lifecycle.MarkRunning()`** = Running+Configured+Connected+Authenticated;
  **`MarkNotConfigured()`** = NotConfigured+Disconnected+NotAuthenticated (uninstall/reset/logout).
  Only for cloud adapters with auth; apps at `*NA` set states individually. For a *failed* logout —
  credentials still on disk — use `SetAppState(AppHealthError, ConfigStateNotConfigured,
  ConnStateDisconnected, AuthStateNotAuthenticated)` instead, so `ConnState` does not stay stale at
  connected.
- **`SetConnAndAuthState(conn, auth)`** emits a single atomic auth event with both states applied —
  use it on auth-loss so a watcher never sees `LOST` with a stale connection state. The
  ConnectivityChecker already uses it internally. Consequence for **your** subscribers: a bundled
  setter reaches several states while emitting one event, so a subscriber tracking connection,
  health or config state must **re-read `lc.State(...)` on any event** rather than filtering on
  `SystemEvent.Type`.
- **`adapter.SeedsFromSelection[T](available, selected, seed)`** builds `ThingSeeds` from a fetched
  list filtered to the user's selection. A **nil** selection includes every available device; a
  non-nil empty one includes none. Duplicate entries collapse to a single seed.
- **`adapter.SyncThings[T](a, fetch, selected, seed)`** = fetch + SeedsFromSelection + exclusion of
  vanished devices + `EnsureThings`. It returns **three** values:

  ```go
  seeds, excludedIDs, err := adapter.SyncThings(a.ad, a.fetchDevices, sel, seedDevice)
  ```

  Drop `excludedIDs` from the persisted selection — nothing in the sync mutates `selected`, so
  otherwise the same vanished device is re-announced on every subsequent sync. `seeds` feeds
  `RebuildChangedThings` without fetching again.

  **The fetch owns the truth**: it fails → nothing is mutated; it succeeds → every selected device
  absent from the response is destroyed, including on an empty response. Make the client return an
  error, never a truncated or empty slice, on a non-2xx, a rate limit or an unparsable body.
  Selected devices the adapter owns no thing for still get an exclusion, which clears stale nodes
  left by a pre-cliffhanger adapter.

  `SyncThings` is **not atomic** — it takes and releases the adapter lock per operation. Serialise it
  against `cmd.thing.delete` and configuration writes with a shared `router.MessageHandlerLocker`.
- **`adapter.RebuildChangedThings(seeds)`** rebuilds a registered thing whose seed yields a different
  service topology (picks up new capabilities), preserving its address (pins `CustomAddress` to the
  live address) **and** its persisted per-thing state, build-before-destroy, under the adapter lock.
  `EnsureThings` handles presence only; call both. Filter the seeds first if a degenerate
  third-party response must never rebuild a thing (sensibo's null capabilities).
- **`selection`** owns the user's device selection: `selection.Devices` is the config mixin carrying
  `selected_devices`, `selection.Store` reads/writes it, and `selection.PrepareManifest` renders the
  device selector's three states (not ready / fetch failed / ready). Pass the store to
  `adapter.RouteAdapter(ad, adapter.WithSelection(store, locker))` so `cmd.thing.delete` also
  deselects the device; without it the next sync recreates the deleted thing.
- **Best effort**: `EnsureThings`, `RebuildChangedThings`, `DestroyAllThings` and `SyncThings` do not
  abort on the first bad device — they continue and join the errors. A non-nil error means
  *partially applied*, not "nothing happened".

### 7.1 Porting a device selection

Config — embed the mixin and wire a store over the adapter's own locked accessors (`selection.Store`
is closure-based precisely so adopting it does not force a `config.Service` migration first, and
keeping a plain `SelectedDevices []string` on `Config` is fine):

```go
store := selection.NewStore(
    func() selection.Selection { return cfgSrv.GetSelectedDevices() },
    func(next selection.Selection) error { return cfgSrv.SetSelectedDevices(next) },
)
```

`get` must copy under the read lock and `set` must write under the write lock and stamp
`ConfiguredAt` — `Store.Remove` is a read-then-write and relies on the handler lock below.

One helper serves both `Configure` and the boot re-sync:

```go
func (a *application) syncDevices(sel selection.Selection) error {
    seeds, excluded, err := adapter.SyncThings(a.adapter, a.fetchDevices, sel, seedDevice)
    if err != nil {
        return fmt.Errorf("sync things: %w", err)
    }

    a.deselect(excluded) // or the same vanished device is re-announced every sync

    // Filter before the drift pass if a degenerate response must never rebuild a thing.
    return a.adapter.RebuildChangedThings(rebuildable(seeds))
}
```

`Configure` calls `syncDevices(cfg.SelectedDevices)` then `store.Set(...)`; the boot path calls
`syncDevices(store.Get())` and logs rather than failing. Run it on **every** boot, not only when the
thing store is empty — a partially lost store never recovers otherwise.

Routing — delete the adapter's own `cmd.thing.delete` route and pass the store instead, using the
same locker as `app.RouteApp`:

```go
cliffAdapter.RouteAdapter(ad, cliffAdapter.WithSelection(store, configurationLocker))
```

Manifest — `selection.PrepareManifest(m, block, ready, fetch, option)` replaces the hand-rolled
three-state builder and tolerates a template that renamed the block or config (dereferencing those
panics the app on `cmd.app.get_manifest`).

**Two hazards when porting:**

1. **The nil flip.** An adapter where a missing `selected_devices` meant "no devices" needs a config
   migration pinning `nil` → `selection.Selection{}`, or every install without the key silently
   includes the whole account. An adapter where empty already meant "all devices" needs none —
   check what its writers actually persist, since `append([]string(nil), …)` marshals to `null`.
2. **The fetch must error.** Any client that returns a short or empty slice on a non-2xx, a rate
   limit, an unparsable body or a truncated page will destroy things. A retry budget for a flaky
   list endpoint belongs *inside* the fetch closure, returning an error on exhaustion — not in the
   sync.

---

## 8. Remaining blocks

- **`stream.Supervisor`** (`stream/supervisor.go`): wrap a push transport's connect loop.
  `NewSupervisor(func(ctx context.Context, connected func()) error {...}, backoff.Stateful)` →
  `Start/Stop/TriggerReconnect`. Register as a `root.Service` so its lifecycle follows the app. It
  fixes the Stop/Start races and the fd leak that bespoke loops have, and its backoff is not
  advanced on clean or triggered reconnects and is reset after the wait regardless of which `select`
  case wins. **No adapter has adopted it yet** — easee and zaptec both still hand-roll their loops,
  so port against `stream/supervisor_test.go`, not against a sibling. Two invariants any
  replacement must keep, learned from the hand-rolled versions: a client restart must invalidate
  *every* per-device subscription flag (a sweep that trusts per-device disconnect notices misses
  some), and the shutdown path must be observable — emitting the disconnect state on `ctx.Done()`,
  not merely recording it.
- **`app` flow helpers** (`app/flow.go`): `Authorize(lc, persist, check)`, `Reset(lc, destroyer,
  resetConfig, teardown...)`, `Logout(lc, clear, teardown...)`, `ConfigModel[T](any)`. Use in the
  app's Login/Logout/Reset/Configure so the lifecycle sequence isn't hand-written.
- **`bootstrap.EdgeRouting[C]` / `EdgeTasks`**: bundle the common routing/task wiring. `EdgeRouting`
  takes an `adapterOptions []adapter.RoutingOption` argument before its variadic extras — pass
  `nil`, or the selection wiring from §7.1.
- **`router.DefaultLogStats(prefixes...)`**: the stats callback with credential-prefix redaction
  (`"cmd.auth."`) — pass to `router.WithStatsCallback`.
- **`event.Bus[T]`**: the fan-out backing `event.Manager` and `lifecycle.Lifecycle`. Use it directly
  if your adapter has its own. Contract: each `Subscribe` under one ID gets its own channel and
  `Unsubscribe(id)` closes **all** of them, so share an ID only across subscribers with the same
  lifetime; filters are cloned on subscribe and a nil filter is skipped; `dropped` may be nil; both
  `dropped` and the filters run under the read lock, so neither may subscribe or unsubscribe.
- **`utils.Throttle`**: `Do(key, emit)` collapses repeated log lines; `Reset()` on recovery. `emit`
  runs **under the lock** — it must not block or call back into the same `Throttle`.
  **`utils.Timestamped[T]`**: last-value + timestamp + staleness. **`utils.Normalize`**:
  case-insensitive lookup returning the canonical spelling (`"kwh"` → `"kWh"`) — use it wherever
  you validate an incoming FIMP unit/mode/state against a supported set.
- **`GetManifest` should not probe.** Read the checker-maintained `ConnectionState()` instead of
  running a synchronous check, and spend a `manifest_timeout` on the device-list fetch (pass a
  `context.WithTimeout` down to the client) if the adapter had one. That deletes the whole
  `checkWithTimeout` construct.
- **Fold the thing sync into the probe.** The checker's `probe()` already makes a cloud call; do the
  device sync there too, so the fetch is not repeated and a selection change made while the adapter
  was offline is picked up on the next check. Guard it with the **same
  `router.MessageHandlerLocker`** passed to `app.RouteApp`, using its non-blocking `Lock()`: a
  configuration handler in flight just means this cycle skips the sync. Sync failures are **logged,
  not returned** — a device-sync hiccup must never be reported as a connectivity loss.

---

## 9. v1.3.3 → v1.3.4

**Almost nothing breaks at compile time, and that is the hazard.** Ordered by risk.

### 9.1 Compile-time breaks

Coming from **1.3.3**, only one thing can fail to build: a stored *function value* of a routing
constructor that gained a variadic `options ...config.RoutingOption` — `telemetry.Route`,
`telemetry.RouteCmdTelemetrySetEnabled/SetValidity/SetSuppressed`, `debug.Route`,
`debug.RouteCmdLogSetLevel/SetFormat/SetFile/SetRevertTimeout`. Ordinary call sites still compile.
(The `Get*` log routings deliberately did not gain options.)

Coming from **1.2.10**, expect these as well:

- **`app.App` embeds `LogProvider`.** Every application needs `ErrorsReport() ([]string, error)`.
  Embed the framework helper rather than hand-rolling a hook — `app.NewLogCapture()` installs a
  single shared warn+ ring buffer on the global logger, so repeated `app.New` calls across a test
  binary do not stack hooks:
  ```go
  type application struct {
      app.LogProvider
      // ...
  }
  a := &application{LogProvider: app.NewLogCapture(), /* ... */}
  ```
  This supersedes the `add_log_provider` skill's manual `formatters.NewErrorHook()` wiring.
- **`telemetry.New(mqtt, resourceName, store, version string)`** gained the trailing build version.
  Thread `version` through `getTelemetry(cfg, version)` / `newRouting(cfg, version)` from `Build`.
- **`debug.Route` panics** with `"debug: Route called before InitializeLogger"` if
  `InitializeLogger` never ran, and an **empty `log_file` is a hard failure** (`setLogOutput`
  returns `log file not set`). `bootstrap.DefaultRoute` calls `debug.Route`, so any test fixture
  carrying `"log_file": ""` turns into a panic at app build. Fix the fixtures, not the code. What
  1.3.4 relaxes: a *first* initialization that fails to open its file now keeps the (non-nil)
  manager and returns the error, so an adapter that logs it and carries on runs without file
  logging instead of crash-looping.
- **`fan_ctrl`** uses string value types and reports the actual mode. Relevant only to adapters
  exposing `fan_ctrl`.
- **`config.Default` gained `LogFlushInterval time.Duration \`json:"log_flush_interval,omitempty"\`**
  — a 64 KiB buffer sits between logrus and lumberjack, drained on a tick (240s default, 2s while
  debug/trace is on), on a full buffer, on error-level entries and on graceful shutdown. No action
  needed, but call `debug.FlushLogs()` before a deliberate exit, and expect log lines to lag by up
  to the interval when tailing a file at info level.

### 9.2 Behaviour changes that compile silently

**1. `telemetry` lifecycle moved out of the constructor — highest risk.**
`telemetry.Telemetry` gained `Start() error` and `Stop() error`, and `Stop()` changed from no return
to `error`. The 1.3.3 workaround `if s, ok := tel.(interface{ Stop() }); ok { s.Stop() }` therefore
**silently fails its type assertion** and telemetry is never stopped — grep for it.
`telemetry.New` no longer starts the cloud config poll; `Start()` arms the validity window and
starts the poll, `Stop()` tears both down. With `root.Builder.WithTelemetry` the app drives it
(started after the MQTT transport, on the start-rollback undo stack, stopped before the transport it
publishes on so a late cloud config report cannot rewrite a store a `Reset()` just wiped; a start
failure is logged, never fatal). **Constructing telemetry outside the root app means it never polls
unless you call `Start()` yourself.** Regenerate any `telemetry.Telemetry` mock, or use the new
`test/mocks/telemetry`.

The 6h first-poll delay is unchanged and still intentional — there is no poll at startup. What
changed is that the clock now starts at app start rather than at process construction, and `Stop()`
blocks until the listener goroutine has exited, so do not call it from inside a handler that
listener could be blocked on.

**2. A partly failed `root` `Start` is now rolled back.** Each successful step pushes an undo
closure; on error they run in reverse, app health becomes `AppHealthStartupError`, and logs are
flushed. Previously a failure after the MQTT connect left MQTT, your services, the router and
subscriptions running while `running` stayed false — the caller's next `Stop()` short-circuited and
a `Reset()` wiped app data under live services.

Consequences: **`Service.Stop()` is now called for services that started before the failing one**,
even though the app never reached steady state — it must be safe on a half-started service. The
service whose `Start()` failed is *not* stopped. And `doStop` is best-effort now: every step runs
and the result is `errors.Join(...)`, so your `Stop()` error no longer aborts the rest of the
shutdown but does surface in a joined error (unwrap with `errors.Is`, do not string-compare).

**3. New auth sentinels.** `auth.ErrRefreshSuspended` and `auth.ErrReloginRequired` are now exported
— the same conditions previously returned unmatchable bare `errors.New` strings. See the taxonomy
in §4 and the probe guard in §3; without it a backoff window reports a disconnection that is not
real.

**4. `storage.Reset()` zeroes the model before reloading defaults.** In 1.3.3 the defaults were
unmarshalled *over* the live model, so fields and map entries absent from the defaults file survived
a reset and were written back on the next `Save()`. They are now genuinely cleared, and `json:"-"`
fields come back empty. `config.Service.Reset()` restores `WorkDir`/`ConfigDir`; a hand-rolled reset
calling `Storage.Reset()` directly must replicate it, and any other `json:"-"` field is yours (§5).

**5. Thing sync collapses into one state write.** Every thing record lives in `adapter.json`, so a
pass over the fleet used to rewrite and fsync the whole file two to three times per device.
`InitializeThings` and `EnsureThings` now run in a deferred-save batch: one write regardless of how
many things the pass touches. What follows from that:

- Per-thing rollbacks do not fire inside a batch — a save "succeeds" in memory and only fails at the
  flush, so a failed flush leaves memory ahead of disk. The next boot sees ghost records, which
  `EnsureThings` heals.
- The flush runs **even when the pass returns an error** (both callers are best-effort per thing),
  and the error is `errors.Join(passErr, flushErr)`.
- The deferral is **process-wide**: an unrelated concurrent `SetState`/`SetInclusionChecksum` during
  a sync is collapsed into the same flush and shares its fate.
- A failed flush keeps the dirty mark so the next batch retries it.

Two related changes in the same area: `InitializeThings` now **skips addresses that are already
live** and registers each thing immediately after its inclusion report succeeds — so a thing created
by an inbound command before the init task runs is no longer duplicated and re-announced, and a
partial failure leaves the successful things registered. And `registerThing` now `Disconnect()`s a
thing it displaces, so **`Disconnect()` must be idempotent and safe on an instance that never fully
connected** — if your `Connector` holds sockets or streams, a replaced instance is torn down instead
of leaked.

Address allocation also moved into the state write: the index increment is rolled back on a failed
write, so a failed thing creation no longer burns an address, and `SetInclusionChecksum` no-ops on
an unchanged checksum.

**6. `outlvlswitch` clamps instead of forwarding.** `cmd.lvl.set` is clamped to
`[min_lvl, max_lvl]` before `SetLevelSwitchLevel` — drop your own clamping, and do not rely on
receiving 0 or a negative to mean "off" (off is `cmd.binary.set`). `start_lvl` in a transition is
still **rejected**, with a reworded message, so error-string matchers in tests break. New failure
mode: an inverted specification (`min_lvl > max_lvl`) errors on **every** level command — and note
`outlvlswitch.Specification(..., switchType string, maxLvl, minLvl int, ...)` takes **maxLvl before
minLvl**. If yours were swapped it silently worked before and now hard-fails.

**7. virtualmeter garbage collection rewritten.** The old collector deleted any device whose
`LastTimeUpdated` was older than the cleaning period — which destroyed configured modes and
accumulated energy for meters that were merely offline, since nothing refreshes that stamp while
inactive. Now only entries with **no live virtual meter service** are candidates, the grace runs
from a persisted `OrphanedSince` stamp that survives restarts, and a live service vetoes deletion.
**`NewManager`'s third argument is reinterpreted**: it is now "grace after a device goes orphaned",
not "staleness of last update". Also: the inclusion report is sent on *every* `add`, not only when
the service was newly inserted, so a retry after a failed publish no longer reports success without
the hub ever learning about the meter.

**8. Router handlers on a value receiver start working.** The nil-handler guard called
`reflect.ValueOf(handler).IsNil()`, which panics for kinds that cannot be nil — the panic was
recovered, so such a handler silently dropped *every* message while logging a stack. If your adapter
has one, it goes live on 1.3.4; make sure it is correct and idempotent.

**9. Smaller, one line each.**

- `lifecycle`/`event` `Subscribe` no longer returns the existing channel when the ID is taken —
  each call gets its own channel, buffer and filters, and every subscriber sees every event. If you
  relied on "subscribe twice, get the same channel" as de-dup, you now have two live subscriptions.
- `Lifecycle.WaitFor` subscribes before reading the state (it could previously block forever on a
  transition landing in the gap), appends a UUID to `subID` (so your `subID` is now a prefix, not
  the registered key), and re-reads `State(...)` on any event instead of matching the event.
- `cmd.auth.logout` is registered once, from a dedicated `LogoutableApp` branch. An app implementing
  both `LogginableApp` and `AuthorizableApp` used to get two routings, log out twice and publish two
  contradictory `evt.auth.status_report`s per command. Side effect: an app implementing `Logout()`
  but neither `Login()` nor `Authorize()` now gets the route where it previously did not.
- An in-progress async login is no longer reported failed — the FHX compatibility hack only marks
  the report failed for genuinely terminal states (`NotAuthenticated`, `Error`, `Lost`).
- The `prime` observer no longer holds the write lock across `GetComponents`, so concurrent stale
  readers make one request between them and the notification stream is not stalled. A failed refresh
  is shared with the readers already waiting behind it, so an outage costs one request, not one per
  reader — and several concurrent `GetDevices()` can now return the *same* error. A `Notify` applied
  while a fetch was in flight wins over the fetched snapshot, so `Refresh(true)` is not guaranteed
  to install it. The refresh event is published outside the lock, so a goroutine workaround around a
  subscriber that reads the observer back can go.
- `prime.Devices.FilterByIDs` deduplicates — a repeated id no longer yields the same device twice.
- `cmd.log.set_file` resolves a relative name against the current log file's directory and persists
  an **absolute** path; `evt.log.file_report` carries it. Under systemd the log used to move to the
  daemon's CWD and move again on every restart.
- `security.Decrypt` returns an error on a ciphertext shorter than the 12-byte nonce instead of
  panicking the process. Drop any recover/guard you had around it.
- `database` compaction percentage defaults to 100 instead of 0 — at 0 the shrink condition
  degenerated to "the file grew at all", rewriting the whole database on the next tick after any
  write past 2 MiB.
- `auth.Transport` treats a redirect hop whose `Response.Request` is nil as stripped rather than
  panicking. Adapter **tests** with a fake `Base` that set `req.Response` now see no `Authorization`
  header.
- `thing.ServiceByTopic` is an exact map lookup on the address cut at `/rt:dev/rn:` instead of a
  suffix scan. A synthesized topic must contain the full service address or it will simply miss.

---

## 10. The `alarm_system` service

New in 1.3.4, pure opt-in, and **wired only into the car charger thing**. The event names are
deliberately the ones the zigbee charger already advertises, so the hub renders faults from a cloud
adapter identically to zigbee ones.

### API

```go
// routing.go
const (
    CmdAlarmGetReport = "cmd.alarm.get_report"
    EvtAlarmReport    = "evt.alarm.report"
    AlarmSystem       = "alarm_system"
)
func RouteService(serviceRegistry adapter.ServiceRegistry) []*router.Routing

// service.go
const (
    PropertySupportedEvents = "sup_events"
    StatusActivate          = "activ"
    StatusDeactivate        = "deactiv"
)
const (
    EventGroundingFault = "GROUNDING_FAULT"
    EventPENFault       = "PEN_FAULT"
    EventOverTemp       = "OVER_TEMP_FAULT"
    EventRCDTrip        = "RCD_TRIP_FAULT"
    EventOverVoltage    = "OVER_VOLTAGE_FAULT"
    EventUnderVoltage   = "UNDER_VOLTAGE_FAULT"
    EventOverCurrent    = "OVER_CURRENT_FAULT"
    EventMeterComFault  = "METER_COM_FAULT"
    EventGridTypeFault  = "GRIDTYPE_CONFIG_FAULT"
    EventOtherChargeErr = "CP_OTHER_ERROR"
)
var DefaultReportingStrategy = cache.ReportOnChangeOnly()

type Report struct {
    Event  string `json:"event"`
    Status string `json:"status"`
}

// Implemented by the adapter. Returns a nil report when the device has no stateful state for
// that event and no ephemeral alert queued.
type Reporter interface {
    AlarmReport(event string) (*Report, error)
}

type Service interface {
    adapter.Service
    SendAlarmReport(event string, force bool) (bool, error)
    SupportedEvents() []string
}

type Config struct {
    Specification     *fimptype.Service
    Reporter          Reporter
    ReportingStrategy cache.ReportingStrategy // optional, defaults to DefaultReportingStrategy
}

func NewService(publisher adapter.ServicePublisher, cfg *Config) Service

// specification.go — note: plain strings, no SpecificationOption variadic
func Specification(resourceName, resourceAddress, address string, groups, events []string) *fimptype.Service

// tasks.go
func TaskReporting(serviceRegistry adapter.ServiceRegistry, frequency time.Duration, voters ...task.Voter) *task.Task
```

### Semantics not obvious from the signatures

- A **nil** `*Report` → `(false, nil)`: no message, no error. That is the "device has nothing
  stateful to say about this event" answer, not an error path.
- The outgoing payload always carries the **advertised** spelling of the event, not whatever the
  reporter put in `Report.Event`. The report is copied before caching, so a reporter may reuse its
  struct between calls.
- `SendAlarmReport` normalizes the event case-insensitively against `sup_events` and errors with
  `"event is unsupported"` if it is not advertised.
- The message is a StrMap `evt.alarm.report` published with
  `WithStorageStrategy(fimpgo.StorageStrategyAggregate, report.Event)` — **each event gets its own
  retained slot and its own reporting-cache key.** This is why the cache key is the advertised event
  and not the reporter's: with `ReportOnChangeOnly`, a genuine over-temp right after a grounding
  fault would otherwise compare equal and be dropped.
- `cmd.alarm.get_report` (null or empty payload) fans out to **every** supported event with
  `force=true`; `TaskReporting` iterates all events with `force=false`.

### Wiring

`thing.CarChargerConfig` has an optional `AlarmConfig *alarm.Config`; leaving it nil creates no
service and changes nothing:

```go
cfg := &thing.CarChargerConfig{
    ThingConfig:       &adapter.ThingConfig{ /* ... */ },
    ChargepointConfig: &chargepoint.Config{ /* ... */ },
    AlarmConfig: &alarm.Config{
        Specification: alarm.Specification(
            ad.Name().Str(), ad.Address(), thingState.Address(), groups,
            []string{alarm.EventGroundingFault, alarm.EventGridTypeFault, alarm.EventOtherChargeErr},
        ),
        Reporter: controller, // implements AlarmReport(event string) (*alarm.Report, error)
    },
}
```

`thing.RouteCarCharger` already includes `alarm.RouteService`, and `thing.TaskCarCharger` already
includes `alarm.TaskReporting` — so an adapter using both gets routing and periodic reporting for
free. **An adapter that hand-builds its task set must add `alarm.TaskReporting(ad, interval, voter)`
itself.**

A generated mock ships at `test/mocks/adapter/service/alarm`, with a
`MockAlarmReport(report, event, err, once)` helper. The worked example is
`adapter/thing/car_charger_alarm_test.go`.

### The two reference implementations

Both live on the `feat/alarm-system-service` branch of their repo, and they are deliberately
opposite shapes.

**zaptec — pull.** One event. The vendor exposes an undocumented `Warnings` bitmask, so there is no
mapping table: any non-zero mask collapses to `EventOtherChargeErr`, with the mask logged. The
reporter hits the charger state each poll:

```go
func (c *controller) AlarmReport(event string) (*alarm.Report, error) {
    states, err := c.charger.States()
    if err != nil {
        return nil, fmt.Errorf("retrieve charger state: %w", err)
    }
    status := alarm.StatusDeactivate
    if states.Warnings != 0 {
        log.Warnf("[%s] Warnings=%#x", c.charger.Name(), states.Warnings)
        status = alarm.StatusActivate
    }
    return &alarm.Report{Event: event, Status: status}, nil
}
```

It hand-builds its task set, so it adds `alarm.TaskReporting` explicitly; routing comes free from
`thing.RouteCarCharger`.

**easee — push.** Three events, driven from SignalR observations, with a cache mirror. The real
mapping table lives on the vendor's `GridType` enum (`IsGroundFault()` → `EventGroundingFault`,
`IsWiringFault()` → `EventGridTypeFault`), plus observation `ErrorCode != 0` →
`EventOtherChargeErr`. The reporter is then a pure cache read with no I/O:

```go
func (c *controller) AlarmReport(event string) (*alarm.Report, error) {
    status := alarm.StatusDeactivate
    if c.cache.AlarmActive(event) {
        status = alarm.StatusActivate
    }
    return &alarm.Report{Event: event, Status: status}, nil
}
```

and the observation handler forces the report, gated on the cache's change detection:

```go
func (h *observationsHandler) sendAlarmReports(events map[string]bool) error {
    service, err := getAlarmService(h.thing)
    if err != nil {
        return err
    }
    for event, active := range events {
        if !h.cache.SetAlarm(event, active) { // returns "changed?"
            continue
        }
        if _, err := service.SendAlarmReport(event, true); err != nil {
            return err
        }
    }
    return nil
}
```

It sets no `ReportingStrategy` — the default is fine because change detection already happens in the
cache. It uses `thing.TaskCarCharger`, so no task change was needed.

**Neither returns a nil report**: a cleared alarm is an explicit `StatusDeactivate`. Reserve nil for
a device that genuinely has no state for the event. Pick the shape from your transport — pull if the
adapter polls, push with a cache mirror if the vendor streams.

---

## 11. Migration checklist

**From v1.2.10:**

1. `go mod edit -go=1.26` (+ bump CI/runtime images to Go 1.26), then
   `go get github.com/futurehomeno/cliffhanger@v1.3.4` + `go mod tidy`; build to surface the §9.1
   deltas.
2. httpclient error contract in the client (§2).
3. Replace `Check()` with `ConnectivityChecker` **iff** the adapter is in the "Yes" fit set (§3),
   with the auth-sentinel probe guard.
4. `Authenticator` + `Transport` for OAuth refresh (§4); skip for hub-proxy/API-key/event-driven.
5. Secrets storage for credentials + a one-time `config.Migration` (§6).
6. `config.Service[C]` swap, same Config JSON (§5).
7. Lifecycle `Mark*`/`SetConnAndAuthState`; `SyncThings`/`RebuildChangedThings` for thing-sync (§7).
8. `stream.Supervisor`, flow helpers, `DefaultLogStats`, utils where they apply (§8).
9. Delete the replaced hand-rolled code.
10. Read §9.2 — the failed-start rollback, `Reset()` clearing, and batched thing-sync writes bite a
    fresh port as hard as an existing one.
11. `go vet` / `golangci-lint` / `make test` (gate 75%, >80%) / `build-arm` / `deb-arm`.
12. **Prove FIMP wire-compat**: diff inclusion report + evt stream vs the pre-bump capture.
13. Review logs (invoke `improve_logs` if available, else by hand).

**From v1.3.3:** bump the pin, then work §9.2 in order (1–9), then §10 if you want `alarm_system`,
then steps 11–13 above. Do not redo anything in §1–§8.

---

## 12. Invariants — do not re-break these

A reimplementation or a "simplification" of the framework code must preserve these. They were all
paid for once.

- **The checker's timer fusion is load-bearing.** A `Cancel()` racing a firing recheck timer is
  caught by a fused `{timer, cancelled}` commit or discarded via a stale guard. Do not reduce the
  checker's mutex to atomics.
- **Auth-loss is one atomic event.** `SetConnAndAuthState` so a watcher never reports `LOST` with an
  outdated connection state.
- **`CheckNow` means a clean slate.** Manual/authorize checks clear pending backoff and reset the
  failure streak, so fresh credentials are not tripped by a stale streak.
- **Backoff must not outlive the grace.** An expired token whose 401 streak outlives
  `UnauthorizedGrace` concludes auth loss even while backoff suppresses exchanges — and the streak
  is tied to the rejected refresh token, so an out-of-band credential swap is not concluded lost on
  a stale timestamp.
- **`repairAppHealth` only repairs a stranded app.** NOT_CONFIGURED → Running/Configured; an
  already-RUNNING app is left untouched.
- **`RebuildChangedThings` preserves identity.** The thing's address (pinned via `CustomAddress`)
  and its persisted per-thing state survive the destroy/create, which runs under the adapter lock.
- **A failed fetch mutates nothing; a successful one is authoritative.** The anti-wipe
  responsibility sits in the adapter's client, which must turn a partial or rate-limited response
  into an error rather than a short slice.
- **`OnAuthLoss` runs under the Authenticator lock.** Lifecycle and notification side effects only;
  anything that might block goes on a goroutine, and snapshot the credentials synchronously so a
  fresh login landing in between is not logged straight back out.

The checker/auth test suites (`app/check_test.go`, `auth/authenticator_test.go`) and the thing-sync
tests (`adapter/drift_test.go`, `adapter/sync_test.go`) are the executable spec — read them when in
doubt about an edge case.
