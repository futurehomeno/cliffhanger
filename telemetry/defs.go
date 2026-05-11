package telemetry

const (
	telemetryInterface      = "evt.telemetry.report"
	telemetryReportEvtTopic = "pt:j1/mt:evt/rt:cloud/rn:telemetry/ad:report"
)

// Event is the payload carried in the FIMP val field.
type Event struct {
	Event  string         `json:"event"`
	Domain string         `json:"domain,omitempty"`
	Data   map[string]any `json:"data,omitempty"`
}
