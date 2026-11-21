## MODIFIED Requirements

### Requirement: Factory Reset
`Reset` SHALL stop the application first — logging but not aborting on a stop error, and treating an
application that is not running as already stopped — and then invoke every registered resetter in
registration order, returning on the first resetter error. The result delivered to a pending `Wait`
(and so returned by `Run`) SHALL be the reset result, not the stop result. The container SHALL handle
the FIMP event `evt.gateway.factory_reset` received on `pt:j1/mt:evt/rt:ad/rn:gateway/ad:1` by
running `Reset` in a separate goroutine, because the reset stops the message router that is
dispatching the very message. The handler SHALL produce no reply and SHALL suppress error replies
(`router.WithSilentErrors`).

#### Scenario: gateway factory reset event
- **WHEN** `evt.gateway.factory_reset` arrives on the gateway event topic and a resetter is registered
- **THEN** the application is stopped and the resetter's `Reset` is invoked
- **AND** no FIMP reply is published for the event

#### Scenario: a resetter fails
- **WHEN** the first of two registered resetters returns an error
- **THEN** `Reset` returns that error wrapped, the second resetter is not invoked, and a pending
  `Wait` returns the same error

### Requirement: Run And Wait
`Run` SHALL start the application, install a SIGINT/SIGTERM handler, and block until the application
stops, returning the shutdown error. `Wait` SHALL return nil immediately when the application is not
running, otherwise block on the internal error channel. Delivery of the shutdown result to that
channel SHALL be non-blocking, so a `Stop` with no waiter does not deadlock, and `Start` SHALL discard
a result left buffered there by a previous run, so a `Wait` after a restart blocks for the new run
instead of returning the old shutdown error. `Run` SHALL recover from a panic in itself or the signal
goroutine, emit a panic telemetry event and re-panic.

#### Scenario: signal-driven shutdown
- **WHEN** the process receives SIGTERM while running
- **THEN** a filtered goroutine dump (entries matching mutex, semaphore, panic or lock) is logged at
  warning level
- **AND** `Stop` is invoked and `Run` returns its error

#### Scenario: reboot milestone
- **WHEN** `Run` starts an edge application whose lifecycle restart count is a positive multiple of 500
- **THEN** a reboot-milestone telemetry event carrying that count is emitted

#### Scenario: wait after a restart
- **WHEN** a `Stop` that failed with no waiter is followed by `Start`, then `Wait`, then a clean `Stop`
- **THEN** `Wait` returns nil from the second stop, not the first stop's error
