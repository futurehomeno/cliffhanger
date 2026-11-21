# App, auth and hub token fixes

## Why

Six open issues in `app`, `auth`, `manifest` and `hub`: a logout racing a successful probe, a doubled
init at boot, a misleading uninstall error, a dead manifest type and two retry edge cases.

## What Changes

- The checker re-checks cancellation under its lock while applying a successful probe, so a logout or
  reset landing after the `Authorized` gate wins.
- The init retry task skips its first run; only the one-shot task initializes at boot.
- A failing uninstall replies `uninstall the app err: <error>` (FIMP-visible text change) and logs an
  uninstall error.
- `manifest.AuthResponse` is deprecated in favour of `app.AuthenticationReport`.
- `ProxyClient` treats a negative `Retry` as 0.
- The token loader treats `Retry < 1` as the default, does not sleep after the final attempt and no
  longer mutates the caller's config.

## Impact

`app/{check,task,routing}.go`, `manifest/manifest.go`, `auth/client.go`, `hub/token.go`.
