# Flush logs written after Run returns

## Why

Adapters log their `+++ Stop` banner in a `defer` after `app.Run()` returns. The shutdown flush in `doStop` has already run, so the banner stays in the 64 KiB buffer and dies with the process: every clean stop looks like a crash in hub logs.

## What Changes

- `debug.DisableLogBuffering()` flushes the buffer and makes the current output write every later line straight through.
- `app.Run()` calls it on return, so lines logged after `Run` reach the file. Buffering during normal operation is unchanged.

## Impact

`debug/log_manager.go`, `root/app.go`, spec `platform/diagnostics`. Adapters need no change.
