# Flush error lines under concurrent logging

## Why

logrus v1.10 runs `Format` outside the logger mutex and takes it only around `Write`. The global `urgent` flag set by the formatter wrapper can be overwritten by another goroutine's line between an error entry's `Format` and its `Write`, so the error line stays in the 64 KiB buffer until the periodic tick (240 s by default) and is lost if the process dies first. Under load, most error lines are hit.

## What Changes

- The formatter wrapper marks the formatted bytes of an error-level entry, keyed by the start of the slice, and `bufferedWriter.Write` flushes when it writes marked bytes. The slice logrus formats is the one it writes, so the mark cannot be taken by another goroutine's line.
- Marks for lines that never reach the buffered writer (output not yet wired) are bounded: the set is cleared once it holds 256 entries.

## Impact

`debug/log_manager.go`, spec `platform/diagnostics`. No API change.
