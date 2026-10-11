# Test Suite Specification

## Purpose
Defines the exported `test/suite` and `test/helper` framework that cliffhanger and the edge adapters
use for MQTT-driven tests: one shared client per suite, case and node ordering, and how expected
messages are matched and asserted.

## Requirements

### Requirement: Suite Run And Shared MQTT Client
`suite.Suite.Run(t)` SHALL start one MQTT client for the whole suite — `Config.MQTTServerURI`
defaulting to `tcp://127.0.0.1:11883` and `Config.MQTTClientIDPrefix` to `cliffhanger_test_suite`,
which is also the client's default FIMP source — subscribe it to `#`, run every case as a subtest
named after the case, and stop the client afterwards. Failing to start or subscribe the client
SHALL fail the test fatally before any case runs. Because the client sees every topic, a node's own
command is also delivered to that node's expectations.

#### Scenario: broker unavailable
- **WHEN** `Run` is called and no broker listens on the configured URI
- **THEN** the test fails fatally and no case runs

### Requirement: Case Setup And Lifecycle Ordering
Before its nodes run, a `Case` SHALL apply its `Setup`: a `BaseSetup` appends the routing, tasks and
mocks it returns, a `ServiceSetup` sets the case's service and appends its mocks. A case without
nodes SHALL apply the setup but start nothing. Otherwise the router (channel
`cliffhanger_test_case`, with `RouterOptions`, only when routing is present), the task manager (only
when tasks are present) and the service (only when set) SHALL be started from the first node's init
callbacks, in that order and after the first node registered its expectations, ahead of that node's
own init callbacks, so a message emitted at start-up cannot be missed. A failing start or stop
SHALL fail the test fatally. After all nodes the case SHALL assert every mock's expectations, then
run its teardown: stop router, task manager and service, in that order, then the case's own
`TearDown` callbacks.

#### Scenario: service publishes on start
- **WHEN** a case's service publishes a report from `Start` and its first node expects that report
- **THEN** the expectation is registered before the service starts and is met

#### Scenario: teardown order
- **WHEN** a case with a service and a `TearDown` callback finishes
- **THEN** the service is stopped before the callback runs

### Requirement: Node Execution
`Node.Run` SHALL, in order: start a fresh test router and register the node's expectations, run its
`InitCallbacks`, replace `Command` with the result of `CommandFn` when one is set, publish the
command (to its `Topic`, or its serialized `Addr` when the topic is empty; a nil command publishes
nothing), run its `Callbacks`, assert the expectations within `Timeout` and stop the router. A zero
`Timeout` SHALL default to 1.5 s. `SleepNode(d)` SHALL be a node that only sleeps for `d` and
publishes nothing.

#### Scenario: command built at run time
- **WHEN** a node has both `Command` and `CommandFn`
- **THEN** the message returned by `CommandFn` is published, after the init callbacks ran

### Requirement: Expectation Matching And Occurrence
An `Expectation` SHALL match a message only when every one of its voters votes for it. For each
received message only the first matching expectation, in registration order, SHALL count it: its
call count is incremented, its `Publish` message (or the result of `PublishFn`, evaluated per
match) is published, and its `Reply` (or the result of `ReplyFn`) is returned as the router's
response to the message. `Occurrence` SHALL default to `AtLeastOnce` and be satisfied by at least
one, exactly one, at most one, or zero matches for `AtLeastOnce`, `ExactlyOnce`, `AtMostOnce` and
`Never` respectively.

#### Scenario: request answered by the expectation
- **WHEN** an expectation with `PublishInResponse(m)` matches a command
- **THEN** `m` is published and another expectation for `m` can match it

#### Scenario: overlapping expectations
- **WHEN** two expectations both match the same message
- **THEN** only the first one registered counts it

### Requirement: Expectation Assertion
`Router.AssertExpectations(timeout)` SHALL return as soon as every expectation is satisfied, except
that when any expectation is `Never` it SHALL wait out the whole timeout before judging. At the
timeout unmet expectations SHALL fail the test (non-fatally) with a report listing, per unmet
expectation, its occurrence, its call count and the received messages that satisfied the most of
its voters. Expectations and the message registry SHALL be cleared afterwards, so the next
`Expect` starts clean.

#### Scenario: forbidden message
- **WHEN** an expectation is marked `Never` and no matching message arrives within the timeout
- **THEN** the assertion passes only after the full timeout has elapsed

### Requirement: Adapter Test Helpers
`adapterhelper.PrepareAdapter` SHALL create an adapter with its state in `workDir`, resource name
`test_adapter` and address `1`, using the given factory function; `PrepareSeededAdapter` and
`SeedAdapter` SHALL additionally initialise its things and, when seeds are given, ensure them. Any
failure SHALL fail the test fatally. `TearDownAdapter(path)` SHALL return a teardown callback
removing `<path>/data` and `<path>/data.db`.

#### Scenario: adapter data removed
- **WHEN** the callback returned by `TearDownAdapter(dir)` runs
- **THEN** `dir/data` and `dir/data.db` no longer exist and other files in `dir` are kept

