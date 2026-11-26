# 11 test/suite tests failed once right after a build

Found on 2026-11-26 while checking `spec/error-report-policy` after #295 was merged (25d9c05).

`go build ./... && go vet … && go test ./router/ ./test/suite/` reported
`FAIL github.com/futurehomeno/cliffhanger/test/suite 2.099s`. The output was cut to its last lines, so
the failing test is not named. The package then passed 41 times: once plain, `-count=8`, and
`-count=4` six times.

Suspects: the broker-backed tests (`TestSuite_CaseLifecycle`, `TestRouterTestSuite`) timing out under
the load of the build that ran just before; maybe the same cause as issue 10.

Reproduce: `go build ./... && go test -count=1 ./test/suite/ 2>&1 | tee /tmp/suite.txt`, then
`grep -B2 -A15 -- '--- FAIL' /tmp/suite.txt`.

Close: the failing test is named and fixed, or runs under load pass repeatedly.
