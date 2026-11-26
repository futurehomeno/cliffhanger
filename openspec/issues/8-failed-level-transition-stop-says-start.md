# A failed stop of a level transition says "failed to start"

`adapter/service/outlvlswitch/service.go`, `StopLevelTransition`: a controller failure is wrapped as
`failed to start level transition`. The text is copied from `StartLevelTransition`, so the log of a
failed `cmd.lvl.stop` names the wrong command. Found while coding the validation errors
(`error-report-policy`); not changed there, no test covers the text.

Reproduce: a `LevelTransitionController` whose `StopLevelTransition` returns an error; the error
returned by the service reads `failed to start level transition: …`.

Closes when: the wrap reads `failed to stop level transition`, with a test on the text.
