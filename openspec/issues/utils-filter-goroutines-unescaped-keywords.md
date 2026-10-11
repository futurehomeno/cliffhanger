# FilterGoroutinesByKeywords compiles keywords as raw regex

`utils/tools.go:13-17` joins the keywords into `(?i)\b(k1|k2)\b` without `regexp.QuoteMeta`.
A keyword with regex metacharacters changes the match (`a.b` matches `axb`) and an invalid one
(`(`, `[`) panics in `regexp.MustCompile` — inside the stall/diagnostic path (`root/app.go:140`)
that is meant to report a problem, not cause one. An empty keyword list compiles to `\b()\b`,
which matches any word boundary and keeps every goroutine block. Close by quoting each keyword and
returning "" for an empty list; add tests for both.
