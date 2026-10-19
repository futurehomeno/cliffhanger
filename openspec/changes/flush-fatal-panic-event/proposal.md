# Flush the fatal panic event before re-panicking

## Why

`RecoverAndEmit(tel, name, true)` publishes the `panic/<name>` event and re-panics at once. The MQTT publish only queues the packet, so the crash usually kills the process before the event leaves it.

## What Changes

- With `terminate` true the panic event is published with fimpgo's `PublishSync`, which waits for the publish to complete, bounded by its timeout (5 s) after the enqueue (bounded by the 15 s write timeout, as for every publish), then re-panics. Gating is unchanged.
- `terminate` false and every other emit stay non-blocking.

## Impact

`telemetry/telemetry.go`, its mock, spec `platform/telemetry`. Adapters need no change.
