# colorctrl advertises cmd.color.set with value type null

`adapter/service/colorctrl/specification.go:43-44` declares the incoming `cmd.color.set` interface
with `ValueType: fimptype.VTypeNull`, but `HandleCmdColorSet` (`colorctrl/routing.go`) requires an
`int_map` payload (`GetIntMapValue`) and fails on anything else. The inclusion report advertises a
command shape the handler rejects.

Close by advertising `fimptype.VTypeIntMap` for `cmd.color.set`.
