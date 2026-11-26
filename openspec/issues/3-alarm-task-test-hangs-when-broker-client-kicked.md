# Alarm task test hangs for 10 minutes when the test client is kicked from the broker

Seen on 2026-11-16 on branch `fix/water-heater-setpoint-object` (alarm package untouched by it).
A `make test` run hung in `adapter/service/alarm`,
`TestTaskReporting/one_change_and_one_error_per_event_during_report_cycles`, until the 10 minute
test timeout. The log shows `cliffhanger_test_suite lost connection with the broker err: EOF`
repeating: another test run on the shared local broker used the same client ID. The goroutine dump
has `task.(*manager).Stop` blocked in `sync.WaitGroup.Wait` (`task/manager.go:76`).

The previous run on the same tree failed in a different package (output lost), and a third run with
the broker idle passed all 54 packages. Related: issue 2.

Open questions:
- Should the test suite use a unique MQTT client ID per run, so parallel runs stop kicking each other?
- Is the `manager.Stop` wait a real deadlock after a connection drop, or only a slow task? Not investigated.
