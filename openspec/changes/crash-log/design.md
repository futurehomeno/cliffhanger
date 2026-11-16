# Design

## Separate file, not the app log

The runtime writes crash output to a file descriptor, outside logrus and its buffer. `<app>.log` is rotated by lumberjack by renaming it, and a descriptor held by the runtime would follow the renamed backup, so crashes would land in a backup that later gets deleted. `crash.log` is not lumberjack's file, so it is never renamed under the runtime; `SetCrashOutput` duplicates the descriptor, and the file is closed right after the call.

## Size bound

Crashes are rare, but a crash loop appends a stack dump per restart. At setup, a `crash.log` over 1 MiB replaces `crash.log.1`: at most about 2 MiB in total, and the newest crash is always in `crash.log`.

## Start marker

The runtime's crash text carries no time or version. One line per start, `--- Start <app> v<version> <RFC3339 time> ---`, matches the shape of the adapters' log banner, which the log analyzer scopes on. The log file is wired before the app is built, and only the builder knows the package name and version, so the marker is written by whichever of the two comes second, once per process and `crash.log` path. Wiring the same log directory again (a `set_file` to a new name) re-arms nothing.
