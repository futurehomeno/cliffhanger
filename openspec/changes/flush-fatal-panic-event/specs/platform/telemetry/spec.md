## MODIFIED Requirements

### Requirement: Panic Reporting
`RecoverAndEmit(tel, name, terminate)` SHALL be usable as a deferred recover: when nothing panicked
it SHALL do nothing. On a recovered panic it SHALL log the value and stack at error level and emit a
`panic`-domain event whose event name is `name` and whose data carries `terminate`. When `terminate`
is true and the event topic is a FIMP address with a j1 or j1c1 payload (the default) it SHALL wait for that event's publish
to complete, bounded by the transport's sync publish timeout (on top of the enqueue, which every
publish already bounds by the client's write timeout), and then re-panic with the original value, so
a fatal panic is reported before it crashes the process and a stalled broker cannot hold the crash
back indefinitely. When `terminate` is false the emit SHALL not wait.

#### Scenario: non-fatal recovery
- **WHEN** a goroutine guarded by `RecoverAndEmit(tel, "poller", false)` panics
- **THEN** the panic is logged, a `panic`/`poller` event is emitted without waiting for the publish,
  and the goroutine returns normally

#### Scenario: fatal recovery
- **WHEN** a goroutine guarded by `RecoverAndEmit(tel, "run", true)` panics
- **THEN** the `panic`/`run` event's publish is awaited with a bounded timeout before the original
  value is re-panicked
