## MODIFIED Requirements

### Requirement: Buffered Rotating Log Output
Log lines SHALL be buffered in a 64 KiB in-memory buffer ahead of a lumberjack rotating file capped
at 5 MiB with 4 retained backups. The buffer SHALL be flushed when it fills, on the periodic tick,
and immediately after writing any entry at error level or above. The urgency of an entry SHALL be
determined by the formatter wrapper, which marks the formatted bytes of an entry at error level or
above; the buffered writer SHALL flush when it writes marked bytes. logrus formats outside its
mutex and writes under it, so a single flag would be overwritten by other goroutines' lines in
between; and a level hook cannot serve because hooks fire before the entry is written. Marks for
lines that never reach the buffered writer SHALL be bounded. Before wiring a new output
the target path's directory SHALL be created and the file SHALL be opened for writing exactly as
lumberjack will, so a path that only passes a read-only probe is rejected up front rather than at
the first flush. `debug.FlushLogs()` SHALL flush the current output and SHALL be a no-op when no
manager exists; the app container calls it on startup failure and on shutdown so the closing lines
survive a deliberate exit.

#### Scenario: an error is logged
- **WHEN** an entry at error, fatal or panic level is written
- **THEN** the buffer is flushed to the rotating file before the write returns

#### Scenario: an error is logged while other goroutines log
- **WHEN** an entry at error level is logged while other goroutines keep logging info lines
- **THEN** the error line is in the rotating file before the logging call returns

#### Scenario: unwritable target path
- **WHEN** the requested log file exists but cannot be opened for writing
- **THEN** `setLogOutput` fails and the previous output keeps receiving lines
