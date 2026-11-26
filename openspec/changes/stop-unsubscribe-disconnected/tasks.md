## 1. Fix

- [x] 1.1 Failing test: a stop with a disconnected transport returns no error and attempts no unsubscribe
- [x] 1.2 Test: an unsubscribe failure on a connected transport is still returned
- [x] 1.3 Skip the unsubscriptions in `doStop` when the transport is not connected
