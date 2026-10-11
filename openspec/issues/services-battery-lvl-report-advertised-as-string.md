# battery advertises evt.lvl.report as string but publishes an int

`adapter/service/battery/specification.go:45` declares the outgoing `evt.lvl.report` interface with
`ValueType: fimptype.VTypeString`, while `SendBatteryLevelReport` (`battery/service.go`) publishes it
with `fimpgo.NewIntMessage`. The inclusion report therefore lies about the value type of the battery
level; a hub/UI that trusts the advertised interface may parse the level wrongly.

Close by advertising `fimptype.VTypeInt` (and updating the specification test fixtures), after
checking no consumer depends on the current `string` declaration.
