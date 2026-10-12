# Empty access token is transient

## Why

PR #275 made a 200 token response with an empty access token a transient error, but the spec still called it a rejection.

## What Changes

- An empty access token is retried like any transient failure and does not match `httpclient.ErrUnauthorized`, so it does not start the authenticator's rejection streak.

## Impact

`openspec/specs/auth/oauth2/spec.md` only; `auth/client.go` already behaves this way.
