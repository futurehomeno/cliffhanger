# 10 Router tests failed once in a full `go test ./...` run

Found on 2026-11-26 while checking the `internal` log-level change on `spec/error-report-policy`.

One `go test ./...` run reported `FAIL github.com/futurehomeno/cliffhanger/router 3.074s`. The run
took over 600 s in all, and its output was cut to the last lines, so the name of the failing test is
lost. The router package then passed 36 times on its own: once plain, 10 times with `-count=10`,
10 more, and 5 times with `-cpu 1`.

Suspects: a timing-dependent test under the load of the whole suite, or tests sharing the global
logrus hook (`logtest.NewGlobal`) while running in parallel.

Reproduce: `go test ./... 2>&1 | tee /tmp/all.txt`, then `grep -B2 -A15 -- '--- FAIL' /tmp/all.txt`.

Close: the failing test is named and fixed, or several full runs pass.
