# Thing Reporting Specification

## Purpose
Defines how a thing announces itself to the hub and how it keeps the hub informed about its
connectivity. Inclusion and exclusion reports are the only way a hub node appears or disappears,
and connectivity reports are the only way the hub learns that a device went unreachable. Because
these reports are published on every boot and on every polling cycle, the capability also defines
the deduplication rules — a persisted inclusion checksum and an in-memory reporting cache — that
keep an idle adapter silent on the bus.

## Requirements

### Requirement: Inclusion Report Publication
A thing SHALL publish its inclusion report as an `evt.thing.inclusion_report` object message on the
adapter topic (`PublishAdapterMessage`), not on any per-service topic. After a successful publish
the thing SHALL emit a `ThingEvent` of domain `adapter_thing` and class `inclusion_report_sent`
carrying the report as payload, and SHALL then persist the report checksum.

#### Scenario: Report is published
- **WHEN** `SendInclusionReport` decides a report is due
- **THEN** an `evt.thing.inclusion_report` message whose service is the adapter resource name is published on the adapter topic
- **AND** an `inclusion_report_sent` thing event carrying the report and the thing address is published on the event bus

#### Scenario: Publishing fails
- **WHEN** the MQTT publish returns an error
- **THEN** `SendInclusionReport` returns `false` and the error
- **AND** the stored inclusion checksum is left unchanged, so the next attempt re-sends the same report

### Requirement: Inclusion Report Deduplication By Checksum
A thing SHALL compute the CRC32 (IEEE polynomial) checksum of the JSON-marshalled inclusion report
and compare it with the checksum persisted in `ThingState`. When the checksums are equal and `force`
is false the thing SHALL NOT publish anything and SHALL return `false` with no error. A `force`
value of `true` SHALL bypass the comparison and always publish. The checksum SHALL be persisted in
the thing's record in `adapter.json`, so deduplication survives a restart.

#### Scenario: Unchanged report on boot
- **WHEN** `InitializeThings` calls `SendInclusionReport(false)` for a thing whose report marshals to the stored checksum
- **THEN** no message is published and no thing event is emitted
- **AND** the call reports that nothing was sent

#### Scenario: Forced report
- **WHEN** `SendInclusionReport(true)` is called for a thing whose report has not changed
- **THEN** the report is published anyway and the checksum is re-persisted

#### Scenario: Checksum persistence fails
- **WHEN** the report was published but writing the checksum to the state file fails
- **THEN** `SendInclusionReport` returns an error
- **AND** the in-memory checksum is rolled back to its previous value so a retry writes it again

### Requirement: Inclusion Report Service List Ownership
`NewThing` SHALL discard any `Services` list supplied on the configured inclusion report and rebuild
it from the services passed to the constructor, indexing each service by its topic. `ThingUpdateAddService`
SHALL be a no-op when a service with the same topic is already present, and `ThingUpdateRemoveService`
SHALL remove the service from both the index and the inclusion report's service list. A thing update
SHALL NOT publish a report by itself; the caller decides when to re-announce.

#### Scenario: Service added at runtime
- **WHEN** a thing is updated with `ThingUpdateAddService` for a topic it does not yet serve
- **THEN** the service becomes reachable through `ServiceByTopic` and its specification is appended to the inclusion report
- **AND** the change reaches the hub only once `SendInclusionReport` is called, whose checksum now differs

### Requirement: Service Lookup By Inbound Topic
A thing SHALL resolve an inbound message topic to a service by cutting everything before the
`/rt:dev/rn:` marker and looking the remainder up in its service index, since a service address
always begins with that prefix while an inbound topic carries a message-type prefix in front of it.
A topic without the marker SHALL simply miss.

#### Scenario: Command topic resolves to a service
- **WHEN** a message arrives on `pt:j1/mt:cmd/rt:dev/rn:<adapter>/ad:1/sv:out_bin_switch/ad:1`
- **THEN** `ServiceByTopic` returns the service registered under `/rt:dev/rn:<adapter>/ad:1/sv:out_bin_switch/ad:1`

### Requirement: Exclusion Report On Destroy
The adapter SHALL publish an `evt.thing.exclusion_report` object message carrying only the thing
address on the adapter topic whenever a thing is destroyed. Destroying SHALL be best effort across
its three steps — removing the persisted record, unregistering and disconnecting the live thing, and
publishing the exclusion — with the errors joined, so a failed state write never skips the
disconnect. An address with no registered thing SHALL still produce an exclusion report, which
clears a stale hub node the state never knew about.

#### Scenario: Unknown address is excluded
- **WHEN** `DestroyThingByAddress` is called with an address the adapter has no state record for
- **THEN** an `evt.thing.exclusion_report` for that address is still published

#### Scenario: State write fails during destroy
- **WHEN** removing the thing's state record fails
- **THEN** the thing is still unregistered and disconnected and the exclusion report is still attempted
- **AND** the returned error joins every step that failed

### Requirement: Connectivity Report Composition
A thing's connectivity report SHALL combine identity fields taken from its inclusion report —
address, `ProductHash`, `ProductName`, power source, wake-up interval and communication technology —
with the `ConnectivityDetails` returned by its `Connector`. The report SHALL be sanitized before it
leaves the thing: an empty `ConnQuality` becomes `undefined`, an empty `ConnType` becomes `unknown`,
and a nil operationability list becomes an empty array rather than JSON `null`.

#### Scenario: Connector returns bare details
- **WHEN** a connector reports only `ConnStatus` `UP`
- **THEN** the published report carries `conn_quality: "undefined"`, `conn_type: "unknown"` and `operationability: []`

### Requirement: Connectivity Vocabulary
A connector SHALL describe connectivity using only the defined vocabularies: `ConnStatusT` is `UP`
or `DOWN`; `ConnTypeT` is `direct`, `indirect` or `unknown`; `ConnQualityT` is one of `high`,
`medium`, `low`, `undefined`, `very_strong`, `strong`, `good`, `ok`, `poor`, `very_poor` or
`no_signal`; `OperationabilityT` is one of `sleep`, `discovery`, `broken`, `not_ready`, `ready`,
`removed`, `left`, `update` or `failed`. A ping SHALL report `PingResult` `SUCCESS` or `FAILED`.

#### Scenario: Connectivity drives task skipping
- **WHEN** a thing's connectivity report carries `ConnStatus` `DOWN`
- **THEN** `ShouldSkipServiceTask` returns true for every service of that thing

### Requirement: Connectivity Report Publication And Deduplication
`SendConnectivityReport` SHALL publish an `evt.network.node_report` object message on the adapter
topic, gated by the thing's connectivity reporting strategy through a `ReportingCache` keyed on
`evt.network.node_report` with an empty sub key. When `force` is false and the strategy declines,
the thing SHALL publish nothing and return `false`. When `ThingConfig.ConnectivityReportingStrategy`
is nil, `NewThing` SHALL default it to `cache.ReportAtLeastEvery(time.Hour)`.

#### Scenario: Unchanged connectivity within the default window
- **WHEN** the connectivity-reporting task calls `SendConnectivityReport(false)` on a thing whose details are unchanged and whose last node report was sent less than an hour ago
- **THEN** no `evt.network.node_report` is published

#### Scenario: Connectivity changed
- **WHEN** the connector's details differ from the last reported ones
- **THEN** an `evt.network.node_report` is published and the cache records the new value as reported

### Requirement: Connectivity Event Is Always Emitted
A thing SHALL publish a `ConnectivityEvent` on every `SendConnectivityReport` call, before the
reporting strategy is consulted. The event SHALL be a `ThingEvent` of domain `adapter_thing` and
class `thing` with a nil payload, carrying the thing address and the current `ConnectivityDetails`
as fields of the `ConnectivityEvent` itself. The event SHALL therefore
be emitted even when the FIMP node report is suppressed by deduplication, and even when the publish
subsequently fails.

#### Scenario: Deduplicated report still notifies listeners
- **WHEN** `SendConnectivityReport(false)` is suppressed because connectivity has not changed
- **THEN** a `ConnectivityEvent` with the current details is still published on the event bus

### Requirement: Reporting Strategies
The reporting strategies SHALL behave as follows: `ReportAlways` always requires a report;
`ReportOnChangeOnly` requires one only when the value changed; `ReportAtLeastEvery(interval)`
requires one when the value changed or when strictly more than `interval` has elapsed since the last
report. `ReportingCache` SHALL compare values with `reflect.DeepEqual` and SHALL require a report
for any key/sub-key combination it has never recorded, so the first report after start-up is always
sent regardless of strategy.

#### Scenario: First report after start-up
- **WHEN** a value is offered for a key the reporting cache holds no entry for
- **THEN** `ReportRequired` returns true without consulting the strategy

#### Scenario: Periodic re-report
- **WHEN** a value is unchanged but its last report is older than the strategy's interval
- **THEN** `ReportAtLeastEvery` requires a report and `ReportOnChangeOnly` does not

### Requirement: Connector Contract
Every thing SHALL be built with a `Connector` supplying `Connectivity()` and `Ping()`. `Ping()` SHALL
never return cached data and SHALL always perform a real round trip. `Connect` and `Disconnect` SHALL
only take effect when the connector also satisfies `ControllableConnector`; otherwise the thing's
`Connect`/`Disconnect` are silent no-ops, which is the expected shape for a purely polling adapter.
Both SHALL be idempotent.

#### Scenario: Thing registered
- **WHEN** the adapter registers a thing
- **THEN** `Connect` is called on it
- **AND** any previously registered instance under the same address is disconnected first, so its connector does not leak

#### Scenario: Thing unregistered
- **WHEN** the adapter unregisters a thing while destroying it
- **THEN** `Disconnect` is called on it

### Requirement: Ping Report
`SendPingReport` SHALL invoke the connector's `Ping()`, measure the elapsed time, and publish an
`evt.ping.report` object message on the adapter topic containing the thing address, the measured
delay in whole milliseconds, the ping status and, optionally, the connection nodes. Ping reports
SHALL NOT be deduplicated.

#### Scenario: Ping executed
- **WHEN** `SendPingReport` is called twice with identical ping details
- **THEN** two `evt.ping.report` messages are published, each carrying its own measured delay

### Requirement: Refresher Caching And Backoff
`cache.Refresher` SHALL return the cached value without invoking the refresh function while less
than its configured interval has elapsed since the last successful refresh. On a failure it SHALL
increment a failure count, record the failure time and return an error without caching a value.
While backoff is in effect it SHALL fail fast with a backoff error instead of calling the refresh
function; backoff SHALL be disabled entirely when the backoff threshold is zero. `IsFailing` SHALL
report true only once the failure count is strictly greater than the failure threshold. A successful
refresh SHALL reset the failure count and clear the backoff; `Reset` SHALL clear the cached value so
the next call refreshes.

#### Scenario: Within the interval
- **WHEN** `Refresh` is called twice in quick succession on a refresher with a one-minute interval
- **THEN** the refresh function runs once and the second call returns the cached value

#### Scenario: Default backoff tiers
- **WHEN** a refresher configured with `WithDefaultOptions` fails
- **THEN** the next calls fail fast for 15s while the failure count is at most 3, for 1m while it is at most 6, and for 5m beyond that
- **AND** `IsFailing` becomes true once the failure count exceeds 2

#### Scenario: Interval offset
- **WHEN** a refresher is created with `WithDefaultIntervalOffset`
- **THEN** its effective interval is 95% of the requested one, the default offset being 0.05
