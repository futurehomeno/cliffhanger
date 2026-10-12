# Hub Integration Specification

## Purpose
Defines how an adapter learns facts about the hub it runs on and how it reads the hub's own device
graph. It covers `hub.Info` file resolution, the beta/prod `Environment` switch, hub token retrieval
from Cloud Bridge over FIMP, the `prime` client for the hub's Vinculum API, and the
`prime/observer` cached projection kept current by Vinculum notifications. Adapters rely on these
contracts to resolve rooms, areas and foreign devices without polling Vinculum on every read.
## Requirements
### Requirement: Hub Info File Resolution
`hub.LoadInfo(path)` SHALL read a JSON `Info` document from the hub filesystem. An empty `path` SHALL
default to the deprecated v1 file `/var/lib/futurehome/hub/hub.json`. When the resolved path is that
v1 file, the loader SHALL prefer `/var/lib/futurehome/hub/hub_v2.json` if that file exists, is not a
directory, and either the v1 file cannot be stat'ed or the v2 file's modification time is later. If
reading the chosen file fails, the loader SHALL retry once: with `hub_v2.json` when the chosen path
was anything other than `hub_v2.json` (including a caller-supplied custom path), and with `hub.json`
otherwise, returning an error naming the retried path only if that read also fails. A malformed
document SHALL be returned as an error, never as a partially populated `Info`.

#### Scenario: v2 file is newer
- **WHEN** `LoadInfo("")` runs on a hub where both files exist and `hub_v2.json` has the later
  modification time
- **THEN** `hub_v2.json` is parsed and its `hub_id`, `site_id` and `environment` are returned

#### Scenario: chosen file is unreadable
- **WHEN** the selected path cannot be read
- **THEN** the other well-known path is read instead
- **AND** an error naming the second path is returned only if that read also fails

#### Scenario: malformed JSON
- **WHEN** the file content is not valid JSON for `Info`
- **THEN** an unmarshal error is returned and no `Info` value is produced

### Requirement: Hub Environment Switch
`hub.Environment` SHALL carry the hub's deployment environment, with `hub.EnvBeta` (`"beta"`) and
`hub.EnvProd` (`"prod"`) as the defined values. Every environment-dependent endpoint SHALL treat
only the exact value `"beta"` as beta and any other value — including the empty string of a hub file
that omits the key — as production. `auth.ProxyURL` SHALL return
`https://partners-beta.futurehome.io` for beta and `https://partners.futurehome.io` otherwise, and
`auth.ProxyCallbackURL` SHALL return
`https://app-static-beta.futurehome.io/playground_oauth_callback` for beta and
`https://app-static.futurehome.io/playground_oauth_callback` otherwise.

#### Scenario: environment missing from the hub file
- **WHEN** the loaded `Info` has an empty `Environment`
- **THEN** the production OAuth2 proxy and callback URLs are used

### Requirement: Hub Token Retrieval
`hub.LoadToken(resourceName)` SHALL obtain the hub token from the Cloud Bridge service over FIMP,
using its own short-lived MQTT transport that it SHALL stop before returning. The default broker
SHALL be `tcp://localhost:1883` and the MQTT client ID prefix SHALL be
`<resourceName>_hub_token_loader`. The loader SHALL send `cmd.clbridge.get_auth_token` to
`pt:j1/mt:cmd/rt:app/rn:clbridge/ad:1` with the response topic
`pt:j1/mt:rsp/rt:app/rn:<resourceName>/ad:1` and a 5 second per-request timeout, retrying up to 7
times with 30 seconds between attempts. A response whose interface is not
`evt.clbridge.auth_token_report` SHALL be rejected as an error rather than parsed.

#### Scenario: Cloud Bridge unresponsive at boot
- **WHEN** the request times out on every attempt
- **THEN** 7 attempts are made, roughly 30 seconds apart, before an error is returned
- **AND** the MQTT transport and sync client are stopped regardless of the outcome

#### Scenario: wrong response type
- **WHEN** Cloud Bridge answers with an interface other than `evt.clbridge.auth_token_report`
- **THEN** an error naming both the expected and received interface is returned

### Requirement: Prime Request Addressing
`prime.NewClient(syncClient, resourceName, timeout)` SHALL send every request as a
`cmd.pd7.request` object message on the `vinculum` service to
`pt:j1/mt:cmd/rt:app/rn:vinculum/ad:1`, with `ResponseToTopic` and the awaited response topic both
set to `pt:j1/mt:rsp/rt:app/rn:<resourceName>/ad:1`, and with the message source set to the caller's
resource name. `prime.NewCloudClient(syncClient, cloudServiceName, siteUUID, timeout)` SHALL keep
the same Vinculum request address but prefix both addresses with the site UUID and expect the
response on `rt:cloud/rn:backend-service/ad:<cloudServiceName>`. The supplied timeout SHALL be
truncated to whole seconds, because the FIMP sync client takes a second-granularity timeout — a
timeout below one second therefore becomes zero.

#### Scenario: sub-second timeout
- **WHEN** a client is constructed with a 500 ms default timeout
- **THEN** requests are issued with a timeout of 0 seconds

#### Scenario: cloud client response routing
- **WHEN** a cloud client is constructed for site `abc` and service name `my-service`
- **THEN** the request carries the site prefix and asks Vinculum to reply on the
  `rt:cloud/rn:backend-service/ad:my-service` topic

### Requirement: Prime Component Queries
Each getter (`GetDevices`, `GetThings`, `GetRooms`, `GetAreas`, `GetHouse`, `GetHub`,
`GetShortcuts`, `GetModes`, `GetTimers`, `GetVinculumServices`, `State`) SHALL request exactly its
own component and SHALL return a nil result with a nil error when the response carries no `param`
entry for that component — an absent component is not an error. `GetComponents` SHALL reject any
component name outside the known set (`device`, `thing`, `room`, `area`, `house`, `hub`, `shortcut`,
`mode`, `timer`, `service`, `state`) before sending anything, and `GetAll` SHALL request all of
them. A transport failure or an unparsable envelope SHALL be returned as an error naming the
requested components.

#### Scenario: component absent from the response
- **WHEN** Vinculum answers a `room` request with a payload containing no `room` key
- **THEN** `GetRooms` returns `nil, nil`

#### Scenario: unknown component requested
- **WHEN** `GetComponents("device", "widget")` is called
- **THEN** an error is returned and no message is published

### Requirement: Prime Commands
`RunShortcut(shortcutID)` and `ChangeMode(mode)` SHALL send a `set` request naming the `shortcut`
respectively `mode` component with the given value as `id`, and SHALL return the raw Vinculum
`Response`. The client SHALL NOT translate a response carrying `success: false` or a non-empty
`errors` field into a Go error; the caller inspects the returned `Response`. Only transport failures
and unparsable envelopes SHALL produce an error.

#### Scenario: hub rejects the shortcut
- **WHEN** Vinculum replies with `success: false` for `RunShortcut(3)`
- **THEN** the response is returned with a nil error and `Success` false

### Requirement: Observer Component Scope
`observer.New(client, eventManager, refreshInterval, components...)` SHALL accept only `device`,
`thing`, `room` and `area`, returning an error for any other component. The observer SHALL fetch
exactly the components it was constructed with, and `Update` SHALL ignore a notification for a
component it does not observe, returning nil.

#### Scenario: unsupported component
- **WHEN** the observer is constructed with `house`
- **THEN** construction fails with an error naming that component

#### Scenario: notification for an unobserved component
- **WHEN** a `mode` notification reaches an observer built for `device` only
- **THEN** it is ignored and no event is published

### Requirement: Observer Staleness And Single-Flight Refresh
Every observer getter SHALL call `Refresh(false)` first and SHALL serve the cached projection
without contacting Vinculum when the cache is fresh. A refresh SHALL be required when no successful
load has happened yet or when more than `refreshInterval` has elapsed since the last successful one.
The fetch SHALL be serialised by a dedicated refresh lock and re-checked after acquiring it, so
concurrent stale readers issue one Vinculum request between them rather than one each. The fetch
SHALL run outside the data lock, so it never stalls readers or the notification stream for its
duration. A caller that waited on a fetch requested after its own call and that failed SHALL be
handed that error rather than issuing a duplicate request; callers arriving later SHALL take their
own attempt. `Refresh(true)` SHALL always fetch.

#### Scenario: two stale readers race
- **WHEN** two goroutines call `GetDevices` on a stale observer at the same time
- **THEN** exactly one `GetComponents` request is sent to Vinculum and both receive the result

#### Scenario: refresh fails
- **WHEN** the Vinculum request returns an error
- **THEN** the getter returns the zero value and the wrapped error, and the previously cached
  projection is left untouched

### Requirement: Observer Notification Application
`Update` SHALL apply an `add`, `edit` or `delete` notification to the cached set under the write
lock and SHALL publish a `ComponentEvent` (domain `prime`, class `observer`) carrying the component,
command and ID. `add` SHALL be ignored when the ID is already present, `edit` SHALL append when the
ID is absent, and `delete` SHALL remove by ID. A notification the observer fails to apply SHALL mark
the cache unrefreshed, so the next read forces a full reload rather than serving a set known to have
missed an update.

#### Scenario: duplicate add
- **WHEN** an `add` notification arrives for a device ID already in the cache
- **THEN** the cache is left unchanged and the entry is not duplicated

#### Scenario: unparsable notification payload
- **WHEN** the payload of an `edit` notification cannot be unmarshalled
- **THEN** an error is returned and the next getter call triggers a full refresh

### Requirement: Observer Refresh Must Not Undo Notifications
When a fetch completes, the observer SHALL install the fetched snapshot only if no notification was
applied while the fetch was in flight, or if no successful load has happened yet, or if a failed
update already requested a resync. Otherwise the notification-updated set SHALL be kept and
reconciliation SHALL be left to the next interval, because the snapshot was requested before those
notifications and installing it would silently undo them. A successful refresh SHALL publish a
`RefreshEvent` naming the observed components, and SHALL publish it outside the data lock so a
subscriber may read the observer back without deadlocking.

#### Scenario: notification lands during the fetch
- **WHEN** a device `delete` is applied while `GetComponents` is still in flight
- **THEN** the fetched snapshot is discarded, the updated set is kept, and the deleted device does
  not reappear

### Requirement: Observer Read Isolation And Wiring
Observer getters SHALL return a copy of the cached slice, so a caller iterating the result is not
affected by concurrent notifications; the pointed-to components themselves are shared and MUST NOT
be mutated by callers. `observer.Route` SHALL subscribe the observer to `evt.pd7.notify` messages on
the `vinculum` service at topic `pt:j1/mt:evt/rt:app/rn:vinculum/ad:1` with silent errors, so a
malformed notification does not produce an error reply on the bus. `observer.Task` SHALL run a
non-forced `Refresh` on the given interval, logging failures. `WaitForDeviceChange` — and the
matching thing/room/area filters — SHALL match both a `ComponentEvent` for `add`, `edit` or `delete`
of that component and a `RefreshEvent` that included it.

#### Scenario: refresh satisfies a device-change waiter
- **WHEN** a full refresh of an observer built for `device` completes
- **THEN** `WaitForDeviceChange` matches the published `RefreshEvent`

#### Scenario: malformed notification on the bus
- **WHEN** an `evt.pd7.notify` message cannot be parsed
- **THEN** the failure is logged by the router without publishing an error response

### Requirement: Configurable Token Loader
`hub.NewTokenLoader(cfg)` SHALL build the loader behind `hub.LoadToken` from a `TokenLoaderConfig`
carrying the resource name, MQTT broker URI, username, password, client ID prefix, retry count and
retry delay. A zero broker URI, retry count or retry delay SHALL take the `LoadToken` defaults, an
empty client ID prefix SHALL default to the resource name, and `_hub_token_loader` SHALL be appended
to whichever prefix results. The request SHALL carry the resource name as its source, and a report
whose value is not a string SHALL be returned as an error.

#### Scenario: custom broker
- **WHEN** `NewTokenLoader` is given `MQTTServerURI` `tcp://10.0.0.2:1883` and prefix `probe`
- **THEN** the loader connects to that broker with client ID prefix `probe_hub_token_loader`, still
  retrying 7 times 30 seconds apart

### Requirement: Prime Notification Parsing
`prime.NotifyFromMessage` SHALL decode an `evt.pd7.notify` payload into a `Notify`, returning an
error when the payload is not an object. `Notify.ParseIntegerID` SHALL accept an ID carried as an
integer, a JSON number or a decimal string, and SHALL return an error for a missing ID or any other
type. Each typed getter (`GetDevice`, `GetThing`, `GetRoom`, `GetArea`, `GetHouse`, `GetHub`,
`GetShortcut`, `GetMode`, `GetTimer`, `GetService`) SHALL return `nil, nil` when the notification is
for a different component, and an error when the `param` does not unmarshal into that component.
`GetHubMode` SHALL return a value only for a `hub` notification whose ID is `"mode"`.

#### Scenario: delete notification with a string id
- **WHEN** a `delete` notification carries `"id": "42"`
- **THEN** `ParseIntegerID` returns 42

#### Scenario: getter for another component
- **WHEN** `GetDevice` is called on a `room` notification
- **THEN** it returns `nil, nil` without inspecting the payload

### Requirement: Prime Device And State Helpers
`Device.GetName` SHALL prefer the user-assigned client name, then `modelAlias`, then `model`.
`GetThingID`, `GetRoomID` and `Room.GetAreaID` SHALL return 0 when the reference is unset, and
`Devices.FilterByThingID(0)` SHALL return nil rather than every unassigned device.
`Devices.FilterByIDs` SHALL return each matching device once, in fleet order, however often its ID
is repeated. The typed service-property getters SHALL return the zero value when the service or
property is absent or does not convert to the requested type, and `HasInterfaces` SHALL require
every listed interface. For Vinculum `state`, `StateService.FindAttribute` SHALL accept either a bare
attribute name or a three-segment interface name (`evt.<attribute>.report`) and match on its middle
segment; `StateAttribute.FindValue` SHALL return the first value whose properties contain every
requested key/value pair; and the `StateDevice.GetAttribute*Value` helpers SHALL return the zero
value and a zero time when the device, service, attribute or value is missing or the value does not
convert, and the converted value with a zero time when only its timestamp does not parse. These lookups SHALL be safe to call on a nil receiver.

#### Scenario: attribute looked up by interface name
- **WHEN** `FindAttribute("evt.meter.report")` is called on a service whose attribute is `meter`
- **THEN** that attribute is returned

#### Scenario: value with an unparsable timestamp
- **WHEN** a matching attribute value has a `ts` that cannot be parsed
- **THEN** `GetAttributeFloatValue` returns the value with a zero time, so the caller can tell the
  reading's age is unknown

### Requirement: Prime Device Topic Matching
`Device.MatchesTopic` SHALL return true only when the topic ends with the address of one of the
device's services, and SHALL ignore services with an empty address. `Devices.FindByTopic` SHALL return
the first device that matches.

#### Scenario: address that is a prefix of another
- **WHEN** devices at `.../sv:out_bin_switch/ad:1` and `.../sv:out_bin_switch/ad:12_0` exist and
  `FindByTopic` gets a topic ending in `.../sv:out_bin_switch/ad:12_0`
- **THEN** the `ad:12_0` device is returned

#### Scenario: empty service address
- **WHEN** a device's only service has an empty address
- **THEN** it matches no topic

