# Write unhandled panics to crash.log

## Why

Go has no global panic handler. A panic in a goroutine without a deferred recover (an app's own `go func()`, or an unprotected cliffhanger one) and every unrecoverable runtime fatal (`concurrent map writes`, out of memory, stack overflow) print only to stderr. Adapter systemd units send stderr to null or the journal, which hub log collection does not see: the log file shows a Start banner and no Stop, with no cause.

## What Changes

- Wiring the log file also opens `crash.log` next to it and hands it to `runtime/debug.SetCrashOutput`, so the runtime writes the panic and the goroutine stacks there as well as to stderr.
- `crash.log` larger than 1 MiB at setup is moved to `crash.log.1` first.
- `debug.SetCrashLogApp(name, version)` names the app; each start appends `--- Start <app> v<version> <RFC3339 time> ---` to `crash.log`. The root app builder calls it with the service discovery package name and version.
- Failures here log a warning and never fail logging setup.

## Impact

`debug/log_manager.go`, `root/builder.go`, spec `platform/diagnostics`. Adapters need no change: they get it on their next cliffhanger bump.
