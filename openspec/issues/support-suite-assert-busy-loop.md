# test/suite Router.AssertExpectations busy-spins a CPU core

`test/suite/router.go:89-110` waits with `select { case <-t.C: ... default: ... }` and no sleep, so
every assertion burns a full core until it returns — for the entire timeout when any expectation is
`Never` (`waitUntilTimeout` hits `continue` immediately). With `-p 1` suites and the MQTT client and
router goroutines competing for CPU on small CI runners this slows message delivery and can make
timing-sensitive cases flaky. Close by polling with a short ticker (e.g. 5 ms) or by signalling
from `processMessage` on every match.
