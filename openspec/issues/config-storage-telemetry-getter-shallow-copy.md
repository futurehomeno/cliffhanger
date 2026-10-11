# config: DefaultStore.Telemetry returns a copy that shares Suppressed with the stored config

`DefaultStore.Telemetry` (config/default.go:206) returns `*s.accessor().Telemetry`, a shallow copy:
its `Suppressed` pointer and the `Domains`/`Events` slices alias the stored configuration. Every
other accessor in the store hands out deep copies (`Default()` clones via `TelemetryConfig.Clone`,
`SetTelemetry` stores a clone). Today it is harmless only because the sole caller
(telemetry/telemetry.go:356) clones the result and all writers replace the pointer rather than
mutate in place; any adapter mutating the returned `Suppressed` mutates live state outside the lock.

Close by returning `s.accessor().Telemetry.Clone()` and adding the getter to the Telemetry
Configuration Storage requirement with a test.
