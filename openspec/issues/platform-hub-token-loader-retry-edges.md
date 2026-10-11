# hub: token loader retry edge cases

`hub/token.go:114-131` (`requestToken`):
- A negative `TokenLoaderConfig.Retry` skips the loop, leaves `err` and `response` nil, and then
  dereferences `response.Interface` — a nil-pointer panic instead of an error.
- After the last failed attempt the loop still sleeps `RetryDelay` (30 s by default) before giving up,
  so the default 7 attempts take ~3.5 min instead of ~3 min, and with `Retry: 1` a failed call always
  waits a full delay for nothing.
- `setDefaults` (`hub/token.go:30`) mutates the caller's config and appends `_hub_token_loader` on
  every `NewTokenLoader` call, so reusing one config yields `x_hub_token_loader_hub_token_loader`.

Close by treating `Retry < 1` as the default (or 1), not sleeping after the final attempt, and copying
the config before defaulting; add tests (the loader is otherwise untested, see the token-loader
tests added by the spec-coverage-gaps change).
