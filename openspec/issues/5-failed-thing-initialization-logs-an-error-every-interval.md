# 5 A failed thing initialization logs an error on every reporting interval, without end

## What happened

Found 2026-11-25 in the zaptec 3.4.2 beta logs. On site `006d6de7` the Zaptec user is logged out, so the adapter
cannot create its charger at start-up. `handleInitialization` (`adapter/tasks.go:32`) then runs on the reporting
interval for as long as the adapter is not initialized and logs at error level each time:

    E [adapter] Initialize things. err: failed to create thing with address 1: [ZAP275612] retrieve states err: /api/chargers/…/state: not logged in

- 3307 lines between the upgrade (2026-11-24 03:38) and the log's end (11-25 07:11), one every 30 s.
- 10287 more of the same line on that hub before the upgrade, so it is not new in zaptec 3.4.2.
- 1 of 49 hubs on zaptec 3.4.2. Four other logged-out hubs do not show it: their thing already existed.

The error level puts it in the log at the normal INFO log level. It fills the log and pushes older lines out. The
loop has no backoff and does not hold back a repeated, unchanged error.

Unknown:

- Whether each retry sends a request to the vendor API or fails locally (in zaptec, "not logged in" may be a local
  check). If it sends, the retry also loads the vendor API.
- Whether other adapters on cliffhanger show the same line when their start-up fails; their logs were not checked.

## How to reproduce

Log: `~/proj/rollo/logs/beta/2026-11-25-zaptec342/zaptec/zaptec_006d6de7-*.log`,
`grep -ac 'Initialize things' <file>`. In code: an adapter whose `InitializeThings` keeps returning an error; the
task in `adapter/tasks.go` logs it once per reporting interval.

## What would close it

The initialization task logs a failure once, and again only when the error text changes or the initialization
succeeds, or it retries with a growing interval. A test with an `InitializeThings` that always fails checks the
number of error lines over several intervals.
