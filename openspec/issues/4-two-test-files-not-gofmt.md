# Two test files are not gofmt-formatted

`gofmt -l adapter` on `develop` (5937f52, 2026-11-23) lists
`adapter/service/chargepoint/cable_lock_test.go` and `adapter/service/outlvlswitch/level_test.go`.
`golangci-lint run` does not report them.

Reproduce: `gofmt -l adapter`.

Closes when both files are formatted (`gofmt -w`), or when the linter enforces gofmt on tests.
