# parameters SendParameterReport panics when the controller returns a nil parameter

`parameters/service.go:120` dereferences `parameter.ID` and line 126 calls `parameter.clone()` on the
value returned by `Controller.GetParameter(id)`. A controller returning `(nil, nil)` (e.g. unknown or
not-yet-read parameter) makes `SendParameterReport` panic, both from `cmd.param.get_report` /
`cmd.param.set` and from any adapter code calling it. `alarm` and `battery` explicitly treat a nil
report as "nothing to send"; `parameters` has no such contract.

Close by either returning a "parameter not found" error (or `adapter.ErrNotReported`) for a nil
parameter, with a test, or documenting on `Controller.GetParameter` that nil is forbidden.
