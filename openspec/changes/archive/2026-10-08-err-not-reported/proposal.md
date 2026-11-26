# Reporting tasks skip a state not reported yet

## Why

A device can exist before it has reported a state: a Hue sensor the bridge has no reading for yet,
or a switch never pressed since the bridge started. The controller has nothing to return, so it
errors, and the periodic reporting task logs that error at every tick (four `E` lines a minute per
sensor on a hue hub).

## What Changes

- `adapter.ErrNotReported` lets a controller say "nothing reported yet".
- The presence, numericsensor and scenectrl reporting tasks skip it without logging. A forced report
  (e.g. a FIMP `get_report`) still returns the error.

## Impact

- `adapter/service.go`, `adapter/service/{presence,numericsensor,scenectrl}/tasks.go`; spec
  `adapter/reporting`.
