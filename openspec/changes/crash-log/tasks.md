## 1. Crash output

- [x] 1.1 Failing tests: wiring the log file creates `crash.log` beside it with the start marker; a goroutine panic without recover in a child process lands in `crash.log` after the marker; a `crash.log` over 1 MiB moves to `crash.log.1`
- [x] 1.2 `setLogOutput` sets the crash output, `debug.SetCrashLogApp`, root builder calls it
