# ProxyClient returns (nil, nil) for a negative Retry

`auth/client.go:111` loops `for i := 0; i <= c.cfg.Retry; i++`. With `ProxyClientConfig.Retry < 0`
the loop never runs and `getToken` returns `nil, nil`: no request is made, and the caller
(`Authenticator.AccessToken`, `app` authorize flow) dereferences a nil `*OAuth2TokenResponse`.
`setDefaults` does not clamp `Retry`. Close by clamping `Retry` to 0 in `setDefaults` (or by always
making at least one attempt) and adding a test for `Retry: -1`.
