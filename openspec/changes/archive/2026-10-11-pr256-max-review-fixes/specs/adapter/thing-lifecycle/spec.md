## ADDED Requirements

### Requirement: Stale Sweep Claims Only Wholly Owned Devices
The stale sweep SHALL treat a hub device as this adapter's only when every service whose topic names
a resource name names this adapter, and the instance, when present, is this adapter's.

#### Scenario: a device mixing services of two adapters
- **WHEN** a hub device has one service on this adapter's resource name and another on a different one
- **THEN** the sweep does not exclude it
