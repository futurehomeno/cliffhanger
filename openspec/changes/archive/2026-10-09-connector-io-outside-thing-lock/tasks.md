## 1. Connector calls outside the thing lock
- [x] 1.1 Test: `Update` completes while `Connectivity()` or `Ping()` is blocked (fails before the fix)
- [x] 1.2 `ConnectivityReport` and `SendPingReport` call the connector without the lock
