## Why
`thing.ConnectivityReport` held the thing's write lock while calling `Connector.Connectivity()`,
and `SendPingReport` held the read lock across `Connector.Ping()`. Both connectors usually wait on
a device or a cloud (the Hue adapter makes bridge HTTP calls of up to ~12 s there), so every other
caller on the thing - the periodic connectivity task, a stream-driven report, `Update`, a
concurrent ping - queued behind that I/O. A shutdown landing in that window prints the queued
goroutine in the SIGTERM goroutine dump, which reads like a deadlock.

## What Changes
- `ConnectivityReport` calls the connector without the thing lock.
- `SendPingReport` calls the connector and publishes without the lock.
- A per-thing connector lock keeps connector calls one at a time and covers
  `SendConnectivityReport` from reading to publishing, so reports stay in order and connectors
  written for the old locking are never called concurrently. `SendConnectivityReport` no longer
  takes the thing lock at all.

## Impact
- Specs: `adapter/reporting` gains a requirement that connector calls run outside the thing lock.
- Code: `adapter/thing.go`. No API change.
