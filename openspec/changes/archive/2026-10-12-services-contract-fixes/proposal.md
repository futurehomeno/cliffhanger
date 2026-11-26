# Service contract fixes

## Why

Six open issues in `openspec/issues/`: services advertise, poll or cache something other than what they do.

## What Changes

- `battery` advertises `evt.lvl.report` as `int`, the type it publishes.
- `colorctrl` advertises `cmd.color.set` as `int_map`, the payload its handler requires.
- `colorctrl.Service.SupportedDurations()` returns the stored `map[string]int` (as `[]int` it was always nil). Minor API change.
- The `chargepoint` reporting task sends the phase mode report when the charger is phase-mode aware, not only when adjustable.
- `parameters.SendParameterReport` returns a "parameter not found" error instead of panicking when the controller returns a nil parameter.
- `thermostat` and `waterheater` deduplicate the setpoint report on the whole setpoint (value and unit).

## Impact

`adapter/service/{battery,colorctrl,chargepoint,parameters,thermostat,waterheater}`, the colorctrl `Service` mock.
