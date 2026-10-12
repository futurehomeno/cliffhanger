# Diagnostics And Logging Specification

## Purpose
Defines how an adapter writes, rotates and remotely controls its log file on the hub, how it retains
recent warnings and errors for the app diagnostic report, and how it surfaces operational signals to
the user. The hub stores logs on eMMC, so log writing is batched rather than line-by-line, and
verbose levels are a bounded, self-reverting operator window instead of a permanent setting.

## Requirements

### Requirement: Logger Initialization And Rollback
`debug.InitializeLogger(store)` SHALL apply the persisted format and level, then validate and wire a
new buffered file output before tearing down any previous log manager. When the new output cannot be
opened, a previous manager SHALL be restored together with the formatter and level that were in
effect before the call, so logging keeps flowing through a manager that is still open. A first
initialization that fails SHALL keep the new manager rather than leaving the package global nil —
`debug.Route` panics on a nil manager, and an application that logs the error and carries on then
runs without file logging instead of crash-looping. A superseded manager SHALL have its flusher
stopped and its buffered writer closed (which flushes it), because nothing else holds a reference to
its up to 64 KiB of pending lines. An unparsable persisted level SHALL fall back to info and SHALL
NOT prevent startup.

#### Scenario: re-initialization with a bad log path
- **WHEN** `InitializeLogger` is called a second time with a log file that cannot be opened for
  writing
- **THEN** the error is returned, the previous manager, formatter and level stay in effect, and no
  buffered lines are lost

#### Scenario: garbage log_level in config.json
- **WHEN** the persisted level is `"verbose"`
- **THEN** a warning is logged, the level becomes info, and initialization continues

### Requirement: Buffered Rotating Log Output
Log lines SHALL be buffered in a 64 KiB in-memory buffer ahead of a lumberjack rotating file capped
at 5 MiB with 4 retained backups. The buffer SHALL be flushed when it fills, on the periodic tick,
and immediately after writing any entry at error level or above. The urgency of an entry SHALL be
determined by the formatter wrapper, which records the entry level immediately before logrus writes
the line under the same logger mutex; a level hook cannot serve here because hooks fire before the
entry is written and so cannot flush the very line that triggered them. Before wiring a new output
the target path's directory SHALL be created and the file SHALL be opened for writing exactly as
lumberjack will, so a path that only passes a read-only probe is rejected up front rather than at
the first flush. `debug.FlushLogs()` SHALL flush the current output and SHALL be a no-op when no
manager exists; the app container calls it on startup failure and on shutdown so the closing lines
survive a deliberate exit.

#### Scenario: an error is logged
- **WHEN** an entry at error, fatal or panic level is written
- **THEN** the buffer is flushed to the rotating file before the write returns

#### Scenario: unwritable target path
- **WHEN** the requested log file exists but cannot be opened for writing
- **THEN** `setLogOutput` fails and the previous output keeps receiving lines

### Requirement: Flush Cadence Follows Verbosity
The periodic flusher SHALL use a 240 second interval by default, or the store's
`log_flush_interval` when it is positive, and SHALL switch to 2 seconds while the level is debug or
trace and the revert deadline has not elapsed — someone enabling debug is about to tail the file,
and the long interval would make a running adapter look hung. The flusher SHALL re-evaluate the
interval on every tick, so an elapsing revert deadline or a changed configured interval takes effect
on its own within one tick rather than only on the next level change. A level change SHALL restart
the ticker only when it actually changes the resulting cadence, so an unrelated change (info to
warn, or re-setting the same level) does not push an almost-due flush back by a full interval.

#### Scenario: debug window ends without any command
- **WHEN** the revert deadline elapses while the level is still debug
- **THEN** within one 2 second tick the flusher returns to the configured (or 240 second) interval

#### Scenario: level changed between two non-verbose levels
- **WHEN** the level goes from info to warn
- **THEN** the flush ticker is not restarted

### Requirement: Self-Reverting Verbose Levels
Setting a level of debug or trace SHALL persist a revert deadline of now plus the configured
`log_revert_timeout`, defaulting to `7 * 24h`, and SHALL write that deadline **before** the level, so
a partial failure leaves the adapter on the previous, less verbose level rather than stranded on a
verbose level with no deadline. Setting a level below debug SHALL clear the deadline. At startup the
persisted level SHALL be applied without re-arming the deadline; if it is debug or trace and the
deadline has already elapsed, the level SHALL be reset to info, persisted, and the deadline cleared.
`SetRevertTimeout` SHALL reject a non-positive duration and SHALL recompute an already-armed
deadline from now.

#### Scenario: hub restarts inside the debug window
- **WHEN** an adapter with a persisted debug level and a future deadline starts
- **THEN** debug is applied and the existing deadline is left untouched, not extended by the restart

#### Scenario: hub restarts after the window
- **WHEN** the persisted deadline has passed
- **THEN** the level is persisted as info, the deadline is cleared, and an explanatory line is logged

### Requirement: Log Format Selection
The log format SHALL be selected by the persisted `log_format` value: `json` for a logrus JSON
formatter with timestamps `2006-01-02 15:04:05.999`, `budzik` for the compact `BudzikFormatter`, and
anything else for a full-timestamp colored text formatter. The chosen formatter SHALL always be
wrapped so entry urgency is recorded for the buffered writer. `SetFormat` SHALL apply the format
before persisting it.

#### Scenario: unknown format requested
- **WHEN** `cmd.log.set_format` carries an unrecognised value
- **THEN** the default text formatter is applied and the value is persisted

### Requirement: Budzik Log Line Shape
`BudzikFormatter` SHALL render one line as `<timestamp> <level> <message>` followed by the entry's
fields as ` key=value` pairs sorted by key, terminated by a newline. The timestamp format SHALL be
`06 01-02 15:04:05` and the levels SHALL be rendered as `PANIC`, `FATAL`, `E`, `W`, `I`, `D`, `T`,
with `D` used for any level index outside that table.

#### Scenario: entry with fields
- **WHEN** a warn entry with fields `b=2` and `a=1` is formatted
- **THEN** the line ends with ` a=1 b=2`, sorted by key, so successive lines stay diffable

### Requirement: Log Control Routing
`debug.Route(serviceName, options...)` SHALL panic when called before `InitializeLogger`, and
otherwise SHALL expose get/set pairs for the level (`cmd.log.get_level` / `cmd.log.set_level` →
`evt.log.level_report`), format (`cmd.log.get_format` / `cmd.log.set_format` →
`evt.log.format_report`), file (`cmd.log.get_file` / `cmd.log.set_file` → `evt.log.file_report`) and
revert timeout (`cmd.log.get_revert_timeout` / `cmd.log.set_revert_timeout` →
`evt.log.revert_timeout_report`). Each successful set SHALL publish a configuration-change event
through the supplied routing options. `cmd.log.set_file` SHALL accept only a plain file name and
SHALL reject an empty value, `.`, `..`, `/` or any value containing a path separator. A relative name
SHALL be resolved against the directory of the currently configured log file and persisted as an
absolute path, so a restart under systemd does not send the logs to whatever working directory the
unit happens to have. `cmd.log.get_revert_timeout` SHALL report the default when none is persisted.

#### Scenario: path injection attempt
- **WHEN** `cmd.log.set_file` carries `../../etc/passwd`
- **THEN** an error is returned and the log output is unchanged

#### Scenario: file changed after a failed initialization
- **WHEN** `cmd.log.set_file` succeeds on a manager whose initial output could not be opened
- **THEN** the periodic flusher is started, so entries below error level no longer sit in the buffer
  until it fills

### Requirement: Recent Errors Ring Buffer
`formatters.ErrorHook` SHALL capture warn, error, fatal and panic entries into a fixed ring buffer of
64 entries (`MaxLogEntries`), overwriting the oldest once full, and SHALL render each entry with a
colorless text formatter and no trailing newline. `ErrorsReport` SHALL return the retained entries
oldest-first, dropping entries older than 30 days (`LogRetention`) from the head first, and SHALL
satisfy `diagnostic.ErrorsReporter` so it can back `cmd.app.get_diag` directly. `app.NewLogCapture`
SHALL install exactly one such hook on the global logger and return the same instance on subsequent
calls, so repeated app construction does not stack duplicate hooks.

#### Scenario: more than 64 warnings
- **WHEN** 100 warn entries are logged
- **THEN** `ErrorsReport` returns the most recent 64 in chronological order

#### Scenario: stale entries
- **WHEN** the buffer holds entries older than 30 days
- **THEN** they are dropped from the report and freed from the buffer

### Requirement: Disk Space Monitor
`monitor.NewDiskSpace(interval, limitPercent)` SHALL create a `root.Service` that samples the used
percentage of the `/` filesystem on each tick of `interval`. `DiskFull()` SHALL report whether the
last sampled usage is greater than or equal to `limitPercent`, and SHALL take a sample on demand
when no usage has been recorded yet, so it is usable before the first tick or without `Start`. A
failed sample SHALL be logged and SHALL leave the previous reading in place. `Start` SHALL fail when
already running and `Stop` SHALL fail when already stopped, and `Stop` SHALL wait for the sampling
goroutine to exit. A panic inside the sampling loop SHALL be logged with its stack and re-panicked.

#### Scenario: queried before the first tick
- **WHEN** `DiskFull()` is called on a monitor whose usage is still zero
- **THEN** a disk usage sample is taken synchronously and compared against the limit

#### Scenario: double start
- **WHEN** `Start` is called on a running monitor
- **THEN** an error is returned and no second goroutine is spawned

### Requirement: Push Notifications
`notification.Notification` SHALL publish push notification events as an
`evt.notification.report` string-map message on the `kind_owl` service to
`pt:j1/mt:evt/rt:app/rn:kind_owl/ad:1`. The payload SHALL always carry the keys `EventName`,
`MessageContent`, `DeviceId`, `DeviceName`, `RoomId`, `RoomName`, `AreaId`, `AreaName` and
`AreaType`, with any zero numeric ID rendered as an empty string rather than `"0"`. `Message(text)`
SHALL be shorthand for an event named `custom` carrying that text, and `EventWithProps` SHALL attach
the given properties to the FIMP message.

#### Scenario: event without a device
- **WHEN** an `Event` with `DeviceID` 0 is published
- **THEN** the payload's `DeviceId` is an empty string

### Requirement: Timeline Entries
`notification.Timeline` SHALL publish `cmd.timeline.set` string-map messages to
`pt:j1/mt:cmd/rt:app/rn:time_owl/ad:1`. `Event` SHALL carry `EventName`, `MessageContent`,
`DeviceId`, `DeviceName`, `RoomName` and `AreaName`, with zero IDs rendered as an empty string, and
`Message` SHALL carry `sender`, `message_en` and `message_no` for a pre-translated entry.

#### Scenario: localized timeline message
- **WHEN** `Message` is called with English and Norwegian texts
- **THEN** a single `cmd.timeline.set` message carrying both translations and the sender is published
