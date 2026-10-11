# chargepoint task never polls the phase mode of an aware-only charger

A controller implementing `PhaseModeAwareController` but not `AdjustablePhaseModeController` gets
`cmd.phase_mode.get_report` / `evt.phase_mode.report` advertised (`chargepoint/service.go`, `IsPhaseModeAware`
branch of `NewService`), yet the periodic task only sends the phase mode report when
`SupportsAdjustingPhaseModes()` is true (`adapter/service/chargepoint/tasks.go:55`). Cable lock, by
contrast, is polled on awareness (`IsCableLockAware`). An aware-only charger's phase mode therefore
reaches the hub only on an explicit get_report.

Close by gating the task's phase mode report on `IsPhaseModeAware()` (adjustable implies aware), with
a task test for an aware-only controller — or document why phase mode is poll-on-demand only.
