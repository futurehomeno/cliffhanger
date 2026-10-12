## ADDED Requirements

### Requirement: Write-Through After Stop
The app container's stop SHALL call `debug.SetLogWriteThrough(true)`, which flushes the current
output and writes every later line straight to the log file, whichever output is wired then, so
lines logged after `Stop()` or `Run()` returns (an adapter's deferred stop banner) survive the
process exit. The app container's start SHALL call `debug.SetLogWriteThrough(false)`, so buffering
while the app runs is unchanged.

#### Scenario: stop banner logged after stop
- **WHEN** an info line is logged after the app has stopped
- **THEN** the line is in the log file without any further flush

#### Scenario: restarted app
- **WHEN** a stopped app is started again
- **THEN** info lines are buffered again
