# Router logs a state not reported yet as a warning

## Why

A FIMP `get_report` for a state the device has never reported (a Hue sensor switched off in the Hue
app) returns `adapter.ErrNotReported`. The router logs it as `E [router] Process message`, one error
line per request, although nothing is wrong with the adapter.

## What Changes

- `router.ErrNotReported` is the sentinel; `adapter.ErrNotReported` refers to the same value (the
  router cannot import `adapter`). No API break.
- The handler logs an error wrapping it at warning level and still replies with `evt.error.report`.
- Typo "reeport" in the presence service error.

## Impact

- `router/handler.go`, `adapter/service.go`, `adapter/service/presence/service.go`; spec
  `messaging/routing`.
