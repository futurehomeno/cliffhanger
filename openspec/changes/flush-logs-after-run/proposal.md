# Flush logs written after Run returns

## Why

Adapters log their `+++ Stop` banner in a `defer` after `app.Run()` returns. The shutdown flush in `doStop` has already run, so the banner stays in the 64 KiB buffer and dies with the process: every clean stop looks like a crash in hub logs.

## What Changes

- `debug.SetLogWriteThrough(on)`: on flushes the buffer and writes every later line straight through (whatever the output), off buffers again.
- The app's stop turns it on, its start turns it off, so lines logged after `Stop`/`Run` returns reach the file for `Run` and `Start`/`Stop` users (CloudBridge) alike. Buffering while running is unchanged.

## Impact

`debug/log_manager.go`, `root/app.go`, spec `platform/diagnostics`. Adapters need no change.
