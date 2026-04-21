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
	Topic = "pt:j1/mt:rsp/rt:cloud/rn:backend-service/ad:telemetry"
	// MessageType is the FIMP type used for telemetry events.
	MessageType = "evt.telemetry.report"
	// Service is the FIMP serv field used for telemetry events.
	Service fimptype.ServiceNameT = "telemetry"

	// SettingEnabled is the config parameter name used by the FIMP
	// cmd.config.set_telemetry_enabled / cmd.config.get_telemetry_enabled
	// commands produced by RoutingForTelemetry.
	SettingEnabled = "telemetry_enabled"
	// SettingValidity is the config parameter name used by the FIMP
	// cmd.config.set_telemetry_validity / cmd.config.get_telemetry_validity
	// commands. Once the window elapses since the last Enable(true),
	// the reporter auto-disables.
	SettingValidity = "telemetry_validity"

	// DefaultValidity is the default window telemetry stays enabled after
	// Enable(true). After that it auto-disables via a background timer.
	DefaultValidity = 30 * 24 * time.Hour
)

// Event is the payload carried in the FIMP val field.
type Event struct {
	Event  string         `json:"event"`
	Domain string         `json:"domain,omitempty"`
	Data   map[string]any `json:"data,omitempty"`
}
