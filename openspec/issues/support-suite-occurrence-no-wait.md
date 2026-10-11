# test/suite ExactlyOnce/AtMostOnce do not catch a late duplicate

`test/suite/router.go:83-110`: `AssertExpectations` returns as soon as every expectation is
satisfied and only waits the full timeout when some expectation is `Never`. `AtMostOnce` is
satisfied by zero calls, so a node with only `AtMostOnce` expectations returns immediately, and an
`ExactlyOnce` expectation passes at the first match — a duplicate arriving a few ms later is
attributed to the next node (or lost) instead of failing this one. Adapter authors reading the
names reasonably expect "no more than once within the timeout". Decide whether the upper-bounded
occurrences should also wait out the timeout (as `Never` does) and spec it either way.
