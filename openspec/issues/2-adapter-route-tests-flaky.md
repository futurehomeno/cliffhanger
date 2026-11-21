# Adapter route tests failed once under make test

The first `make test` on branch `fix/stop-unsubscribe-disconnected` failed
`TestRouteAdapter_ThingDelete`, `TestRouteAdapter_ThingDeleteErrorPaths` and
`TestRouteAdapter_NetworkReset` in `adapter`. Three reruns of `./adapter` and a second full
`make test` passed. The change touches only `root.doStop`; the failures look like timing against the
shared local broker. Not investigated further.
