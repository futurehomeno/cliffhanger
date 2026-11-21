# The virtual meter's level poll skips things that are down

## Why

`virtualmeter.Tasks` polls the level of every `out_lvl_switch` without the connectivity skip the
other reporting tasks apply, so it hits the controller of unreachable devices at every poll. On hue
3.0.5 beta hubs this poll produced 2,274 `E [cliff] Send level report` lines on 14 of 42 hubs for lights
behind a dead bridge; edge-hue-adapter PR #84 works around it with a filtering registry.

## What Changes

- The level polling task is `outlvlswitch.TaskReporting`, which skips a level switch whose thing is
  down (`adapter.ShouldSkipServiceTask`).
  Modes reporting and garbage cleaning are unchanged.

## Impact

- `adapter/service/virtualmeter/tasks.go`; spec `devices/fimp-services`.
