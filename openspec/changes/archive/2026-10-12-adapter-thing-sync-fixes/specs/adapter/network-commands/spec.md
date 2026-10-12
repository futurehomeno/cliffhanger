## MODIFIED Requirements

### Requirement: All Nodes Query
On `cmd.network.get_all_nodes` the router SHALL call `Adapter.SendConnectivityReport`, which
publishes a single `evt.network.all_nodes_report` object message on the adapter topic carrying the
array of connectivity reports for all registered things. The reports SHALL be collected without
consulting any reporting strategy or cache, and the handler SHALL return no direct reply. The value
SHALL always be a JSON array, never `null`.

#### Scenario: No things registered
- **WHEN** `cmd.network.get_all_nodes` is received by an adapter with no registered things
- **THEN** an `evt.network.all_nodes_report` is still published, carrying the empty array `[]`
