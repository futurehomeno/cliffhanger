## ADDED Requirements

### Requirement: Forced Node Report Restarts The Deduplication Window
`SendConnectivityReport(true)` SHALL record the published report in the thing's reporting cache
exactly like an unforced report, so after a `cmd.network.get_node` the connectivity-reporting task
stays silent until the connectivity changes or the strategy's interval has elapsed since the forced
report. A failed publish, forced or not, SHALL leave the reporting cache unchanged.

#### Scenario: periodic report after a forced one
- **WHEN** `SendConnectivityReport(true)` publishes a node report and `SendConnectivityReport(false)`
  follows with unchanged connectivity under `ReportOnChangeOnly`
- **THEN** the second call publishes nothing and returns `false`

#### Scenario: publish fails
- **WHEN** the first node report of a thing fails to publish
- **THEN** an error is returned and the next unforced call publishes, the cache holding no entry yet

### Requirement: Refresher Serialises Refreshes
`cache.Refresher.Refresh` SHALL hold the refresher's lock for the whole call, including the refresh
function, so concurrent callers never run the refresh function in parallel: a caller arriving while a
refresh is in flight waits for it and then receives the freshly cached value; when that refresh
failed, a refresher with backoff enabled answers the waiting callers with the backoff error.

#### Scenario: concurrent callers on an expired cache
- **WHEN** several goroutines call `Refresh` at once on a refresher whose cached value has expired
- **THEN** the refresh function runs once and every caller receives its result

## MODIFIED Requirements

### Requirement: Connectivity Report Composition
A thing's connectivity report SHALL combine identity fields taken from its inclusion report —
address, `ProductHash`, `ProductName`, power source, wake-up interval and communication technology —
with the `ConnectivityDetails` returned by its `Connector`. The report SHALL be sanitized before it
leaves the thing: nil `ConnectivityDetails` from the connector are replaced by empty details, an
empty `ConnQuality` becomes `undefined`, an empty `ConnType` becomes `unknown`, and a nil
operationability list becomes an empty array rather than JSON `null`.

#### Scenario: Connector returns bare details
- **WHEN** a connector reports only `ConnStatus` `UP`
- **THEN** the published report carries `conn_quality: "undefined"`, `conn_type: "unknown"` and `operationability: []`

#### Scenario: Connector returns nil details
- **WHEN** a connector's `Connectivity()` returns nil
- **THEN** the report is still built without panicking, carrying `conn_quality: "undefined"`,
  `conn_type: "unknown"` and `operationability: []`

### Requirement: Refresher Caching And Backoff
`cache.Refresher` SHALL return the cached value without invoking the refresh function while less
than its configured interval has elapsed since the last successful refresh. On a failure it SHALL
increment a failure count, record the failure time and return an error without caching a value.
While backoff is in effect it SHALL fail fast with a backoff error instead of calling the refresh
function; that error SHALL match `cache.ErrBackoff` with `errors.Is` and SHALL wrap the last refresh
error. Backoff SHALL be disabled entirely when the backoff threshold is zero, which is the case
unless a backoff option is given. `IsFailing` SHALL report true only once the failure count is
strictly greater than the failure threshold. A successful refresh SHALL reset the failure count and
clear the backoff. `Reset` SHALL clear the cached value so the next call outside a backoff refreshes;
it SHALL NOT clear the failure count or the backoff, so a call landing inside a backoff still fails
fast.

#### Scenario: Within the interval
- **WHEN** `Refresh` is called twice in quick succession on a refresher with a one-minute interval
- **THEN** the refresh function runs once and the second call returns the cached value

#### Scenario: Default backoff tiers
- **WHEN** a refresher configured with `WithDefaultOptions` fails
- **THEN** the next calls fail fast for 15s while the failure count is at most 3, for 1m while it is at most 6, and for 5m beyond that
- **AND** `IsFailing` becomes true once the failure count exceeds 2

#### Scenario: Backoff error carries the cause
- **WHEN** a refresh fails with an error and the next call lands inside the backoff
- **THEN** the returned error matches both `cache.ErrBackoff` and the refresh error with `errors.Is`

#### Scenario: Interval offset
- **WHEN** a refresher is created with `WithDefaultIntervalOffset`
- **THEN** its effective interval is 95% of the requested one, the default offset being 0.05

#### Scenario: Reset during a backoff
- **WHEN** `Reset` is called while a backoff from a failed refresh is in effect
- **THEN** the next `Refresh` returns the backoff error without calling the refresh function
