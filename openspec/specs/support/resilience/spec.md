# Resilience Primitives Specification

## Purpose
Defines the shared building blocks every adapter uses to survive an unreliable vendor cloud: the
three-stage backoff, the HTTP error contract the rest of the framework keys on, the supervisor that
keeps one long-lived streaming connection alive, and the small utilities that carry behaviour rather
than convenience — log throttling, out-of-order rejection and panic reporting. Getting these wrong
shows up as an adapter that hammers a rate-limited API, logs the same failure forever, or silently
serves stale data.

## Requirements

### Requirement: Three-Stage Backoff Delay
`backoff.New(initial, repeated, final, initialFailureCount, repeatedFailureCount)` SHALL produce a
delay that depends only on the consecutive failure count: `initial` while the count is at most
`initialFailureCount`, `final` once the count exceeds `initialFailureCount + repeatedFailureCount`,
and `repeated` in between. `Delay(0)` SHALL therefore return `initial`, and the progression SHALL
never decrease as the count grows.

#### Scenario: default streaming backoff
- **WHEN** a backoff built as `New(10s, 1m, 5m, 1, 3)` is asked for failures 1 through 5
- **THEN** the delays are 10s, 1m, 1m, 1m and 5m

### Requirement: Backoff Should Semantics
`Should(lastFailure, failureCount)` SHALL report whether the caller is still inside the backoff
window, i.e. whether `lastFailure` plus the delay for that failure count is still in the future. A
zero `lastFailure` SHALL always return false, so a caller that has never failed is never held back.

#### Scenario: never failed
- **WHEN** `Should` is called with a zero time
- **THEN** false is returned regardless of the failure count

### Requirement: Stateful Backoff Accounting
`backoff.Stateful` SHALL track the consecutive failure count and the time of the last failure behind
its own mutex, so one instance may be shared across goroutines. `Next` SHALL record a failure and
return the delay for the resulting count — it both advances the streak and answers it. `Fail` SHALL
record a failure without returning a delay, `Should` SHALL evaluate the recorded state, and `Reset`
SHALL clear both the count and the last-failure time so the next delay starts from `initial` again.

#### Scenario: success resets the streak
- **WHEN** `Next` has been called four times and `Reset` is then called
- **THEN** the next `Next` returns the initial delay and `Should` returns false until it is called

### Requirement: Tolerant Fixed Backoff
`backoff.NewTolerantFixed(tolerance, delay)` SHALL apply no delay for the first `tolerance`
consecutive failures, absorbing transient blips, and exactly `delay` for every failure after that.

#### Scenario: blip inside the tolerance
- **WHEN** a tolerant-fixed backoff with tolerance 2 records its first and second failure
- **THEN** both return a zero delay and the third returns the fixed delay

### Requirement: HTTP Error Contract
`httpclient` SHALL define the sentinel errors the rest of the framework keys on: `ErrUnauthorized`
for HTTP 401 and 403, `ErrNotFound` for 404 and `ErrTooManyRequests` for 429. Any status below 300
SHALL map to no error, and any other failing status SHALL map to a generic error naming the status
code. `ErrorFromResponse(resp)` SHALL return a `*TooManyRequestsError` for a 429, carrying the
server-requested `RetryAfter`, and that type SHALL match `ErrTooManyRequests` under `errors.Is` so
callers may test the sentinel without knowing about the concrete type. Adapter clients SHALL return
these errors rather than an empty or truncated result, because the sync flow treats a successful
fetch as a complete device list.

#### Scenario: rate limited with a delay
- **WHEN** a 429 response carries `Retry-After: 120`
- **THEN** `errors.Is(err, ErrTooManyRequests)` holds and the error's `RetryAfter` is 2 minutes

#### Scenario: forbidden treated as unauthorized
- **WHEN** the API responds 403
- **THEN** `ErrUnauthorized` is returned, so the auth-loss handling triggers exactly as for 401

### Requirement: Retry-After Parsing And Cap
`Retry-After` SHALL be accepted in both delta-seconds and HTTP-date form and SHALL be clamped to one
hour, so a misconfigured or hostile server cannot park the client indefinitely. A non-positive or
absent value SHALL yield a zero duration, as SHALL a date already in the past. A numeric value at or
above the cap SHALL clamp before the seconds-to-duration multiplication, so an oversized value cannot
overflow into a small positive delay, and a numeric value too large for `int` SHALL likewise clamp to
the cap rather than being reinterpreted as a date.

#### Scenario: absurd delta-seconds value
- **WHEN** the header is `Retry-After: 99999999999999999999`
- **THEN** the reported delay is one hour, not a wrapped short delay

#### Scenario: date in the past
- **WHEN** the header carries an HTTP-date that has already elapsed
- **THEN** the reported delay is zero

### Requirement: JSON Request Construction
`NewJSONRequest(ctx, method, url, body, headers)` SHALL build a context-bound request, marshalling a
non-nil body to JSON and setting `Content-Type: application/json` only in that case, then applying
the supplied headers last so a caller may override the content type. A body that cannot be marshalled
SHALL return an error and no request.

#### Scenario: bodyless request
- **WHEN** `NewJSONRequest` is called with a nil body
- **THEN** the request has no body reader and no `Content-Type` header is set by the builder

### Requirement: Stream Supervisor Lifecycle
`stream.NewSupervisor(connect, backoff)` SHALL implement `root.Service` and SHALL substitute a
default backoff of `NewStateful(10s, 1m, 5m, 1, 3)` when none is supplied. `Start` SHALL fail when
the supervisor is already running, and SHALL run exactly one connection goroutine. `Stop` SHALL
cancel the supervising context, wait for that goroutine to exit, and only then clear its running
state, so a subsequent `Start` can never overlap two connections; `Stop` on a supervisor that is not
running SHALL return nil. Subscription registries and resubscription remain the caller's
responsibility — the supervisor owns the connection lifecycle only.

#### Scenario: restart after stop
- **WHEN** `Stop` returns and `Start` is called again
- **THEN** the previous connection goroutine has already exited before the new one dials

#### Scenario: double start
- **WHEN** `Start` is called on a running supervisor
- **THEN** an error is returned and no second connection is opened

### Requirement: Connection Contract And Reconnection
The `Connection` function SHALL block until the connection ends or its context is cancelled, and
SHALL invoke the supplied `connected` callback once the connection is established — that callback is
the backoff reset, so a connection that never reports itself established makes every reconnect look
like a fresh failure streak. A returned error SHALL be logged at warn level and followed by the next
backoff delay. A nil error SHALL be treated as a clean shutdown: the supervisor SHALL still wait a
delay before redialling but SHALL reset the backoff, so a stream that closes cleanly in a loop is
paced like a first retry without ever escalating the streak. When the supervising context is
cancelled the supervisor SHALL return without dialling again, both while connected and while waiting
out a delay.

#### Scenario: connection drops repeatedly
- **WHEN** `connect` returns an error three times in a row without ever calling `connected`
- **THEN** the waits follow the backoff progression instead of restarting at the initial delay

#### Scenario: clean close loop
- **WHEN** `connect` returns nil each time after calling `connected`
- **THEN** each redial is preceded by the initial delay and the delay never escalates

### Requirement: Triggered Reconnect
`TriggerReconnect` SHALL drop the current connection, or cut short a pending backoff wait, so a
credentials change takes effect immediately. A triggered drop SHALL NOT count as a failure: the
backoff SHALL be reset and the supervisor SHALL redial immediately, so an attempt that fails right
after the change starts at the initial delay. The reset SHALL also be applied when a trigger races a
firing backoff timer and loses the select, and `TriggerReconnect` on a stopped supervisor SHALL do
nothing.

#### Scenario: token refreshed mid-backoff
- **WHEN** `TriggerReconnect` is called while the supervisor is waiting out a 5 minute delay
- **THEN** the wait ends immediately, the backoff is reset, and the next failure waits the initial
  delay

### Requirement: Repeated Message Throttling
`utils.Throttle.Do(key, emit)` SHALL invoke `emit` only when `key` differs from the key of the
previous call, collapsing a run of identical messages — a revoked token repeated on every retry — into
one emission. The first call SHALL always emit, including when its key is the empty string, because
never-called is tracked separately from a previous empty key. `emit` runs under the throttle's lock
and therefore MUST NOT block or call back into the same `Throttle`. `Reset` SHALL forget the last key
so the next `Do` emits again.

#### Scenario: error clears and returns
- **WHEN** `Do` is called with key A, then key B, then key A again
- **THEN** `emit` runs all three times

#### Scenario: first call with an empty key
- **WHEN** `Do("", emit)` is the first call on a fresh throttle
- **THEN** `emit` runs

### Requirement: Out-Of-Order Value Rejection
`utils.Timestamped[T].Set(value, ts)` SHALL store the value and report true unless a value with a
strictly later timestamp is already held, in which case it SHALL be discarded and false returned —
the case of a streamed observation arriving after a fresher poll. A timestamp equal to the held one
SHALL win, matching the last-write-wins semantics of the source adapters. `Get` SHALL return the held
value, its timestamp, and false when nothing has been stored yet. The lock guards the ordering only:
for a `T` holding references the caller MUST NOT mutate what it passed to `Set` or received from
`Get`.

#### Scenario: late stream event
- **WHEN** a value timestamped one minute before the held one is set
- **THEN** false is returned and the held value is unchanged

#### Scenario: same-timestamp update
- **WHEN** a value carrying exactly the held timestamp is set
- **THEN** it replaces the held value and true is returned

### Requirement: Value And Enum Helpers
`utils.Ptr(v)` SHALL return a pointer to a copy of `v`, for building optional fields inline.
`utils.Normalize(value, allowed)` SHALL match `value` case-insensitively against the allowed
spellings and return the canonical one with true, or the zero value with false when nothing matches,
so vendor-supplied strings can be mapped onto FIMP enum values without accepting unknown ones. The
phase helpers SHALL enumerate the supported phases, phase modes and grid types, and `PhaseMode`
SHALL derive a phase mode from a grid type plus the utilized phases, returning
`types.PhaseModeUnknown` for any combination it does not recognise — TN grids yield the neutral-
prefixed modes, IT and TT grids the phase-to-phase ones.

#### Scenario: vendor spelling normalized
- **WHEN** `Normalize("CHARGING", allowed)` is called and the canonical value is `charging`
- **THEN** `charging` and true are returned

#### Scenario: unknown grid/phase combination
- **WHEN** `PhaseMode` is called for a TN grid with only L1 and L3
- **THEN** `types.PhaseModeUnknown` is returned rather than a guessed mode

### Requirement: Panic And Stack Diagnostics
`utils.PrintStackOnRecover(name, terminate)` SHALL be usable as a deferred recover: it SHALL do
nothing when nothing panicked, and otherwise SHALL log the panic value and the stack at error level,
re-panicking with the original value when `terminate` is true.
`utils.FilterGoroutinesByKeywords(dump, keywords)` SHALL keep only those goroutine blocks of a stack
dump in which at least one line after the `goroutine ` header matches one of the keywords, compared
case-insensitively on word boundaries, and SHALL separate the kept blocks with a blank line while
trimming trailing blank lines.

#### Scenario: filtered stack dump
- **WHEN** a dump of many goroutines is filtered by an adapter-specific keyword
- **THEN** only the goroutine blocks whose frames mention it are returned, each block intact

#### Scenario: keyword only in the header line
- **WHEN** the keyword appears solely in a block's `goroutine ` header line
- **THEN** that block is not kept
