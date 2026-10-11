# thermostat/waterheater setpoint deduplication ignores the unit

`SendSetpointReport` in `adapter/service/thermostat/service.go:207` and
`adapter/service/waterheater/service.go:215` passes only the temperature `value` to
`ReportRequired`/`Reported`, although the published `evt.setpoint.report` also carries `unit`. If a
device switches units while the numeric value stays the same (or the controller corrects a wrong
unit), the periodic task considers the setpoint unchanged and the hub keeps the stale unit until a
forced report.

Probably rare in practice; close by caching the `(value, unit)` pair (e.g. the `Setpoint` struct), or
decide the unit is fixed per device and note it on the controller interface.
