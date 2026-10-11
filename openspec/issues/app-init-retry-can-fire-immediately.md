# Initialization retry can run back-to-back with the failed startup run

`app/task.go:43-46` builds a one-shot task (interval 0) and a periodic retry task gated by
`WhenAppEncounteredStartupError`. `task.Manager` starts both goroutines together and
`runContinuously` (`task/manager.go:84-91`) runs the periodic task once immediately, before the
first tick. When `Initialize` fails fast (e.g. no credentials, DNS error), the one-shot run can set
`STARTUP_ERROR` before the periodic task's immediate vote, so `Initialize` runs a second time
right away instead of 10 minutes later as the `app-service` "Lifecycle Capability Interfaces"
scenario says. It's harmless for idempotent initializers but doubles vendor calls/log noise at
boot and makes the spec scenario timing-dependent.

Close it by having the retry task skip its immediate first run (or by running the startup attempt
as the retry task's first execution only), then pin it with a test that counts calls in the first
few ms after `Start`.
