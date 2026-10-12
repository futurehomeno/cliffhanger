## MODIFIED Requirements

### Requirement: Telemetry Configuration Storage
`DefaultStore.SetTelemetry` SHALL store a clone of the supplied configuration, including cloned
`Suppressed.Domains` and `Suppressed.Events` slices, so the caller's value cannot mutate the stored
one afterwards. `DefaultStore.Telemetry` SHALL return such a clone too, so mutating the returned
`Suppressed` entry or its slices cannot change the stored configuration. `DefaultStore.Telemetry`
SHALL return an error when no telemetry configuration is present, whereas `Default.GetTelemetry`
SHALL return the zero `TelemetryConfig` and no error in that case.

#### Scenario: Telemetry never configured
- **WHEN** `DefaultStore.Telemetry` is called and the model has no telemetry configuration
- **THEN** an error is returned

#### Scenario: Caller mutates the value it stored
- **WHEN** a caller mutates the slices of a `TelemetryConfig` it passed to `SetTelemetry`
- **THEN** the stored configuration keeps the values as of the call

#### Scenario: Caller mutates the value it read
- **WHEN** a caller mutates the `Suppressed` slices of the value returned by `DefaultStore.Telemetry`
- **THEN** a later `DefaultStore.Telemetry` returns the stored values unchanged
