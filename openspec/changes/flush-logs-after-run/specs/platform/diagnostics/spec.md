## ADDED Requirements

### Requirement: Write-Through After Run
When the app container's `Run()` returns, it SHALL call `debug.DisableLogBuffering()`, which
flushes the current output and makes it write every later line straight to the log file, so lines
logged after `Run()` (an adapter's deferred stop banner) survive the process exit. Buffering during
normal operation SHALL be unchanged; a later `InitializeLogger` wires a new, buffered output.

#### Scenario: stop banner logged after Run
- **WHEN** an info line is logged after `Run()` has returned
- **THEN** the line is in the log file without any further flush
