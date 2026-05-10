package telemetry

import "github.com/futurehomeno/fimpgo/fimptype"

const (
	MessageType                                    = "evt.telemetry.report"
	defaultTelemetryEvtTopic                       = "pt:j1/mt:evt/rt:cloud/rn:backend-service/ad:telemetry"
	Service                  fimptype.ServiceNameT = "telemetry"
)

// Event is the payload carried in the FIMP val field.
type Event struct {
	Event  string         `json:"event"`
	Domain string         `json:"domain,omitempty"`
	Data   map[string]any `json:"data,omitempty"`
}
