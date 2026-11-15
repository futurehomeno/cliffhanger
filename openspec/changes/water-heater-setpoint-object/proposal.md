# Water heater setpoint is an object again

## Why

hoiax 2.1.0 (cliffhanger 1.5.1) rejects every `cmd.setpoint.set` on `water_heater` with `provided setpoint string map has an incorrect format` (49 hits on beta hub 26121a89 since the upgrade on 2026-11-10; the same commands were accepted by hoiax 2.0.0 on cliffhanger 1.3.7).

The FIMP specification (`openspec/docs/fimp-api-water_heater.md`) defines `cmd.setpoint.set` and `evt.setpoint.report` of `water_heater` as `object` (`{"type": "boost", "temp": 85.0, "unit": "C"}`). Up to 1.3.7 the service read and published an object but advertised `str_map`. #257 (1.4.0) resolved the mismatch the wrong way: it moved the handler and the report to `str_map`, so the senders that follow the specification are rejected and the report changed its wire format.

## What Changes

- `water_heater` reads `cmd.setpoint.set` as an object and publishes `evt.setpoint.report` as an object.
- The service specification advertises `object` for both.
- `waterheater.SetpointFromStringMap` is removed (added in 1.4.0, only used by the handler).

The thermostat keeps `str_map`, as its specification says.

## Impact

`adapter/service/waterheater`, spec `devices/fimp-services`. Adapters with a water heater (hoiax) get the fix with the cliffhanger bump; a caller of `waterheater.SetpointFromStringMap` uses `thermostat.SetpointFromStringMap`.
