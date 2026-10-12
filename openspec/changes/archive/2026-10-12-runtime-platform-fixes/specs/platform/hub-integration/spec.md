## ADDED Requirements

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
