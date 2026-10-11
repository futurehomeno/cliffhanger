# Uninstall failures are reported and logged as configuration failures

`app/routing.go:282` (`HandleCmdAppUninstall`) reuses `makeConfigurationReply`, so a failing
`Uninstall` replies with `op_error` `configure the app err: <error>` and logs `[app] Configure app
err` (`app/routing.go:416-421`). The hub UI and anyone reading the hub log see a configuration error
for an uninstall. The main spec (`app-service` "Configuration Reports Never Fail The Message")
currently pins the misleading text, so changing it is a FIMP-visible change.

Close it by passing an operation label into `makeConfigurationReply` (e.g. `uninstall the app err`)
and updating the requirement, or by deciding the text is part of the hub contract and leaving it.
