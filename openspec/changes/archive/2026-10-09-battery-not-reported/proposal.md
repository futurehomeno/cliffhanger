# The battery reporting task skips a level not reported yet

## Why

A Hue sensor reports `"battery": null` until the bridge has its level. The battery reporting task
logs the controller's error for the level and for the low-battery alarm at every tick — 588 `E` lines
in five hours from one sensor on a hue 3.0.4 beta hub:

    E [battery] Send battery level report. err: failed to get battery level report: controller: sensor config battery is not a float64

## What Changes

- The battery reporting task skips `adapter.ErrNotReported` for the level and the alarm reports, like
  the presence, numericsensor and scenectrl tasks. A forced report still returns the error.

## Impact

- `adapter/service/battery/tasks.go`; spec `adapter/reporting`.
