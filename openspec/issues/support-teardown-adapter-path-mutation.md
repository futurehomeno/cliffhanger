# adapterhelper.TearDownAdapter callback mutates its captured path

`test/helper/adapter/helpers.go:85-86` does `path = filepath.Join(path, "data")` on the variable
captured by the returned closure. The first run removes `<dir>/data` and `<dir>/data.db`; any
second run of the same callback (a case reused across suites, or the slice shared by several
cases) removes `<dir>/data/data` and `<dir>/data/data.db` instead, leaving the real adapter state
behind for the next case. Close by using a local (`dataDir := filepath.Join(path, "data")`).
