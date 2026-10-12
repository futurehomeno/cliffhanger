# Runtime and platform issue fixes

## Why

Eight open issues in `openspec/issues/` about the task manager, the root builder/app, prime, the disk
monitor, the goroutine filter and the timeline service name.

## What Changes

- `task.NewManager` drops (and logs) nil tasks, so `Start`/`Stop` keep working.
- `root.Builder` copies its routing and subscription slices into the built app.
- `root.App.Start` logs the config directory only when `-c` is registered; it never registers or parses flags.
- `prime.Device.MatchesTopic` matches a non-empty service address as the topic suffix.
- `StateDevice.GetAttribute*Value` evaluate the lookup before returning the value.
- The disk-space monitor samples through an unexported usage seam; its error and zero-usage paths are tested.
- `utils.FilterGoroutinesByKeywords` matches keywords literally and returns "" for no keywords.
- `Timeline.Event` keeps the `kind_owl` service name on purpose; the spec records it.

## Impact

`task/manager.go`, `root/builder.go`, `prime/type.go`, `monitor/diskspace.go`, `utils/tools.go`, specs.
