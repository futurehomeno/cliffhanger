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

type ConnStatusT string

const (
	ConnStatusUp   ConnStatusT = "UP"
	ConnStatusDown ConnStatusT = "DOWN"
)

type OperationabilityT string

const (
	OperationabilitySleep     OperationabilityT = "sleep"
	OperationabilityDiscovery OperationabilityT = "discovery"
	OperationabilityBroken    OperationabilityT = "broken"
	OperationabilityNotReady  OperationabilityT = "not_ready"
	OperationabilityReady     OperationabilityT = "ready"
	OperationabilityRemoved   OperationabilityT = "removed"
	OperationabilityLeft      OperationabilityT = "left"
	OperationabilityUpdate    OperationabilityT = "update"
	OperationabilityFailed    OperationabilityT = "failed"
)

type ConnQualityT string

const (
	ConnQualityHigh      ConnQualityT = "high"
	ConnQualityMedium    ConnQualityT = "medium"
	ConnQualityLow       ConnQualityT = "low"
	ConnQualityUndefined ConnQualityT = "undefined"

	ConnQualityVeryStrong ConnQualityT = "very_strong"
	ConnQualityStrong     ConnQualityT = "strong"
	ConnQualityGood       ConnQualityT = "good"
	ConnQualityOK         ConnQualityT = "ok"
	ConnQualityPoor       ConnQualityT = "poor"
	ConnQualityVeryPoor   ConnQualityT = "very_poor"
	ConnQualityNoSignal   ConnQualityT = "no_signal"
)

type ConnTypeT string

const (
	ConnTypeDirect   ConnTypeT = "direct"
	ConnTypeIndirect ConnTypeT = "indirect"
	ConnTypeUnknown  ConnTypeT = "unknown"
)

type ConnectivityReports []*ConnectivityReport

type ConnectivityReport struct {
	Address        string `json:"address"`
	Hash           string `json:"hash"`
	Alias          string `json:"alias"`
	PowerSource    string `json:"power_source"`
	WakeupInterval string `json:"wakeup_interval"`
	CommTechnology string `json:"comm_tech"`

	*ConnectivityDetails
}

func (c *ConnectivityReport) sanitize() {
	if c.ConnQuality == "" {
		c.ConnQuality = ConnQualityUndefined
	}

	if c.ConnType == "" {
		c.ConnType = ConnTypeUnknown
	}

	if c.Operationability == nil {
		c.Operationability = make([]OperationabilityT, 0)
	}
}

// ConnectivityDetails represents connectivity details of a thing.
type ConnectivityDetails struct {
	ConnStatus       ConnStatusT         `json:"status"`
	Operationability []OperationabilityT `json:"operationability"`
	ConnQuality      ConnQualityT        `json:"conn_quality"`
	ConnType         ConnTypeT           `json:"conn_type"`
}

type PingResult string

const (
	PingResultSuccess PingResult = "SUCCESS"
	PingResultFailed  PingResult = "FAILED"
)

type PingReport struct {
	Address string `json:"address"`
	Delay   int    `json:"delay"`

	*PingDetails
}

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
