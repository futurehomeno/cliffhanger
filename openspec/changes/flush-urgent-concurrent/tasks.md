## 1. Fix

- [x] 1.1 Failing test: an error line logged while other goroutines log info reaches the file before `logrus.Error` returns
- [x] 1.2 Mark urgent lines by their formatted bytes instead of a global flag
- [x] 1.3 Ration the test's noise per error line, so no schedule can push a flushed line out of the searched tail
