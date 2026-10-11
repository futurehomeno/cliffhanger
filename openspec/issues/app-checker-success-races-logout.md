# ConnectivityChecker success path can undo a logout or reset landing after the Authorized gate

`app/check.go:174-180` — after a successful probe `check()` passes `stale()`, calls `settle()`,
then `authorized()` and finally `apply(AUTHENTICATED, CONNECTED)` + `repairAppHealth()`. `Cancel()`
takes only `c.mu`, not `checkMu`, so a `Logout`/`Reset` (clear credentials → `Cancel` →
`MarkNotConfigured`) that runs between the gate and the restore is overwritten: the app ends
`RUNNING`/`CONFIGURED`/`AUTHENTICATED` with no credentials. The `CheckerConfig.Authorized` doc
comment acknowledges the window. It matters because the hub then shows a logged-in adapter that
fails every call until the next probe reports unauthorized.

Close it by re-checking `stale()` under `c.mu` while applying (e.g. hold `c.mu` across the
cancelled read and the lifecycle writes, or make `Cancel` take `checkMu` after the in-flight probe),
plus a test that cancels from inside a stubbed `Authorized` callback.
