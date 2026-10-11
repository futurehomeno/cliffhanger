# root.App.Start parses command-line flags as a side effect

`root/app.go` `doStart` calls `logBootstrapDirs` (`root/builder.go:142`), which calls
`bootstrap.GetConfigurationDirectory()`. When no `-c` flag is registered yet, that registers it and
calls `flag.Parse()` on `flag.CommandLine` (ExitOnError). An adapter that registers further flags
after building/starting the app, or is started with a flag it never defined, gets `os.Exit(2)` from
inside `Start`, and the parse happens on every start until `-c` exists. It is also not specced in
runtime/app-container Startup Ordering.

Decide whether `Start` should only log directories when the flag set is already parsed
(`flag.Parsed()`), or document the side effect in the Startup Ordering requirement.
