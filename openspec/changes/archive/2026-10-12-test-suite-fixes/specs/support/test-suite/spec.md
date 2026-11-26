## MODIFIED Requirements

### Requirement: Expectation Matching And Occurrence
An `Expectation` SHALL match a message only when every one of its voters votes for it. For each
received message only the first matching expectation, in registration order, SHALL count it: its
call count is incremented, its `Publish` message (or the result of `PublishFn`, evaluated per
match) is published, and its `Reply` (or the result of `ReplyFn`) is returned as the router's
response to the message. `PublishFn` and `ReplyFn` results SHALL be used only for that match and
never stored on the expectation, so concurrent matches of one expectation do not share them.
`Occurrence` SHALL default to `AtLeastOnce` and be satisfied by at least one, exactly one, at most
one, or zero matches for `AtLeastOnce`, `ExactlyOnce`, `AtMostOnce` and `Never` respectively.

#### Scenario: request answered by the expectation
- **WHEN** an expectation with `PublishInResponse(m)` matches a command
- **THEN** `m` is published and another expectation for `m` can match it

#### Scenario: overlapping expectations
- **WHEN** two expectations both match the same message
- **THEN** only the first one registered counts it

#### Scenario: concurrent matches
- **WHEN** several messages match one expectation with a `ReplyFn` concurrently
- **THEN** each match replies with its own `ReplyFn` result and no data race occurs

### Requirement: Expectation Assertion
`Router.AssertExpectations(timeout)` SHALL return as soon as every expectation is satisfied, except
that when any expectation is `Never` it SHALL wait out the whole timeout before judging. It SHALL
poll the expectations on a short interval rather than spinning. The upper bound of `ExactlyOnce`
and `AtMostOnce` SHALL be checked only until the node's expectations are satisfied: a duplicate
arriving after the assertion returned is not attributed to that node; `Never` is the way to assert
absence over the timeout. At the timeout unmet expectations SHALL fail the test (non-fatally) with a
report listing, per unmet expectation, its occurrence, its call count and the received messages
that satisfied the most of its voters. Expectations and the message registry SHALL be cleared
afterwards, so the next `Expect` starts clean.

#### Scenario: forbidden message
- **WHEN** an expectation is marked `Never` and no matching message arrives within the timeout
- **THEN** the assertion passes only after the full timeout has elapsed

#### Scenario: upper-bounded occurrence satisfied
- **WHEN** a node's only expectation is `AtMostOnce` and no matching message has arrived
- **THEN** the assertion returns without waiting out the timeout

#### Scenario: waiting does not spin
- **WHEN** the assertion waits out the timeout for a `Never` expectation
- **THEN** it uses only a small fraction of a CPU core

### Requirement: Adapter Test Helpers
`adapterhelper.PrepareAdapter` SHALL create an adapter with its state in `workDir`, resource name
`test_adapter` and address `1`, using the given factory function; `PrepareSeededAdapter` and
`SeedAdapter` SHALL additionally initialise its things and, when seeds are given, ensure them. Any
failure SHALL fail the test fatally. `TearDownAdapter(path)` SHALL return a teardown callback
removing `<path>/data` and `<path>/data.db` every time it runs.

#### Scenario: adapter data removed
- **WHEN** the callback returned by `TearDownAdapter(dir)` runs
- **THEN** `dir/data` and `dir/data.db` no longer exist and other files in `dir` are kept

#### Scenario: callback reused
- **WHEN** the same callback runs a second time after the adapter recreated its data
- **THEN** `dir/data` and `dir/data.db` are removed again
