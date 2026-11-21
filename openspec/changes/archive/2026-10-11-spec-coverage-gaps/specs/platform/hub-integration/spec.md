## MODIFIED Requirements

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

## ADDED Requirements

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
