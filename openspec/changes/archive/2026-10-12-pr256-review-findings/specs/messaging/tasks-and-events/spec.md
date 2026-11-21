## MODIFIED Requirements

### Requirement: Subscription Registry Semantics
`Subscribe(subID, buffer, filters...)` SHALL create and return a fresh channel with the requested
buffer for each call, even when the ID is already in use, and each such subscription SHALL keep its
own filters. `Unsubscribe(subID)` SHALL close every channel registered under that ID and remove them,
so an ID is only safe to share between subscribers with the same lifetime; unsubscribing an unknown
or already removed ID SHALL be a no-op. The filter slice SHALL be cloned on subscribe so that later
mutation of the caller's slice cannot alter a live subscription. A nil filter, including a nil
`FilterFn`, SHALL be ignored rather than invoked.

#### Scenario: two subscribers share an ID
- **WHEN** two components subscribe under the same ID and one of them unsubscribes
- **THEN** both channels are closed and both subscriptions are removed

#### Scenario: subscriber-specific filters
- **WHEN** two subscriptions under the same ID declare different filters
- **THEN** each receives only the values matching its own filters

#### Scenario: nil FilterFn
- **WHEN** a subscriber passes a nil `FilterFn`
- **THEN** it is ignored and publishing does not panic
