# A failed charge start or stop still sends a state report

## Why

`cmd.charge.start` and `cmd.charge.stop` return the controller's error before the forced
`evt.state.report`, so a start or stop that fails hard (REST or transport error on Zaptec, Easee)
answers with `evt.error.report` only. A client such as energy guard then keeps its last known state and
repeats the command. The operator decided (2026-11-20) that both commands always end in an
`evt.state.report`, as FH Charge in zigbee-ad does (edge-zaptec-adapter issue 014, core-energy-guard
issue 059).

## What Changes

- When `StartCharging` or `StopCharging` fails (also a rejected charging mode), the handler still sends a forced state report and
  then answers with the original error.
- When that state report fails too, its error is logged and the original error stays the answer.

## Impact

`adapter/service/chargepoint` routing, spec `devices/fimp-services`. Every charger adapter (Easee,
Zaptec) gets it with the cliffhanger bump; no API change.
