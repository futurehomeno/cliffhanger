# Telemetry Specification

## Purpose
Defines how an adapter reports rare, cross-app operational events — panics, auth loss, invariant
violations, reboot milestones — to the Futurehome cloud over FIMP, and how the cloud controls that
reporting remotely. Telemetry is off-by-default in effect: it is bounded by a validity window that
expires on its own, gated by cloud-pushed suppression rules, and every emit helper is heavily
throttled so a device stuck in a failure loop cannot flood the pipeline.

## Requirements

### Requirement: Telemetry Construction
`telemetry.New(mqtt, sourceRn, store, version)` SHALL reject a nil MQTT transport, an empty source
resource name and a nil store. When the store holds no telemetry configuration it SHALL seed one
with `Enabled: true` and `EnabledAt` set to now, so a freshly installed adapter starts inside a
validity window rather than silently unreported. Construction SHALL resume the validity window and
prepare — but not start — the cloud configuration poll; the poll's goroutine, timer and MQTT
subscription are tied to the application lifecycle via `Start`/`Stop`, not to process lifetime.

#### Scenario: fresh install
- **WHEN** `New` runs against a store with no telemetry block
- **THEN** a config with `Enabled: true` and `EnabledAt: now` is persisted before the constructor
  returns

#### Scenario: stop then start in one process
- **WHEN** `Stop` is followed by `Start`
- **THEN** the validity window timer is re-armed from the persisted `EnabledAt`, so telemetry does
  not stay enabled past its window until a cloud config report happens to arrive

### Requirement: Emit Helpers Never Fail The Caller
`Emit`, `EmitOnChange`, `EmitIfMore` and `ResetEventCounters` SHALL be no-ops when the `Telemetry`
value is nil, so call sites need no guard, and SHALL log any failure at warn level instead of
returning it. Telemetry SHALL never propagate an error into adapter logic.

#### Scenario: telemetry not wired
- **WHEN** `Emit(nil, DomainAuth, EventLoggedOut, nil)` is called
- **THEN** nothing is published and no panic occurs

#### Scenario: publish fails
- **WHEN** the MQTT publish returns an error
- **THEN** a warn line naming the event is logged and the caller continues unaffected

### Requirement: Telemetry Event Publication
A published event SHALL be an object message with interface `evt.telemetry.report`, service name and
source both set to the adapter's resource name, and a body of `{event, domain, data}`. The default
destination topic SHALL be `pt:j1/mt:rsp/rt:cloud/rn:backend-service/ad:telemetry`, overridable via
`SetEvtTopic`, where an empty topic SHALL restore that default. When a non-empty `version` was
supplied to `New`, it SHALL be copied into the event `data` under the key `version` without mutating
the caller's map. An empty event name SHALL be rejected as an error.

#### Scenario: version injection
- **WHEN** an adapter built with version `1.4.2` emits an event with `data` `{"count": 3}`
- **THEN** the published payload carries `{"count": 3, "version": "1.4.2"}` and the caller's map is
  left with only `count`

### Requirement: Enablement And Suppression Gating
`emit` SHALL publish nothing while the persisted config is not enabled. When a `SuppressedEntry` is
present, an entry whose `Domains` and `Events` are both empty SHALL suppress every event for the
app, and otherwise an event SHALL be dropped when its domain appears in `Domains` **or** its event
name appears in `Events`. A nil `Suppressed` SHALL mean no suppression. Every configuration snapshot
read for gating SHALL be deep-copied through `types.TelemetryConfig.Clone`, which copies the
`Suppressed` entry and its `Domains` and `Events` slices, so no caller can reach the stored
configuration through a returned entry. `SetSuppressed` SHALL read
only the entry keyed by the adapter's own resource name: an absent key clears suppression, a present
but empty entry suppresses everything, and a populated entry replaces the stored lists wholesale.

#### Scenario: app-wide suppression pushed by the cloud
- **WHEN** the cloud sends a suppression map whose entry for this resource name has no domains and
  no events
- **THEN** every subsequent `Emit` is dropped before publishing

#### Scenario: this app not mentioned
- **WHEN** the suppression map contains entries only for other resource names
- **THEN** stored suppression is cleared and all events publish again

### Requirement: Change-Throttled Emission
`EmitOnChange(tel, domain, event, data, interval)` SHALL publish at most once per `interval` per
`domain/event` pair, regardless of the payload. The first call for a pair SHALL always be forwarded
to `emit`, and the throttle timestamp SHALL be advanced only on calls that are not throttled.

#### Scenario: repeated within the interval
- **WHEN** the same domain/event is emitted three times inside one interval
- **THEN** only the first reaches `emit`

### Requirement: Threshold-Counted Emission
`EmitIfMore(tel, domain, event, threshold, reset, data, interval)` SHALL count calls per
`domain`/`event`/data-fingerprint triple and SHALL forward to `emit` only once the count has reached
`threshold` and at least `interval` has passed since the last emission for that triple. A
`threshold` below 1 SHALL be a no-op. Counters SHALL be bucketed by the JSON fingerprint of `data`,
with the parts separated by a NUL byte so a sibling event whose name shares this one's prefix cannot
collide with it. When `reset` is true the counter SHALL be zeroed on emission, making the threshold
recur; when false the counter keeps climbing and only `interval` paces further emissions.

#### Scenario: per-device counting
- **WHEN** the same event is emitted with `{"device_id": 1}` and `{"device_id": 2}`
- **THEN** each device accumulates its own count against the threshold

#### Scenario: threshold reached inside the interval of a previous emission
- **WHEN** the count reaches the threshold again less than `interval` after the last emission
- **THEN** nothing is published and the counter is not reset

### Requirement: Scoped Counter Reset
`ResetEventCounters(tel, domain, event, scope)` SHALL delete the `EmitIfMore` counters of that
domain/event whose stored data contains every key/value pair of `scope`, compared with the same JSON
semantics used to bucket the counters so the scope value need not match the stored value's exact Go
type. A nil or empty scope SHALL clear every counter for the event, so a success can reset just its
own subject without disturbing other devices.

#### Scenario: one device recovers
- **WHEN** `ResetEventCounters` is called with scope `{"device_id": 7}`
- **THEN** only counters whose data contains `device_id: 7` are removed and other devices' counts
  survive

### Requirement: Validity Window
Telemetry SHALL be enabled only for a bounded window, defaulting to `30 * 24h` when no positive
`Validity` is configured. `Enable(true)` SHALL stamp `EnabledAt` with now, persist, and arm a timer
for the full validity; `Enable(false)` SHALL clear `EnabledAt` and stop the timer. On expiry the
timer SHALL disable telemetry and persist that, and a timer that is no longer the current one SHALL
do nothing. On construction and on `Start`, the window SHALL be resumed from the persisted
`EnabledAt`: a zero or future `EnabledAt` SHALL be reset to now, and an already-elapsed window SHALL
disable telemetry and persist it before any event can be emitted. `SetValidity` SHALL reject a
non-positive duration and SHALL re-arm the timer for the remaining time, disabling immediately when
the new validity is already exceeded by the elapsed time.

#### Scenario: hub was powered off past the window
- **WHEN** the adapter starts with `EnabledAt` older than the validity
- **THEN** telemetry is persisted as disabled with a cleared `EnabledAt` and no event is published

#### Scenario: validity shortened below elapsed time
- **WHEN** `SetValidity(1h)` is called 3 hours after telemetry was enabled
- **THEN** telemetry is disabled and the new validity is persisted

### Requirement: Predefined Domains And Milestone Sampling
The framework SHALL define the shared domains `panic`, `auth`, `should_never_happen` and `reboot`,
plus the event names `logged_out` and `milestone`, so the cloud can group and suppress by stable
keys. `EmitRebootMilestone(tel, count)` SHALL emit `reboot`/`milestone` with `{"count": count}` only
when `count` is positive and an exact multiple of 500, so callers may invoke it on every boot while a
crash-looping device still contributes at most one event per 500 restarts.

#### Scenario: ordinary boot
- **WHEN** `EmitRebootMilestone(tel, 501)` is called
- **THEN** nothing is emitted

#### Scenario: milestone boot
- **WHEN** `EmitRebootMilestone(tel, 1000)` is called
- **THEN** a `reboot`/`milestone` event carrying `count: 1000` is emitted

### Requirement: Panic Reporting
`RecoverAndEmit(tel, name, terminate)` SHALL be usable as a deferred recover: when nothing panicked
it SHALL do nothing. On a recovered panic it SHALL log the value and stack at error level and emit a
`panic`-domain event whose event name is `name` and whose data carries `terminate`. When `terminate`
is true it SHALL re-panic with the original value after emitting, so a fatal panic is reported and
still crashes the process.

#### Scenario: non-fatal recovery
- **WHEN** a goroutine guarded by `RecoverAndEmit(tel, "poller", false)` panics
- **THEN** the panic is logged, a `panic`/`poller` event is emitted, and the goroutine returns
  normally

### Requirement: Telemetry Settings Routing
`telemetry.Route(tel, options...)` SHALL return no routings for a nil telemetry, and otherwise SHALL
expose get and set routings for the settings `telemetry_enabled` (bool), `telemetry_validity`
(duration string) and `telemetry_suppressed` (object), all on the adapter's own service name using
the `cmd.config.get_<setting>` / `cmd.config.set_<setting>` commands and answering with
`evt.config.<setting>_report`. A validity string that `time.ParseDuration` rejects SHALL produce an
error reply, and each successful set SHALL publish a configuration-change event through the supplied
routing options.

#### Scenario: unparsable validity
- **WHEN** `cmd.config.set_telemetry_validity` carries `"forever"`
- **THEN** an error naming the value is returned and the stored validity is unchanged

#### Scenario: suppression set from the bus
- **WHEN** `cmd.config.set_telemetry_suppressed` carries a map of per-service entries
- **THEN** only this service's entry is applied and the reply reports the resulting suppression

### Requirement: Cloud Configuration Poll
The configuration poller SHALL request `cmd.telemetry.get_config` on
`pt:j1/mt:cmd/rt:cloud/rn:telemetry/ad:config` and SHALL listen for `evt.telemetry.config_report` on
the shared broadcast topic `pt:j1/mt:evt/rt:cloud/rn:telemetry/ad:config`, which every app subscribes
to permanently so one app's poll response configures all of them. The first subscribe attempt SHALL
be made 3 seconds after start and retried with escalating backoff (1 minute, then 30 minutes) until
it succeeds or the poller stops. Messages from
any other topic or interface SHALL be ignored. Each scheduled poll SHALL be skipped — and merely
rescheduled — when a config report was received less than 30 minutes ago, because another app
already polled. After sending a request the poller SHALL always schedule a 6 hour fallback retry, so
polling never stops when a response is lost; a received report reschedules sooner. The next delay
SHALL be taken from the report's `next_update` timestamp when it parses and lies in the future,
capped at 24 hours, and SHALL fall back to 6 hours otherwise; a random jitter of up to 30 minutes
SHALL be added to spread the timers of apps sharing the broadcast topic. A received report SHALL be
applied by enabling or disabling telemetry and replacing the suppression rules.

#### Scenario: another app already polled
- **WHEN** a scheduled poll fires within 30 minutes of the last received config report
- **THEN** no request is published and the poll is rescheduled

#### Scenario: next_update in the past
- **WHEN** the report's `next_update` is unparsable or already elapsed
- **THEN** the next poll is scheduled 6 hours plus jitter later

### Requirement: Deliberate Cold-Start Poll Delay
The first configuration poll after `Start` SHALL be scheduled a full `DefaultPollInterval` — 6 hours
— later, with no jitter and no immediate poll at startup. This delay is intentional cold-start
backoff against the cloud: hubs reboot in waves (firmware rollouts, power cuts, network restoration)
and every app on every hub shares one request topic, so polling at startup would turn each wave into
a synchronised burst against the telemetry backend. Telemetry keeps operating on its persisted
configuration in the meantime, so nothing is lost by waiting. This delay SHALL NOT be shortened or
replaced by an immediate first poll.

#### Scenario: adapter starts
- **WHEN** `Start` is called on the configuration poller
- **THEN** no `cmd.telemetry.get_config` is published for 6 hours, while the persisted enablement and
  suppression continue to gate emission

#### Scenario: cloud pushes config before the first poll
- **WHEN** another app's poll causes a broadcast `evt.telemetry.config_report` during those 6 hours
- **THEN** this app applies it too and reschedules its own poll from that report
