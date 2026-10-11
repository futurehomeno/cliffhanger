# evt.network.all_nodes_report carries JSON null when no thing is registered

`Adapter.SendConnectivityReport` (adapter/adapter.go:429) builds `var connectivityReports ConnectivityReports`
and appends one report per thing. With no registered things the slice stays nil and the published
`evt.network.all_nodes_report` value marshals as `null` instead of `[]`. The thing-level report
deliberately sanitises a nil operationability list to `[]` for the same reason (hub/UI consumers
iterate the array). The network-commands spec only says the report carries "no connectivity reports".
Close it by deciding the wire shape: initialise the slice (`ConnectivityReports{}`) and add a
scenario asserting `[]`, or spec that `null` is accepted by the hub.
