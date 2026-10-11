# prime: Device.MatchesTopic / Devices.FindByTopic match by substring

`prime/type.go:361` — `MatchesTopic` returns true when `strings.Contains(topic, srv.Addr)` for any
service. Consequences: a service whose `addr` is empty matches every topic, and a device at
`.../sv:out_bin_switch/ad:1` matches a topic for `.../sv:out_bin_switch/ad:12_0`, so `FindByTopic`
can return the wrong device depending on map/slice order. Adapters resolving a foreign device from
an incoming topic act on the wrong node.

Close by matching the address as a topic suffix (or exact address after the `pt:j1/mt:<x>` prefix)
and skipping empty addresses, with tests for both cases; then spec it in hub-integration.
