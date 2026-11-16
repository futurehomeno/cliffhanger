# A stop after broker loss returns no unsubscribe error

## Why

When the MQTT broker connection is lost and the app then stops, `doStop` still unsubscribes every
topic. paho rejects an unsubscribe on a client that is not connected (`not currently connected and
ResumeSubs not set`), `doStop` collects one error per topic, and `Run` returns them, so the adapter
logs `start app err: unsubscribe ...` and exits with code 1 although the stop completed.

## What Changes

- `Stop` skips the topic unsubscriptions when the MQTT transport is not connected
  (`fimpgo.MqttTransport.IsConnected`); a dead connection holds no subscriptions to remove.
- An unsubscribe failure on a connected transport is still returned as before.

## Impact

`root/app.go`, spec `runtime/app-container`. No API change; adapters need no change.
