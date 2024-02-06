package adapter

import "github.com/futurehomeno/cliffhanger/event"

// Connector represents a service responsible for thing connection management.
type Connector interface {
	// Connectivity returns a connectivity report for the thing.
	Connectivity() *ConnectivityDetails
	// Ping executes a ping and returns a ping details report for the thing.
	// This method must never use cached values and should always execute a real ping.
	Ping() *PingDetails
}

// ControllableConnector is a connector that can be used to control the connection status of a thing.
type ControllableConnector interface {
	Connector

	// Connect ensures that a thing is connected to the source of its data.
	// If the thing is already connected, this method does nothing.
	// Implementation might be empty if polling is the only strategy used by the adapter.
	Connect(t Thing)
	// Disconnect ensures that a thing is disconnected from the source of its data.
	// If the thing is already disconnected, this method does nothing.
	// Implementation might be empty if polling is the only strategy used by the adapter.
	Disconnect(t Thing)
}

// ConnectionStatus represents a connection status of a thing.
type ConnectionStatus string

// Constants representing available connection statuses.
const (
	ConnectionStatusUp   ConnectionStatus = "UP"
	ConnectionStatusDown ConnectionStatus = "DOWN"
)

// Operationability represents a current operationability status of a thing.
type Operationability string

// Constants representing available operationability statuses.
const (
	OperationabilitySleep     Operationability = "sleep"
	OperationabilityDiscovery Operationability = "discovery"
	OperationabilityUpdate    Operationability = "update"
)

// ConnectionQuality represents a connection quality of a thing.
type ConnectionQuality string

// Constants representing available connection qualities.
const (
	ConnectionQualityHigh      ConnectionQuality = "high"
	ConnectionQualityMedium    ConnectionQuality = "medium"
	ConnectionQualityLow       ConnectionQuality = "low"
	ConnectionQualityUndefined ConnectionQuality = "undefined"
)

// ConnectionType represents a connection type between an adapter and a thing.
type ConnectionType string

// Constants representing available connection types.
const (
	ConnectionTypeDirect   ConnectionType = "direct"
	ConnectionTypeIndirect ConnectionType = "indirect"
	ConnectionTypeUnknown  ConnectionType = "unknown"
)

// ConnectivityReports represents a set of connectivity reports of multiple things.
type ConnectivityReports []*ConnectivityReport

// ConnectivityReport represents a connectivity report of a thing.
type ConnectivityReport struct {
	Address        string `json:"address"`
	Hash           string `json:"hash"`
	Alias          string `json:"alias"`
	PowerSource    string `json:"power_source"`
	WakeupInterval string `json:"wakeup_interval"`
	CommTechnology string `json:"comm_tech"`

	*ConnectivityDetails
}

// sanitize sanitizes the connectivity report.
func (c *ConnectivityReport) sanitize() {
	if c.ConnectionQuality == "" {
		c.ConnectionQuality = ConnectionQualityUndefined
	}

	if c.ConnectionType == "" {
		c.ConnectionType = ConnectionTypeUnknown
	}

	if c.Operationability == nil {
		c.Operationability = make([]Operationability, 0)
	}
}

// ConnectivityDetails represents connectivity details of a thing.
type ConnectivityDetails struct {
	ConnectionStatus  ConnectionStatus   `json:"status"`
	Operationability  []Operationability `json:"operationability"`
	ConnectionQuality ConnectionQuality  `json:"conn_quality"`
	ConnectionType    ConnectionType     `json:"conn_type"`
}

// PingResult represents a result of a ping.
type PingResult string

// Constants representing available ping results.
const (
	PingResultSuccess PingResult = "SUCCESS"
	PingResultFailed  PingResult = "FAILED"
)

// PingReport represents a ping report from a thing.
type PingReport struct {
	Address string `json:"address"`
	Delay   int    `json:"delay"`

	*PingDetails
}

// PingDetails represents ping details from a thing.
type PingDetails struct {
	Status PingResult       `json:"status"`
	Nodes  []ConnectionNode `json:"nodes,omitempty"`
}

// ConnectionNode represents a node in a connection graph.
// Useful only for networks allowing mesh structure or with a clearly distinguishable nodes.
type ConnectionNode struct {
	Address string `json:"address"`
	Type    string `json:"type"`
	Value   string `json:"value"`
}

// ConnectivityEvent represents a connectivity event.
type ConnectivityEvent struct {
	ThingEvent

	Connectivity *ConnectivityDetails
}

func newConnectivityEvent(t Thing, c *ConnectivityDetails) *ConnectivityEvent {
	return &ConnectivityEvent{
		ThingEvent:   NewThingEvent(t.Address(), nil),
		Connectivity: c,
	}
}

func WaitForConnectivityEvent() event.Filter {
	return event.WaitFor[*ConnectivityEvent]()
}
