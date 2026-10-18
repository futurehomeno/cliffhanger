## 1. Fix

- [x] 1.1 Failing test: an error line logged while other goroutines log info reaches the file before `logrus.Error` returns
- [x] 1.2 Mark urgent lines by their formatted bytes instead of a global flag
