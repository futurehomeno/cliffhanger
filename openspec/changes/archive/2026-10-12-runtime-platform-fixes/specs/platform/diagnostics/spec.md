## MODIFIED Requirements

### Requirement: Timeline Entries
`notification.Timeline` SHALL publish `cmd.timeline.set` string-map messages to
`pt:j1/mt:cmd/rt:app/rn:time_owl/ad:1`. `Event` SHALL carry `EventName`, `MessageContent`,
`DeviceId`, `DeviceName`, `RoomName` and `AreaName`, with zero IDs rendered as an empty string, and
`Message` SHALL carry `sender`, `message_en` and `message_no` for a pre-translated entry. `Event`
SHALL send the service name `kind_owl` and `Message` the service name `time_owl`; the `kind_owl`
value is deliberate and matches what core-device-monitor sends for the same command.

#### Scenario: localized timeline message
- **WHEN** `Message` is called with English and Norwegian texts
- **THEN** a single `cmd.timeline.set` message carrying both translations and the sender is published

#### Scenario: timeline event service name
- **WHEN** `Event` is published
- **THEN** the `cmd.timeline.set` message carries the service name `kind_owl`
