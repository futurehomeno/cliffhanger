## MODIFIED Requirements

### Requirement: Startup Ordering
`Start` SHALL bring the container up in exactly this order: MQTT transport (with a 10 second connect
timeout), telemetry, the registered services in registration order, the message router, the
deduplicated topic subscriptions, the auth-loss watcher, and finally the task manager. Starting an
already running application SHALL be a no-op returning nil. On entry the lifecycle app health SHALL
be set to `STARTING`, and the working and configuration directories SHALL be logged only when the
command line is already parsed (`flag.Parsed()`); `Start` SHALL NOT register or parse flags.

#### Scenario: watcher subscribes before the first task probe
- **WHEN** a `WhenAppIsRunning`-gated task loses authorization on its very first probe immediately
  after `taskManager.Start()`
- **THEN** the auth-loss watcher — subscribed one step earlier — observes the transition and reports
  it exactly once

#### Scenario: duplicate subscriptions collapse
- **WHEN** the same topic is passed twice through `WithTopicSubscription`
- **THEN** `mqtt.Subscribe` is invoked once for that topic

#### Scenario: flags not parsed yet
- **WHEN** `Start` runs before the adapter has called `flag.Parse`
- **THEN** no `-c` flag is registered, the command line stays unparsed and the directories are not logged
