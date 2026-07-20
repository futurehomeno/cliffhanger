---
name: port_to_cliffhanger_1.2
description: Port a Futurehome edge adapter (legacy fimpgo or cliffhanger v0.x) to cliffhanger v1.2.10, or build a new adapter on it. Use when asked to port/migrate an edge adapter to cliffhanger, modernize an adapter, or align an adapter with the mill/sensibo/zaptec architecture (futurehome packaging, unified Makefile/CI, mockery tests).
---

# Port an edge adapter to cliffhanger v1.2.10

Read [references/porting-manual.md](references/porting-manual.md) first — it contains the full
architecture, package structure, template table, and code patterns. This file is the procedure.

Reference projects are the sibling `edge-*-adapter` repos in this workspace (here, `~/proj/go/`;
adjust to wherever they're checked out): edge-mill-adapter (primary template), edge-sensibo-adapter
(tooling/packaging golden copy), edge-zaptec-adapter (packaging migration + adapter.json migration),
edge-sonos/easee (push transports), edge-netatmo (OAuth). Cliffhanger source is in the Go module
cache (`$(go env GOMODCACHE)/github.com/futurehomeno/cliffhanger@v1.2.10/`).

## Procedure

1. **Survey the legacy adapter.** Record: FIMP services + interfaces emitted, exact device
   addresses (`sv:<svc>/ad:<id>`), config file shape and location, auth flow, polling interval,
   packaging paths (`/opt/thingsplex/...`?), quirks (unit scaling, clamps, rate limits).
   **FIMP messages must stay wire-compatible** (manual §14): same resource name (`rn:`), same
   service names, same service addresses (via `ThingSeed.CustomAddress`), same interfaces and
   value types — diff legacy vs ported inclusion reports and evt messages to prove it.

2. **Check vendor API conformity.** Find the vendor's public API docs (and OpenAPI spec if
   published). Compare with what the legacy code calls; note auth model, units, rate limits.
   Decide: keep the existing auth architecture (e.g. Futurehome proxy) unless told otherwise.
   If the vendor offers an async channel (LAN API, websocket, SignalR, AMQP, UPnP), prefer it
   over polling, with a slow REST poll as reconciliation.

3. **Decisions before coding** (confirm with the user if ambiguous): thing granularity
   (device vs room), service set, prefab thing vs generic assembly, polling default
   (**floor 60 s for REST cloud polling**), what legacy config fields map to.

4. **Scaffold** per manual §1–§2: `main.go`, `cmd/{root,factory,testing}.go`,
   `internal/{app,config,routing,tasks,adapter,<vendor>}`. Copy the shape from mill; keep
   `go.mod` at cliffhanger v1.2.10 and the Go version used by the reference projects.

5. **Implement in order**: config (+ legacy migration, manual §8; the 0→1 migration must
   forcefully set `LogLevel = "info"` and `LogFormat = "budzik"`) → API client + auth →
   controller + thing factory → app (Check per manual §3) → routing → tasks.
   Reporting: `cache.ReportAtLeastEvery(time.Hour)` for variable values (setpoint, mode,
   temperature, meter); static parameters are never reported periodically.

6. **Tooling** (manual §12): copy sensibo's Makefile + go.yml, add mill's `check-mocks`/`broker`
   targets, netatmo/sonos `.golangci.yaml`, mill's `.mockery.yaml`; substitute app name, module
   path, version. Coverage gate 75% in `make test`, aim > 80%. Never SHA-pin actions.

7. **Packaging** (manual §13): futurehome layout (`/usr/bin`, `/var/lib/futurehome`,
   `/usr/share/futurehome`, `/var/log/futurehome`), zaptec's maintainer scripts + `migrate.sh`
   adapted to the app (path sed rules for the legacy config).
   **Gitignore trap**: a bare `<app>` ignore pattern (for the built binary) also swallows every
   packaging path containing `/<app>/` (`usr/lib/futurehome/<app>/migrate.sh`, `usr/share/.../defaults/`)
   — the local `deb-arm` still builds, so it ships silently broken. Anchor binary ignores
   (`/<app>`, `src/<app>`) and verify `git ls-files package/` matches the on-disk tree.

8. **Tests**: `make generate-mocks`; unit tests for controller/config/app/routing against mocks;
   e2e FIMP round trips with local mosquitto (`make broker`). Iterate until every package clears
   the 75% gate and total is > 80%.
   **Goroutine lifecycle in ported engines**: legacy restart paths often re-enter Init and
   re-spawn worker/ticker goroutines with no shutdown, leaking one per stop/start cycle.
   When porting, guard spawns (sync.Once) or give loops a stop channel closed in Stop(),
   and order side effects so a failed Start doesn't leave tickers/transports running.

9. **Verify**: `go vet ./...`, `golangci-lint run`, `make test`, `make build-arm`, `make deb-arm`.
   Delete dead legacy packages. Update README, app-manifest.json, defaults/config.json.

10. **Review logs**: if the `improve_logs` skill is available, invoke it on the ported code;
    otherwise review log statements by hand for the conventions (wrap-then-log-once,
    [component] prefixes, appropriate levels).

## Output expectations

- A working adapter on cliffhanger v1.2.10 with the manual's package structure.
- Legacy users upgrade seamlessly: same FIMP addresses, migrated config/state, packaging
  migration from /opt/thingsplex.
- CI green: build, lint (golangci-lint v2 + staticcheck), tests with coverage gate.
