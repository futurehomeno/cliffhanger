# Process Bootstrap Specification

## Purpose
The `bootstrap` package is the thin layer between the operating system and the application
container: it resolves where configuration and data live, blocks the main goroutine until the hub
terminates the process, and assembles the route and task sets every edge adapter is expected to
expose. The `discovery` package supplies the one route no cliffhanger application may omit — the
responder that tells the hub which resource, package, instance and version this process is.

## Requirements

### Requirement: Configuration Directory Resolution
`GetConfigurationDirectory` SHALL return the value of the `-c` command line flag, falling back to
`./` when the flag is absent or empty. It SHALL register and parse the `-c` flag itself only when no
flag named `c` has been registered yet, so an application that defines its own `-c` flag keeps
ownership of it.

#### Scenario: flag supplied
- **WHEN** the process is started with `-c /var/lib/futurehome/app`
- **THEN** `GetConfigurationDirectory` returns `/var/lib/futurehome/app`

#### Scenario: no flag supplied
- **WHEN** the process is started with no `-c` flag
- **THEN** `GetConfigurationDirectory` returns `./`

### Requirement: Working Directory Resolution
`GetWorkingDirectory` SHALL return the process working directory, and SHALL return `./` instead of an
error when the working directory cannot be determined.

#### Scenario: working directory unavailable
- **WHEN** the working directory cannot be resolved
- **THEN** `GetWorkingDirectory` returns `./`

### Requirement: Shutdown Barrier
`WaitForShutdown` SHALL block until the process receives SIGINT or SIGTERM, and SHALL deregister its
signal handler before returning so a later handler installation is unaffected.

#### Scenario: termination signal
- **WHEN** the hub sends SIGTERM to the adapter process
- **THEN** `WaitForShutdown` returns and the signal notification is stopped

### Requirement: Default Route Bundle
`DefaultRoute` SHALL return the routes every service exposes: the configuration report route, the
eight debug log-control routes and the six telemetry configuration routes — fifteen routes in total.
The `config.RoutingOption` values passed to it SHALL be forwarded to the configuration-change
publication of the debug and telemetry setter routes.

#### Scenario: bundle contents
- **WHEN** `DefaultRoute` is called with a service name, a config getter and a live telemetry service
- **THEN** the returned slice contains fifteen routes
- **AND** `cmd.config.get_report`, `cmd.log.get_level` and `cmd.config.get_telemetry_enabled` are all
  answered by it

#### Scenario: telemetry disabled at build time
- **WHEN** `DefaultRoute` is called with a nil telemetry service
- **THEN** the six telemetry routes are omitted and the remaining routes still work

### Requirement: Configuration Report Route
The configuration report route SHALL answer `cmd.config.get_report` addressed to the service with
`evt.config.report` of type object, whose value is whatever the supplied getter returns at the moment
of the request. The getter SHALL be re-invoked per request rather than snapshotted at wiring time.

#### Scenario: custom payload
- **WHEN** the getter returns an application-specific struct and `cmd.config.get_report` arrives
- **THEN** `evt.config.report` is published carrying that struct serialized as the object value

### Requirement: Edge Route Composition
`EdgeRouting` SHALL combine, in this order, the `DefaultRoute` bundle, the app-service routes from
`app.RouteApp` and the adapter routes from `adapter.RouteAdapter`, followed by any extra route slices
passed as variadic arguments. The `excludeAllThings` callback SHALL be forwarded to `app.RouteApp`
and the routing options SHALL be forwarded to `adapter.RouteAdapter`.

#### Scenario: extras are appended
- **WHEN** `EdgeRouting` is called with two extra route slices
- **THEN** the result contains the default, app and adapter routes followed by both extra slices

### Requirement: Edge Task Composition
`EdgeTasks` SHALL combine the app tasks from `app.TaskApp` and the adapter tasks from
`adapter.TaskAdapter` — the latter driven by the supplied reporting interval — followed by any extra
task slices passed as variadic arguments.

#### Scenario: reporting interval is forwarded
- **WHEN** `EdgeTasks` is called with a one minute reporting interval
- **THEN** the adapter reporting task in the result runs on that interval

### Requirement: Service Discovery Responder
The discovery route SHALL listen on the topic `pt:j1/mt:cmd/rt:discovery` for the message type
`cmd.discovery.request` and SHALL reply with an object message of type `evt.discovery.report` from
the service `system`, carrying `resource_name`, `resource_type`, `package_name`, `instance_id` and
`version`. The reply SHALL correlate with the request payload so the router can address it to the
requester's response topic.

#### Scenario: discovery request
- **WHEN** `cmd.discovery.request` arrives on `pt:j1/mt:cmd/rt:discovery`
- **THEN** `evt.discovery.report` is returned with the application's resource name, resource type
  (`app` or `ad`), package name, instance ID and version

### Requirement: Discovery Reports Fresh App State
When a lifecycle instance was supplied, each discovery reply SHALL include the application states
read at the moment of the request under the `app_state` field. When no lifecycle was supplied — a
core application — the `app_state` field SHALL be omitted from the report.

#### Scenario: state changes between requests
- **WHEN** app health changes to `RUNNING` between two discovery requests
- **THEN** the second reply reports `RUNNING` while the first reported the earlier state

#### Scenario: core application
- **WHEN** the responder was built with a nil lifecycle
- **THEN** the report contains no `app_state` field
