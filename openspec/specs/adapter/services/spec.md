# Adapter Services Specification

## Purpose
Defines what a FIMP service is inside the framework: an addressable unit that owns one service
specification, delegates device work to an adapter-supplied Controller or Reporter, and reaches the
outside world only through a Publisher. It also defines the capability-by-interface convention that
lets one service implementation cover devices with different feature sets, and the shape of the
service events that in-process listeners subscribe to.

## Requirements

### Requirement: Service Identity Is Its Specification Address
A service SHALL take its name from `Specification().Name` and its topic from
`Specification().Address`; `Topic()` and the specification address SHALL never diverge. A service
address SHALL follow the pattern `/rt:dev/rn:<adapter>/ad:<adapterAddress>/sv:<serviceName>/ad:<serviceAddress>`,
which is the key a thing indexes its services by and the address the publisher parses when sending
a service message.

#### Scenario: Service registered on a thing
- **WHEN** a thing is constructed with a service whose specification address is `/rt:dev/rn:easee/ad:1/sv:chargepoint/ad:1`
- **THEN** `ServiceByTopic` for that address returns the service
- **AND** the same address appears in the thing's inclusion report service list

### Requirement: Specification Options
A service specification SHALL be produced by the service package's `Specification` constructor,
which fills in the address, name, groups, `Enabled` and the required interfaces, and then applies
every supplied `SpecificationOption` in order. An option SHALL mutate the `fimptype.Service` in
place; `SpecificationOptionFn` SHALL adapt a plain function to that interface. Options are the only
supported way for an adapter to declare service properties such as supported charging modes,
supported max current, grid type or phase count.

#### Scenario: Property declared through an option
- **WHEN** a chargepoint specification is built with `WithSupportedMaxCurrent(32)`
- **THEN** the specification's props carry the supported max current
- **AND** the value is announced to the hub as part of the thing's inclusion report

### Requirement: Controller And Reporter Split
An adapter author SHALL implement the device-specific behaviour behind the interface the service
package defines for it — `Controller` where the service accepts commands, `Reporter` where the
service is read-only, as in `numericmeter`, `numericsensor`, `battery` and `alarm` — and SHALL NOT
implement the `Service` interface itself. A service SHALL translate a controller error into a wrapped error
prefixed with the service name and SHALL NOT publish a report when the controller failed.

#### Scenario: Controller fails while reporting
- **WHEN** a controller's state read returns an error during a send-report call
- **THEN** the service returns an error naming the service and sends no FIMP message
- **AND** the reporting cache is left unchanged so the next attempt still reports

### Requirement: Capability By Interface
Optional device capabilities SHALL be expressed as additional interfaces that the adapter's
controller may implement, discovered by type assertion at construction time. Each optional
capability SHALL expose a `Supports*()`/`Is*Aware()` predicate that is exactly that type assertion,
and `NewService` SHALL add the capability's FIMP interfaces to the service specification — via
`EnsureInterfaces`, so no interface is duplicated — only when the predicate holds. Invoking an
unsupported capability SHALL return a "not supported" error rather than panicking.

#### Scenario: Controller implements an optional reporter
- **WHEN** a diagnostic service is built with a controller that implements `LQIReporter` but not `RSSIReporter`
- **THEN** `SupportsLQI` is true, `SupportsRSSI` is false
- **AND** the emitted specification contains the `cmd.lqi.get_report`/`evt.lqi.report` interfaces and no RSSI interfaces

#### Scenario: Unsupported capability invoked
- **WHEN** a send-report call is made for a capability the controller does not implement
- **THEN** the service returns an error stating the functionality is not supported and publishes nothing

### Requirement: Default Reporting Strategy Per Service
A service configuration SHALL treat a nil `ReportingStrategy` as a request for the service package's
`DefaultReportingStrategy`, substituted at construction time. Each service SHALL own its own
`cache.ReportingCache` instance, so deduplication is scoped to one service and never shared with
other services on the same thing. Strategy and cache semantics are specified once in the
`adapter/reporting` capability; this requirement governs only their per-service wiring, and the
per-package default values are tabulated in `devices/fimp-services`.

#### Scenario: Strategy omitted
- **WHEN** an out binary switch service is configured without a reporting strategy
- **THEN** the service reports on change only, that being its package default

### Requirement: Publisher Is The Only Egress
A service SHALL send FIMP messages only through `ServicePublisher.PublishServiceMessage` and publish
events only through `ServicePublisher.PublishServiceEvent`; nothing below the publisher SHALL hold
the MQTT transport or the event manager. The publisher SHALL derive the outbound address from the
service topic, force its message type to `evt`, and overwrite the message's service field with the
service name, so a service cannot publish a command or impersonate another service.

#### Scenario: Service report sent
- **WHEN** a service calls `SendMessage` with an event message
- **THEN** the message is published to the service's own address with message type `evt` and service name set from the specification

#### Scenario: Unparsable topic
- **WHEN** a service topic cannot be parsed as a FIMP address
- **THEN** the publish fails with an error naming the offending topic and nothing is sent

### Requirement: Thing And Adapter Messages Use The Adapter Topic
`PublishThingMessage` and `PublishAdapterMessage` SHALL both publish to the adapter's own address —
resource type `ad`, the adapter resource name and the adapter resource address — with the message's
service field set to the adapter name. The thing passed to `PublishThingMessage` SHALL NOT influence
the destination topic; a thing's node, ping and inclusion reports therefore all appear on the single
adapter topic rather than on per-thing topics.

#### Scenario: Node report destination
- **WHEN** a thing publishes its `evt.network.node_report`
- **THEN** the message is addressed to the adapter resource, not to any service address of the thing

### Requirement: Publisher Interface Layering
The publisher interfaces SHALL be strictly nested: `ServicePublisher` for service messages and
events, `ThingPublisher` adding thing messages and thing events, and `Publisher` adding adapter
messages. A `ThingFactory` SHALL receive the full `Publisher` and SHALL hand each constructed
service only the narrower `ServicePublisher` it needs.

#### Scenario: Service constructed
- **WHEN** a thing factory builds a service
- **THEN** the service is given a `ServicePublisher` and has no way to publish an adapter-level message

### Requirement: Service Event Enrichment At Publish Time
`NewServiceEvent` SHALL capture only the FIMP event type and the `hasChanged` flag; the domain,
class, address and service name SHALL be filled in by the publisher when the event is published.
The publisher SHALL set the domain to `adapter_service`, the class to the service name, and the
address to the service topic. A service event SHALL therefore be published through
`Service.PublishEvent`, since an event that never passed through the publisher carries no embedded
`event.Event` at all.

#### Scenario: Level report published
- **WHEN** an out level switch service publishes a level event for `evt.lvl.report`
- **THEN** subscribers receive an event with domain `adapter_service`, class `out_lvl_switch`, the service topic as address and the event type `evt.lvl.report`

#### Scenario: Change-only listener
- **WHEN** a listener subscribes with `WaitForChange`
- **THEN** it receives only service events whose `HasChanged` is true

### Requirement: Service Registry Lookup
A `ServiceRegistry` SHALL return every service across all things when `Services` is called with an
empty name, and only the matching ones otherwise. `ServiceByTopic` SHALL return nil rather than an
error when no service is registered for the topic. `IsRegistryInitialized` SHALL expose the
registry's `IsInitialized` as a task voter, so periodic service tasks do not run before things have
been loaded from the persisted state.

#### Scenario: Task gated on initialization
- **WHEN** a service task voted by `IsRegistryInitialized` fires before `InitializeThings` has succeeded
- **THEN** the task handler is not executed

### Requirement: Skipping Tasks For Disconnected Things
`ShouldSkipServiceTask` SHALL return true only when the service registry is also a `ThingRegistry`,
a thing is found for the service topic, and that thing's connectivity report has `ConnStatus` `DOWN`.
In every other case — a registry that is not a thing registry, or a topic that resolves to no thing —
it SHALL return false, so an unknown topology never silently disables reporting.

#### Scenario: Thing reported down
- **WHEN** a periodic service task consults `ShouldSkipServiceTask` for a service of a thing whose connector reports `DOWN`
- **THEN** the task is skipped for that service

#### Scenario: Topic resolves to no thing
- **WHEN** the registry cannot resolve the service topic to a thing
- **THEN** the task is not skipped
