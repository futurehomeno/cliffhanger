# FIMP Message Routing Specification

## Purpose
Every FIMP command an adapter receives over MQTT enters through the router. The router owns the
dispatch contract — which handler sees which message, how many messages are processed at once, where
a reply is published and what happens when a handler fails — so adapter authors write only
`MessageProcessor` bodies and inherit consistent addressing, error reporting and panic containment.

## Requirements

### Requirement: Routing Composition
A `Routing` SHALL consist of exactly one `MessageHandler` and zero or more `MessageVoter`s, created
by `NewRouting(handler, voters...)`. `Routing.Wrap(voters...)` SHALL return a new routing reusing the
same handler with the additional voters prepended to the existing ones; it SHALL NOT mutate the
receiver. `Combine[T]` SHALL flatten a mix of `*Routing` values and `[]*Routing` slices into a single
slice, preserving argument order.

#### Scenario: wrapping a routing
- **WHEN** `Wrap` is called on a routing that already has voters
- **THEN** a new routing is returned whose voter list is the wrapping voters followed by the original ones
- **AND** the original routing keeps its own voter list unchanged

#### Scenario: combining routings
- **WHEN** `Combine` receives both single routings and slices of routings
- **THEN** the result contains every routing in the order the arguments were given

### Requirement: Router Registration On The MQTT Transport
The router SHALL register a single buffered `fimpgo.MessageCh` on the supplied `fimpgo.MqttTransport`
under the channel ID given to `NewRouter`, and the transport SHALL fan every received message into
that channel. The root application SHALL use `router.DefaultChannelID` (`"main_router"`). Because the
transport keys registered channels by ID, two routers sharing a channel ID SHALL NOT both receive
messages — any additional router MUST be constructed with a distinct channel ID.

#### Scenario: router start
- **WHEN** `Start` is called on a stopped router
- **THEN** a message channel is registered on the transport under the router's channel ID and worker goroutines begin consuming it

#### Scenario: double start or double stop
- **WHEN** `Start` is called on an already running router, or `Stop` on an already stopped one
- **THEN** an error is returned and the router state is left untouched

#### Scenario: router stop
- **WHEN** `Stop` is called on a running router
- **THEN** the channel is unregistered from the transport first, the workers are signalled, and the call blocks until every worker goroutine has returned

### Requirement: Topic Pattern Rendering
`TopicPattern.String()` SHALL render an MQTT subscription topic out of the seven FIMP address fields,
substituting the single-level wildcard `+` for every field left empty. A pattern whose resource type
is `discovery` SHALL stop after the resource type, so it renders three segments; `ad`, `app` and
`cloud` patterns SHALL end after the resource name and resource address, so they render five; every
other resource type SHALL render all seven, including service name and service address. The helpers
`TopicPatternAdapter`, `TopicPatternDevice`, `TopicPatternApplication`, `TopicPatternDeviceService`
and `TopicPatternRoomService` SHALL build the common patterns with the default payload type `pt:j1` —
the first three pinning the resource address to `1`, `TopicPatternRoomService` pinning the resource
name to `room` — and `CombineTopicPatterns` SHALL concatenate pattern slices preserving argument
order. The rendered strings are what an adapter passes to the root builder as topic subscriptions.

#### Scenario: unset fields render as wildcards
- **WHEN** a device pattern sets only the payload type, resource type and service name
- **THEN** the rendered topic is `pt:j1/+/rt:dev/+/+/sv:<service>/+`

#### Scenario: discovery pattern is truncated
- **WHEN** a pattern's resource type is `discovery`
- **THEN** the rendered topic ends at the resource type, for example `+/mt:evt/rt:discovery`

### Requirement: Concurrency And Buffer Defaults
The router SHALL default to 5 concurrent worker goroutines and an incoming message buffer of 10.
`WithSyncProcessing()` SHALL set concurrency to 1, guaranteeing that messages are processed one at a
time in arrival order. `WithAsyncProcessing(n)` SHALL set concurrency to `n` but SHALL ignore any `n`
below 2. `WithMessageBuffer(n)` SHALL set the channel buffer to `n` but SHALL ignore negative values.

#### Scenario: async processing below the minimum
- **WHEN** `WithAsyncProcessing(1)` or `WithAsyncProcessing(0)` is applied
- **THEN** the option is ignored and the concurrency stays at its previous value (5 by default)

#### Scenario: synchronous processing
- **WHEN** `WithSyncProcessing()` is applied
- **THEN** exactly one worker goroutine is started and no two messages are handled concurrently

### Requirement: Per-Message Routing Walk
A worker SHALL, for each message it takes from the channel, walk every registered routing in
registration order and process the message with each routing whose voters all accept it. Matching
SHALL NOT stop at the first accepting routing, so a message accepted by several routings is handled
several times. If an accepting routing has no handler, the router SHALL log the missing handler for
the message topic and continue with the next routing.

#### Scenario: two routings match one message
- **WHEN** a message satisfies the voters of two routings
- **THEN** both handlers are invoked, in registration order, for that single message

#### Scenario: handler implemented on a value receiver
- **WHEN** a routing's handler is a non-pointer value implementing `MessageHandler`
- **THEN** it is treated as present and invoked, not mistaken for a nil handler

### Requirement: Voter Conjunction
A routing SHALL process a message only if every one of its voters returns true; a routing with no
voters SHALL accept every message. `ForTopic`, `ForMessageType`, `ForResourceType`, `ForResourceName`
and `ForSource` SHALL match on the message address or payload source; `ForService`, `ForType` and
`ForServiceAndType` SHALL match the payload's service and interface; `ForServicePrefix` SHALL match
services by string prefix; `ForProperty` SHALL match a payload property by exact value and SHALL
reject messages where the property is absent. `Or` SHALL accept when at least one nested voter
accepts, and `Not` SHALL invert a voter.

#### Scenario: property voter with a missing property
- **WHEN** `ForProperty` is asked to vote on a message whose payload does not carry the property
- **THEN** the voter returns false and the routing is skipped

#### Scenario: one voter rejects
- **WHEN** a routing has several voters and any one of them returns false
- **THEN** the handler is not invoked and no stats callback is fired for that routing

### Requirement: Reply Address Resolution
When a handler returns a non-nil reply, the router SHALL publish it to the address parsed from the
incoming message's `ResponseToTopic` when that field is set, and otherwise to the reply's own
`Addr`. If `ResponseToTopic` is empty and the reply carries no address, the reply SHALL be dropped
without publishing. If `ResponseToTopic` cannot be parsed as a FIMP address, the error SHALL be
logged and nothing published. With `WithPreservedGlobalPrefix()` the router SHALL copy the incoming
message's global prefix onto the reply address; without it the prefix is left as resolved.

#### Scenario: request carries a response topic
- **WHEN** the incoming payload sets `ResponseToTopic` and the handler returns a reply with its own address
- **THEN** the reply is published to the `ResponseToTopic` address, not to the reply's address

#### Scenario: no reply address available
- **WHEN** the incoming payload has no `ResponseToTopic` and the handler's reply has a nil address
- **THEN** nothing is published and processing of the message continues normally

### Requirement: Correlation Identifier Propagation
The router SHALL set the reply payload's correlation ID to the incoming payload's UID, unless the
handler already set a non-empty correlation ID on the reply, which SHALL be preserved.

#### Scenario: default correlation
- **WHEN** a handler returns a reply whose correlation ID is empty
- **THEN** the published reply carries the incoming message's UID as its correlation ID

### Requirement: Panic Containment Per Message
The router SHALL recover a panic raised while voting on or handling a single message, log it with the
message topic, service, type and stack, and continue processing subsequent messages. When a panic
callback is configured with `WithPanicCallback`, the router SHALL invoke it with the offending
message and the recovered value. A panic escaping the worker loop itself SHALL be logged with its
stack and re-raised.

#### Scenario: handler panics
- **WHEN** a message handler panics
- **THEN** the panic is recovered and logged, the panic callback (if any) receives the message and the recovered value
- **AND** the worker goroutine stays alive and processes the next message

### Requirement: Default Message Handler Reply Addressing
`NewMessageHandler` SHALL derive the reply address from the incoming message address by copying its
payload type, resource type, resource name, resource address, service name and service address and
forcing the message type to `evt`. `WithDefaultAddress` SHALL make the handler derive the reply from
the supplied address instead of the request address.

#### Scenario: inferred reply address
- **WHEN** a handler built by `NewMessageHandler` replies to a command addressed to a device
- **THEN** the reply is addressed to the same resource and service with the message type set to `evt`

### Requirement: Error Reporting Contract
When the `MessageProcessor` returns an error, the handler SHALL log it with the request topic,
service and type, and SHALL reply with an `evt.error.report` message carrying the request's service,
a string value `"failed to process incoming message"` and the properties `msg` (the error text),
`cmd_topic`, `cmd_service` and `cmd_type` taken from the request. `WithSilentErrors()` SHALL suppress
the reply while keeping the log. An error reply addressed to a device resource SHALL be marked with
the skip storage strategy so device errors are not persisted by the hub.

#### Scenario: processor returns an error
- **WHEN** a message processor returns an error and silent errors are not enabled
- **THEN** an `evt.error.report` is replied with `msg`, `cmd_topic`, `cmd_service` and `cmd_type` properties

#### Scenario: device error not stored
- **WHEN** the error reply resolves to a `rt:dev` address and the payload has no storage strategy
- **THEN** the payload is tagged with the skip storage strategy before publishing

### Requirement: Success Confirmation
A handler configured with `WithSuccessConfirmation()` SHALL, when its processor returns no reply and
no error, respond with an `evt.success.report` carrying a null value and the properties `cmd_topic`,
`cmd_service` and `cmd_type`. Without that option a processor returning a nil reply SHALL produce no
response at all.

#### Scenario: silent processor with confirmation enabled
- **WHEN** a processor returns `nil, nil` and the handler was built with `WithSuccessConfirmation()`
- **THEN** an `evt.success.report` is published to the inferred reply address

### Requirement: Handler Serialisation Lock
`WithLock()` SHALL give a handler a private try-lock and `WithExternalLock(locker)` SHALL make it
share a caller-supplied `MessageHandlerLocker`. A locked handler SHALL attempt the lock without
blocking: on failure it SHALL NOT invoke the processor and SHALL instead take the error path with
"another operation is already running, skipping message". The framework shares one locker instance
across the app service handlers (configuration writes, reset, uninstall, login, token set, logout,
config actions) and the adapter's `cmd.thing.delete` handler, so those mutating operations SHALL
never run concurrently with one another.

#### Scenario: second command while one is in flight
- **WHEN** a second message reaches a handler whose shared locker is already held
- **THEN** the processor is not invoked and an error report is returned (or only logged, with `WithSilentErrors`)
- **AND** the in-flight operation is unaffected

#### Scenario: lock released after processing
- **WHEN** a locked handler finishes processing, with or without an error
- **THEN** the lock is released and the next message is accepted

### Requirement: Processing Statistics And Redaction
When a stats callback is configured with `WithStatsCallback`, the router SHALL invoke it once for
every message a routing actually handled, passing the input message, the handler's reply (nil when
there was none) and the wall-clock handling duration. `DefaultLogStats(prefixes...)` SHALL log
incoming messages at info level and outgoing ones at debug level, and SHALL replace the value of any
incoming message whose interface starts with one of the given prefixes (for example `cmd.auth.`) with
`***`, so credentials never reach the log.

#### Scenario: handler returned no reply
- **WHEN** a handler processes a message and returns nil
- **THEN** the stats callback is still invoked, with a nil output message and the measured duration

#### Scenario: credential-carrying command
- **WHEN** `DefaultLogStats("cmd.auth.")` logs an incoming `cmd.auth.login` message
- **THEN** the logged value is `***` instead of the payload value
