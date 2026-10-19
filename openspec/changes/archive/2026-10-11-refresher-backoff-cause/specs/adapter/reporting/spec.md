## MODIFIED Requirements

### Requirement: Refresher Caching And Backoff
`cache.Refresher` SHALL return the cached value without invoking the refresh function while less
than its configured interval has elapsed since the last successful refresh. On a failure it SHALL
increment a failure count, record the failure time and return an error without caching a value.
While backoff is in effect it SHALL fail fast with a backoff error instead of calling the refresh
function; that error SHALL match `cache.ErrBackoff` with `errors.Is` and SHALL wrap the last refresh
error. Backoff SHALL be disabled entirely when the backoff threshold is zero. `IsFailing` SHALL
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

#### Scenario: Backoff error carries the cause
- **WHEN** a refresh fails with an error and the next call lands inside the backoff
- **THEN** the returned error matches both `cache.ErrBackoff` and the refresh error with `errors.Is`

#### Scenario: Interval offset
- **WHEN** a refresher is created with `WithDefaultIntervalOffset`
- **THEN** its effective interval is 95% of the requested one, the default offset being 0.05
