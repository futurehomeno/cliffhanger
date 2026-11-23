# Session current rejects only negative values

## Why

`cmd.current_session.set_current` shares the validation of `cmd.max_current.set`, so a session
current below 6 A is rejected. A client that balances load (energy guard) has to be able to lower
the offered current of a running session under 6 A, down to 0 A, and let the charger decide what
it does with it. Only a negative value is meaningless.

## What Changes

- `cmd.current_session.set_current` rejects a negative current and a current above `sup_max_current`;
  0–5 A reach the controller.
- `cmd.max_current.set` keeps its 6 A floor.

## Impact

`adapter/service/chargepoint` service, spec `devices/fimp-services`. Charger adapters (Easee,
Zaptec) get it with the cliffhanger bump; an adapter whose charger refuses low currents has to
reject them in its controller. No API change.
