# monitor: disk-space tests depend on the host's real disk usage

`monitor/diskspace.go:113` samples `disk.Usage("/")` directly and there is no seam to inject a usage
source. The original tests used limits of 90 % and 10 % and asserted the result, so on a host with a
full disk (this run: `/` at 100 %) `TestDiskSpace_Start` and `TestDiskSpace_DiskFull` failed.
`DiskFull()` is already true before `Start` because, by design, an unsampled monitor (usage 0) takes
a synchronous sample on the first call — so on a 100 % disk any limit <= 100 reports full.

The spec-coverage-gaps change rewrote those tests to use limits of 0 % (always full) and 101 %
(never full), which are host-independent. Still open: the "failed sample keeps the previous reading"
and "re-sample while usage is 0" paths cannot be tested without a usage-source seam (e.g. an
unexported `usage func() (float64, error)` field set by an internal test). Close by adding that seam
and tests for both paths.
