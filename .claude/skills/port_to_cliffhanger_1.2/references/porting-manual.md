# Porting a Futurehome edge adapter to cliffhanger v1.2.10

Manual for migrating legacy edge adapters (hand-rolled fimpgo routing or cliffhanger v0.x) to the
cliffhanger v1.2.10 framework, and for building new adapters on it. Distilled from:
`edge-mill-adapter` (cloud thermostat + LAN fallback — best overall template),
`edge-sensibo-adapter` (multi-service climate things, modern packaging + CI),
`edge-sonos-adapter` (UPnP push), `edge-easee-adapter` (SignalR push),
`edge-zaptec-adapter` (AMQP push, packaging migration reference), `edge-netatmo-adapter` (OAuth).

## 0. Pick your templates

| Concern | Copy from |
|---|---|
| Overall skeleton (cmd/, internal/, app, tasks, routing) | mill |
| Multi-service / capability-driven things | sensibo |
| Makefile + packaging (futurehome layout, staging build) | sensibo (zaptec equivalent) |
| go.yml CI workflow | sensibo (release needs test+linter, first-release-safe changelog) |
| .golangci.yaml | netatmo or sonos (fullest: includes paralleltest/tparallel) |
| .mockery.yaml | mill (has `issue-845-fix`, `resolve-type-alias`) |
| Debian maintainer scripts + migrate.sh | zaptec (best documented; sensibo equivalent) |
| Async push transport | easee (SignalR), sonos (UPnP/GENA), zaptec (AMQP), mill (LAN REST fallback) |
| Adapter-state (adapter.json) migration | zaptec `src/internal/migration/` |

Do NOT use `core-energy-guard` as a template — it is a core service, not an edge adapter.
Easee lags at cliffhanger v1.2.7; use it only for the SignalR pattern.

## 1. Package structure

```
Makefile                      # unified style, see §12
VERSION file only if legacy scripts need it; version lives in Makefile + git tags
.github/workflows/go.yml
.githooks/pre-commit          # gofmt guard
package/debian/               # futurehome layout, see §13
scripts/coverage_gate.awk     # or coverage-gate.sh
src/
  main.go                     # thin: cmd.Execute(PackageName, Version)
  go.mod                      # module github.com/futurehomeno/edge-<name>-adapter, cliffhanger v1.2.10
  .golangci.yaml
  .mockery.yaml
  cmd/
    root.go                   # Execute() + Build() -> root.NewEdgeAppBuilder()
    factory.go                # serviceContainer: lazy get*() singletons (hand-rolled DI)
    testing.go                # bootstrap helpers for e2e tests
  internal/
    app/app.go                # cliffhanger app.App implementation (§3)
    config/config.go          # Config + thread-safe Service + Migrate() (§8)
    routing/routing.go        # router.Combine(...) table + LogStats (§10)
    tasks/tasks.go            # task.Combine(...) periodic tasks (§5)
    adapter/thing.go          # ThingFactory (§4)
    <vendor>/                 # domain: client.go, auth.go, controller.go, model
    model/                    # DTOs if not in <vendor>/
    migration/                # legacy config/adapter.json migration if porting (§8)
    test/mocks/               # mockery output; test/fakes/ hand-written
```

One package = one concern. The `<vendor>` package owns the API client and the `Controller`
(device driver implementing service interfaces). `adapter` assembles things from services.

## 2. Bootstrap

`main.go`:

```go
var Version = "unknown"; var PackageName = "<name>"
func main() { cmd.Execute(PackageName, Version) }
```

`cmd/root.go` — `Execute` loads config (`bootstrap.GetConfigurationDirectory()`), calls `Build`,
`app.Run()`. The builder:

```go
root.NewEdgeAppBuilder().
    WithMQTT(getMQTT(cfg)).
    WithServiceDiscovery(routing.ResourceName, discovery.ResourceTypeAd, packageName, "1", version).
    WithLifecycle(getLifecycle()).
    WithTelemetry(getTelemetry(cfg)).
    WithTopicSubscription(
        router.TopicPatternAdapter(routing.ResourceName, fimptype.MsgTypeCmd),
        router.TopicPatternDevice(routing.ResourceName, fimptype.MsgTypeCmd),
    ).
    WithRouterOptions(router.WithStatsCallback(routing.LogStats)).
    WithRouting(newRouting(cfg)...).
    WithTask(newTasks(cfg)...).
    WithServices(/* push transport managers, event listeners — root.Service */).
    Build()
```

`cmd/factory.go` — package-level `services = &serviceContainer{}` with lazy singletons:
- `getConfigStorage()` → `storage.New(config.New(workDir), workDir, "config.json")`
- `getLifecycle()` → `lifecycle.New(getConfigService().DefaultStore())`
- `getAdapter()` → `cliffAdapter.NewState(workDir)` + `cliffAdapter.NewAdapter(mqtt, eventManager, thingFactory, state, routing.ResourceName, "1")`

Async transports (SignalR/UPnP/AMQP managers) are registered via `WithServices(...)` so their
`Start()/Stop()` follow the app lifecycle.

## 3. The app package — lifecycle and Check()

Implement and assert the interfaces you need (cliffhanger `app/app.go`):

```go
var (
    _ app.App              = (*Application)(nil)   // GetManifest, Configure, Uninstall
    _ app.InitializableApp = (*Application)(nil)   // Initialize
    _ app.CheckableApp     = (*Application)(nil)   // Check, CheckInterval
    _ app.LogginableApp    = (*Application)(nil)   // Login (note spelling)
    _ app.LogoutableApp    = (*Application)(nil)   // Logout
    // app.AuthorizableApp for cmd.auth.set_tokens, app.ResettableApp for factory reset
)
```

`RouteApp` picks these up by type assertion — never hand-write `cmd.auth.*` handlers.

**Check()** (mill `app.go:198` is the canonical state machine):
1. No credentials → `SetConnState(Disconnected)`, return.
2. Probe the cloud with a cheap real call.
3. 401 → drop token, `SetAuthState(AuthStateLost)`, `SetConnState(Disconnected)`.
4. 429/transient → stay `Connected` (or schedule a 5-min recheck before flipping — mill
   `scheduleRecheck`); avoids hour-long false outages since `CheckInterval()` is typically 1h.
5. Success → `Authenticated` + `Connected`, then self-heal: reconcile registered things vs the
   device list (`EnsureThings`).

**Initialize()**: `adapter.InitializeThings()`, set lifecycle from stored credentials
(none → NotConfigured/NotAuthenticated/Disconnected; present → Configured/Authenticated/
Connected/Running), then an initial `Check()`.

**Configure(model any)**: type-assert `*config.Config`, apply changes (may re-run
`EnsureThings`), `SetConfigState(Configured)`, `SetAppHealth(Running, nil)`.

## 4. Adapter and ThingFactory — creating/destroying devices

Implement `adapter.ThingFactory` (single `Create` method); the adapter owns destruction.

```go
func (f *thingFactory) Create(ad adapter.Adapter, publisher adapter.Publisher, ts adapter.ThingState) (adapter.Thing, error) {
    var info model.Info
    ts.Info(&info)                        // persisted per-thing info from the seed
    controller := vendor.NewController(...)
    cfg := &thing.ThermostatConfig{
        ThingConfig: &adapter.ThingConfig{Connector: controller, InclusionReport: f.inclusionReport(...)},
        ThermostatConfig: &thermostat.Config{Specification: ..., Controller: controller,
            ReportingStrategy: cache.ReportAtLeastEvery(time.Hour)},
        SensorTempConfig: &numericsensor.Config{...Reporter: controller...},
        MeterElecConfig:  &numericmeter.Config{...Reporter: controller...},
    }
    return thing.NewThermostat(publisher, ts, cfg), nil
}
```

Use a prefab thing (`thing.NewThermostat`, `thing.NewChargepoint`, ...) when it fits; otherwise
assemble `[]adapter.Service` and call `adapter.NewThing(...)` (sensibo pattern).

**Create/destroy = seeds + EnsureThings**: build `adapter.ThingSeeds`
(`ThingSeed{ID, Info, CustomAddress}`) from the device list and call `adapter.EnsureThings(seeds)`
— it creates missing things and destroys removed ones, emitting inclusion/exclusion reports.
**`CustomAddress` must reproduce the legacy FIMP address** so devices survive the upgrade without
re-inclusion. Other methods: `InitializeThings`, `Things`, `ThingByID`, `CreateThing`,
`DestroyThingByID`, `DestroyAllThings` (pass as `RouteApp`'s exclude-all callback + use in
`Uninstall`). If service topology can change in place, checksum it and rebuild the thing
(sensibo `reconcile.go`).

## 5. Polling and async transports

`internal/tasks/tasks.go`:

```go
return task.Combine[[]*task.Task](
    app.TaskApp(application, appLifecycle),                      // runs Check() every CheckInterval
    cliffAdapter.TaskAdapter(adapter, cfg.GetPollingInterval()),
    thing.TaskThermostat(adapter, cfg.GetPollingInterval(), task.WhenAppIsConnected(appLifecycle)),
)
```

Rules:
- **REST cloud polling must be ≥ 60 s.** Enforce a floor in `Config.GetPollingInterval()`.
  Respect vendor rate limits (HTTP 429/402) — back off, don't tighten the loop.
- **Prefer an async channel over polling if the vendor offers one**: LAN REST (mill),
  UPnP/GENA events (sonos), SignalR (easee), AMQP (zaptec), webhooks, MQTT. Wire it as a
  `root.Service` manager + per-thing `adapter.Connector.Connect(thing)` registering a handler
  that calls `svc.Send...Report(false)`. Keep a slow REST poll as reconciliation.
- Gate tasks with `task.Voter`s so polling stops when disconnected (custom voters for
  "connected OR locally reachable", mill `whenConnectedOrLocal`).

## 6. Reporting strategy

Cliffhanger `adapter/cache` strategies, set per service via `Config.ReportingStrategy`:
- `cache.ReportOnChangeOnly()` — only on change.
- `cache.ReportAtLeastEvery(time.Hour)` — on change, plus an hourly heartbeat. **This is the
  default for variable values**: thermostat mode, setpoint, sensor temperature, meter readings.
- Static/slow parameters (settings, supported ranges, max setpoint) are NOT reported
  periodically — they live in the inclusion report / extended config report and are re-sent only
  on change (forced inclusion report) or on explicit `cmd.*.get_report`.

Every `Send...Report(force bool)` consults the strategy when `force=false` (polling, push
handlers); `force=true` bypasses it (explicit get_report commands, post-inclusion sync).
Suppress sensor jitter in the controller before the cache (mill `ambientReportDelta = 0.3`).

## 7. Services — climate device assembly

Controller implements the service interfaces (one struct, several roles):

```go
var (
    _ thermostat.Controller  = (*Controller)(nil)  // SetThermostatMode/Setpoint, *Report()
    _ numericsensor.Reporter = (*Controller)(nil)  // NumericSensorReport(unit)
    _ numericmeter.Reporter  = (*Controller)(nil)  // MeterReport(unit)
    _ adapter.Connector      = (*Controller)(nil)  // Connectivity(), Ping()
)
```

Specs (mill):

```go
thermostat.Specification(name, adapterAddr, thingAddr, groups,
    []string{"off", "heat"}, []string{"heat"}, []string{"idle", "heat"})
numericsensor.Specification(name, adapterAddr, numericsensor.SensorTemp, thingAddr, groups,
    []string{numericsensor.UnitC})
numericmeter.Specification(numericmeter.MeterElec, name, adapterAddr, thingAddr, groups,
    []numericmeter.Unit{numericmeter.UnitW, numericmeter.UnitKWh})
```

Clamp setpoints to the vendor's range in `SetThermostatSetpoint`. Preserve legacy group/channel
suffixes (`ch_0`...) for hub-upgrade compatibility.

## 8. Config — model, service, migrations

```go
type Config struct {
    config.Default                      // MQTT, LogLevel, WorkDir, ConfiguredAt
    PollingInterval string      `json:"pollingInterval"`
    Credentials     Credentials `json:"credentials"`
    ...
}
func New(workDir string) *Config { return &Config{Default: config.NewDefault(workDir)} }
func Factory() *Config           { return &Config{} }   // for RouteApp
```

`Service` embeds `storage.Storage[*Config]` + `sync.RWMutex`; every setter stamps
`ConfiguredAt` and `Save()`s. Provide `DefaultStore()`, `GetPublicConfig()` (credentials
stripped), and `Migrate()`.

**Migrating legacy config/state files.** Old adapters persist different shapes
(`data/config.json` with flat fields, `data.json`, no `adapter.json` at all). The port must read
the old files once and translate:
- `config.json`: map legacy fields (e.g. `poll_time_min: "5"` minutes → `pollingInterval: "5m"`,
  legacy token fields → `credentials`), drop cached API state. Use cliffhanger versioned
  migrations: `m.Migrate(config.Migration{From: 0, To: 1, Do: func() error {...}})` — the 0→1
  migration MUST forcefully set `m.LogLevel = "info"` and `m.LogFormat = "budzik"` (mill
  `config.go` Migrate is the reference) — or a
  custom unmarshal of the legacy shape when fields moved (sensibo migrates a legacy bearer-key
  file into `credentials.accessToken`; netatmo `MigrateClientID`).
- `adapter.json` (thing registry): legacy adapters that hand-published inclusion reports have no
  adapter.json — recreate things via `EnsureThings` with `CustomAddress` = the legacy service
  address so FIMP addresses (and therefore the hub's device registry) are preserved. Adapters
  with an old-format adapter.json need a rewrite pass like zaptec
  `internal/migration/adapterstate.go` (`MigrateAdapterState(workDir)` re-keys thing records,
  returns an old→new mapping fed to `MigrateSelectedDevices`).
- Absolute paths inside migrated config are rewritten by packaging's `migrate.sh` (§13), not in Go.

## 9. Auth patterns

- **Username/password → JWT + refresh** (mill `internal/mill/auth.go`): `Authenticator` caches
  the token, refreshes on expiry with `backoff.NewStateful`, forces app logout on refresh
  failure/401 (`AuthStateLost` + push notification).
- **OAuth authorization-code** (netatmo): tokens arrive via `cmd.auth.set_tokens`
  (implement `app.AuthorizableApp`); store `AccessToken/RefreshToken/ExpiresAt` in config.
- **Hub-proxied auth** (legacy adax pattern): device API called through the Futurehome proxy
  (`auth.ProxyURL(env)` + hub token from `hub.NewTokenLoader`); only a one-time vendor call to
  bind the account. No refresh logic needed on the hub.
- **Plain API key** (sensibo): `credentials.accessToken` used directly.

Always: credentials live in the persisted config, stripped by `GetPublicConfig`, cleared by
`Logout`/`Uninstall`. `LogStats` masks `cmd.auth.*` payloads out of logs.

## 10. Routing

```go
return router.Combine(
    bootstrap.DefaultRoute(ServiceName, cfgSrv.GetPublicConfig, tel),
    []*router.Routing{
        cliffConfig.RouteCmdConfigGetDuration(ServiceName, "polling_interval", cfgSrv.GetPollingInterval),
        cliffConfig.RouteCmdConfigSetDuration(ServiceName, "polling_interval", cfgSrv.SetPollingInterval),
        // Get/Set String, Bool, StringMap variants for each app setting
    },
    app.RouteApp(ServiceName, appLifecycle, cfgSrv, config.Factory, locker, application, adapter.DestroyAllThings),
    cliffAdapter.RouteAdapter(adapter),
    thing.RouteThermostat(adapter),          // per-service routes for each service used
)
```

`locker := router.NewMessageHandlerLocker()` serializes config mutations. `RouteApp`
auto-provides `cmd.app.get_state/get_manifest/uninstall`, `cmd.config.get_extended_report/
extended_set`, and `cmd.auth.login/logout/set_tokens` from the interfaces the app implements.

## 11. Tests

- **mockery v2.53.5**, config in `src/.mockery.yaml` (mill's is the model):

```yaml
with-expecter: true
disable-version-string: true
issue-845-fix: true
resolve-type-alias: false
dir: "internal/test/mocks/{{.PackageName}}"
mockname: "{{.InterfaceName}}"
outpkg: "mock{{.PackageName}}"
filename: "{{.InterfaceName}}.go"
packages:
  github.com/futurehomeno/edge-<name>-adapter/internal/<vendor>:
    interfaces: { Client: }
  github.com/futurehomeno/cliffhanger/adapter:
    interfaces: { Adapter:, ThingState:, Publisher: }
```

  Mock both project interfaces and consumed cliffhanger interfaces. `make generate-mocks`.
- Unit-test the controller against a mocked client; e2e-test FIMP round trips via
  `cmd/testing.go` bootstrap + a local mosquitto broker (port 11883, `make broker`).
- **Coverage: target > 80%, hard gate at 75% per package** enforced by `make test`
  (`scripts/coverage_gate.awk`, dedups `-coverpkg` blocks by position; skips the main package).
  CI has no separate threshold — the Makefile gate is the gate.

## 12. Tooling — unified Makefile / go.yml / golangci

**Makefile** (sensibo base + mill's `check-mocks`/`broker`): targets `build-local`, `build-arm`
(GOARM=6), `build-linux-amd64`, `lint`, `generate-mocks`, `check-mocks`, `broker`, `test`
(with coverage gate), `configure`, `package-deb` (staging from `git ls-files` so untracked files
never ship), `deb-arm`, `run`, `upload/deploy/install`. Common ldflags:
`-ldflags="-s -w -X main.Version=$(VERSION) -X main.PackageName=$(APP_NAME)"`.
`COVERAGE_THRESHOLD := 75`, `MQTT_PORT := 11883`.

**go.yml** (sensibo's): jobs `setup → build → linter → test → release`; Go `1.26`
(`actions/setup-go@v5`); mockery installed and mocks shared as artifact; lint =
`golangci/golangci-lint-action@v7` (v2.12.2) + `staticcheck@latest`; test spins
`eclipse-mosquitto:2` on 11883 then `make test`; release on `v*` tags with
`softprops/action-gh-release@v2` (never SHA-pin actions — major tags only), `needs: [test,
linter]`, first-release-safe changelog. `GOPRIVATE=github.com/futurehomeno/*` + insteadOf with
`GH_OAUTH_TOKEN`.

**.golangci.yaml**: v2 config from netatmo/sonos (includes `paralleltest`/`tparallel`,
gofumpt/gci with `prefix(github.com/futurehomeno/edge-<name>-adapter)`, test-file exclusions).

Substitute per project: `APP_NAME`, module path (golangci prefix, mockery package keys,
coverage-gate module skip string), `VERSION`, artifact/deb names, systemd unit, container name.

## 13. Packaging — thingsplex → futurehome migration

Legacy layout: `/opt/thingsplex/<app>/` (binary + data + defaults), logs
`/var/log/thingsplex/<app>`. New layout (zaptec/sensibo):

```
/usr/bin/<app>                              binary
/usr/lib/futurehome/<app>/migrate.sh        one-time state migration (run via fh-drop)
/usr/lib/systemd/system/<app>.service       ExecStart=/usr/bin/<app> -c /var/lib/futurehome/<app>
/usr/share/futurehome/<app>/defaults/       read-only defaults (dpkg-owned)
/var/lib/futurehome/<app>/defaults          symlink -> /usr/share/futurehome/<app>/defaults
/var/lib/futurehome/<app>/data              state + credentials (0750, postinst-created)
/var/log/futurehome/<app>/                  logs
```

- `control`: `Depends: fh-drop`; maintainer `Futurehome AS <dev@futurehome.no>`.
- `preinst`: stop service; on upgrade remove old real `defaults` dir so dpkg can unpack the symlink.
- `postinst configure`: create `futurehome` group + `<app>` system user → dirs → ownership →
  `fh-drop <app> /usr/lib/futurehome/<app>/migrate.sh` → enable+start service.
- `migrate.sh`: if new `data/` absent and `/opt/thingsplex/<app>/data` exists, copy via temp dir +
  `sync -f` + atomic `mv`; `sed`-rewrite absolute paths in `config.json`
  (`/opt/thingsplex/<app>` → `/var/lib/futurehome/<app>`, log dir likewise); best-effort log copy.
  Never deletes the old tree (downgrade-safe). `umask 027`, refuses to run as root.
- `prerm/postrm`: stop/disable on remove; delete data+logs only on purge.
- FIMP identity (resource name, topics `pt:j1/.../rt:ad/rn:<app>/ad:1`) is unchanged by this —
  it is purely a filesystem/packaging move.

## 14. FIMP compatibility — hard requirement

The port must be wire-compatible with the legacy adapter. The hub's device registry, flows and
apps key on these identifiers — changing any of them breaks every existing installation:

- **Resource name** (`rn:<name>` in topics, `WithServiceDiscovery` resource) — keep identical.
- **Service names** (`thermostat`, `sensor_temp`, `meter_elec`, ...) — same set, same names.
- **Service addresses** (`/rt:dev/rn:<name>/ad:1/sv:<svc>/ad:<id>`) — preserve the legacy `<id>`
  via `ThingSeed.CustomAddress`.
- **Interfaces + value types** (`cmd.setpoint.set` str_map, `evt.sensor.report` float, units,
  props like `sup_modes`) — the new inclusion report must be a superset of the legacy one.
- Groups/channels (`ch_0`), `ManufacturerId`/`ProductHash` — keep legacy values.

Verify by diffing a captured legacy inclusion report + evt messages against the ported ones
before release.

## 15. Porting checklist

1. Map the legacy adapter: services + FIMP addresses emitted, config file shape, auth flow,
   polling interval, packaging paths. Record legacy addresses — they must survive.
2. Verify FIMP compatibility plan per §14 (resource name, service names, addresses, interfaces).
3. Check the vendor's public API docs for conformity (endpoints, auth, units, rate limits).
4. Scaffold from mill: `main.go`, `cmd/{root,factory,testing}.go`,
   `internal/{app,config,routing,tasks,adapter,<vendor>}`.
5. Config package + legacy config migration (§8).
6. API client + auth in `internal/<vendor>` (context-aware, typed errors for 401/429).
7. Controller implementing the service interfaces; ThingFactory; `EnsureThings` with
   `CustomAddress` preserving legacy addresses.
8. App: Configure/Initialize/Check/Login(or SetTokens)/Logout/Uninstall.
9. Routing + tasks; polling floor ≥60s; `ReportAtLeastEvery(time.Hour)` on variable values.
10. Tooling: Makefile, go.yml, .golangci.yaml, .mockery.yaml, coverage gate, pre-commit hook.
11. Packaging: futurehome layout + migrate.sh + maintainer scripts.
12. Tests: mocks, controller/app/config/routing tests, e2e; coverage >80% (gate 75%).
13. Update app-manifest.json (auth block, settings), defaults/config.json, README.
14. Verify: `go vet`, `golangci-lint run`, `make test`, `make build-arm`, `make deb-arm`.

## 16. Commonly forgotten

- `WithTelemetry` + `router.WithStatsCallback(routing.LogStats)` (with `cmd.auth.*` masking).
- `task.Voter` gates — polling must stop when logged out.
- Forced vs cached reports: `force=true` only for explicit get_report / inclusion sync.
- `DestroyAllThings` wired into both `Uninstall()` and `RouteApp`'s exclude-all callback.
- Legacy group labels / channel suffixes in specifications (hub-upgrade compatibility).
- `config.Factory()` (zero-value factory) for `RouteApp` — not `config.New`.
- Manifest auth block endpoints (often environment-dependent via `auth.ProxyURL(env)`).
- `.githooks/pre-commit` + `make init` to install it.
- Don't SHA-pin GitHub Actions; use major version tags.
- Delete dead legacy code after the port (old router/poller/processor packages).
