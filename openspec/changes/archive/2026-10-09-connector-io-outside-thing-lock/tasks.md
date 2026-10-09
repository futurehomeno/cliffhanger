## 1. Connector calls outside the thing lock
- [x] 1.1 Test: `Update` completes while `Connectivity()` or `Ping()` is blocked (fails before the fix)
- [x] 1.2 `ConnectivityReport` and `SendPingReport` call the connector without the lock
- [x] 1.3 Test: a second connector call waits for an in-flight connectivity report (fails without the fix)
- [x] 1.4 Connector calls and the reporting cache share a per-thing connector lock, so reports stay ordered
