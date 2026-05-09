// Package telemetry publishes FIMP telemetry events from cliffhanger apps to
// the cloud backend-service. See Telemetry for the main interface and New for
// the constructor.
package telemetry

import (
	"time"

	"github.com/futurehomeno/fimpgo/fimptype"
)

const (
	// Topic is the default FIMP topic used by the cloud telemetry pipeline.
	// Picking mt:rsp is deliberate: it matches the existing CloudBridge
	// LocalToCloud default route so no bridge change is needed.
	DefaultTelemetryEvtTopic = "pt:j1/mt:evt/rt:cloud/rn:backend-service/ad:telemetry"
	// MessageType is the FIMP type used for telemetry events.
	MessageType = "evt.telemetry.report"
	// Service is the FIMP serv field used for telemetry events.
	Service fimptype.ServiceNameT = "telemetry"

	// CmdGetConfig is the FIMP message type for requesting telemetry config
	// from the cloud.
	CmdGetConfig = "cmd.telemetry.get_config"
	// EvtConfigReport is the FIMP message type for telemetry config response
	// from the cloud.
	EvtConfigReport = "evt.telemetry.config_report"

	// ConfigRequestTopic is the MQTT topic for config requests to the cloud.
	// Uses mt:cmd/rt:cloud which matches the CloudBridge SDU/MDU
	// LocalToCloud default route (+/mt:cmd/rt:cloud/#).
	ConfigRequestTopic = "pt:j1/mt:cmd/rt:cloud/rn:telemetry/ad:config"

	// configResponseTopicFmt is the MQTT topic the cloud publishes config
	// responses to. Uses mt:evt/rt:cloud which matches the existing
	// CloudBridge CloudToLocal default route (+/mt:evt/rt:cloud/#).
	// The %s placeholder is the app's FIMP resource name (source).
	configResponseTopicFmt = "pt:j1/mt:evt/rt:cloud/rn:%s/ad:telemetry-config"

	// DefaultPollInterval is the fallback interval when the cloud response
	// does not include next_update or on error.
	DefaultPollInterval = 6 * time.Hour
	// MaxPollInterval caps the delay derived from the cloud next_update
	// field. Prevents a misconfigured response from silencing config
	// updates indefinitely.
	MaxPollInterval = 24 * time.Hour
)

// Event is the payload carried in the FIMP val field.
type Event struct {
	Event  string         `json:"event"`
	Domain string         `json:"domain,omitempty"`
	Data   map[string]any `json:"data,omitempty"`
}

// ConfigResponse is the payload of evt.telemetry.config_report from the cloud.
//
// Suppressed lists domain names for which Report/Emit are dropped on this
// app; ReportRequired/EmitRequired still publish for those domains.
type ConfigResponse struct {
	Enabled    bool     `json:"enabled"`
	Suppressed []string `json:"suppressed"`
	NextUpdate string   `json:"next_update"`
}
