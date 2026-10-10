# Fixes from the PR #256 max review

## Why

The max-effort review of `develop` → `main` (PR #256) found five defects worth fixing before the release.

## What Changes

- The stale sweep claims a hub device only when every service with a resource name belongs to this adapter.
- A 200 token response with an empty access token counts as a rejection (`httpclient.ErrUnauthorized`), so the authenticator's grace and auth-loss handling apply.
- The water heater reuses the thermostat's setpoint `str_map` helpers. Both reject a NaN or ±Inf `temp`.
- The water heater keys its setpoint report cache by the normalized mode.
- The connectivity checker's recheck timer leaves cancellation to `check()`.

## Impact

`adapter/stale.go`, `auth/client.go`, `adapter/service/{thermostat,waterheater}`, `app/check.go`.
