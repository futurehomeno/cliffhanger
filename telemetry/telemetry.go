package telemetry

import (
	"errors"
	"fmt"
	"sync"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/config"
	"github.com/futurehomeno/cliffhanger/router"
)

const (
	// Topic is the default FIMP topic used by the cloud telemetry pipeline.
	// Picking mt:rsp is deliberate: it matches the existing CloudBridge
	// LocalToCloud default route so no bridge change is needed.
	defaultTelemetryEvtTopic = "pt:j1/mt:rsp/rt:cloud/rn:backend-service/ad:telemetry"
	evtTelemetryReport       = "evt.telemetry.report"
	serviceName              = "telemetry"

	// SettingEnabled is the config parameter name used by the FIMP
	// cmd.config.set_telemetry_enabled / cmd.config.get_telemetry_enabled
	// commands produced by RoutingForReporter.
	SettingEnabled = "telemetry_enabled"
)

// Event is the payload carried in the FIMP val field.
type Event struct {
	Event  string         `json:"event"`
	Domain string         `json:"domain,omitempty"`
	Data   map[string]any `json:"data,omitempty"`
}

// Reporter emits telemetry events over MQTT to the cloud backend-service.
type Reporter interface {
	// Report publishes an event with the given name, optional domain, and
	// free-form data payload. Returns nil without publishing when disabled.
	Report(event, domain string, data map[string]any) error
	// SetTargetTopic overrides the default target topic.
	// Passing an empty string restores the default.
	SetTargetTopic(topic string)
	// Enable toggles reporting. When false, subsequent Report calls become
	// silent no-ops and return nil.
	Enable(enabled bool) error
	// IsEnabled reports whether telemetry is currently publishing.
	IsEnabled() bool
}

// New returns a Reporter that publishes telemetry events as the given source.
// The source becomes the FIMP src field and is used by the consumer to populate
// the app column, so it must uniquely identify the emitting application.
func New(mqtt *fimpgo.MqttTransport, source string) (Reporter, error) {
	if mqtt == nil {
		return nil, errors.New("mqtt transport is nil")
	}

	if source == "" {
		return nil, errors.New("source is not set")
	}

	return &reporter{
		mqtt:    mqtt,
		source:  fimptype.ResourceNameT(source),
		topic:   defaultTelemetryEvtTopic,
		enabled: true,
	}, nil
}

type reporter struct {
	mqtt   *fimpgo.MqttTransport
	source fimptype.ResourceNameT

	mu      sync.RWMutex
	topic   string
	enabled bool
}

func (r *reporter) Report(event, domain string, data map[string]any) error {
	if event == "" {
		return errors.New("telemetry event name is required")
	}

	r.mu.RLock()
	topic := r.topic
	enabled := r.enabled
	r.mu.RUnlock()

	if !enabled {
		return nil
	}

	msg := fimpgo.NewObjectMessage(evtTelemetryReport, serviceName, event, nil, nil, nil)
	msg.Source = r.source

	if err := r.mqtt.PublishToTopic(topic, msg); err != nil {
		return fmt.Errorf("publish telemetry event err: %w", err)
	}

	return nil
}

func (r *reporter) SetTargetTopic(topic string) {
	if topic == "" {
		topic = defaultTelemetryEvtTopic
	}

	r.mu.Lock()
	r.topic = topic
	r.mu.Unlock()
}

func (r *reporter) Enable(enabled bool) error {
	r.mu.Lock()
	r.enabled = enabled
	r.mu.Unlock()

	return nil
}

func (r *reporter) IsEnabled() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.enabled
}

// RouteCmdGetEnabled returns a routing for cmd.config.get_telemetry_enabled
// that replies with the current Reporter enabled state.
func RouteCmdGetEnabled(serviceName fimptype.ServiceNameT, reporter Reporter, options ...config.RoutingOption) *router.Routing {
	return config.RouteCmdConfigGetBool(serviceName, SettingEnabled, reporter.IsEnabled, options...)
}

// RouteCmdSetEnabled returns a routing for cmd.config.set_telemetry_enabled
// that toggles the Reporter enabled state.
func RouteCmdSetEnabled(serviceName fimptype.ServiceNameT, reporter Reporter, options ...config.RoutingOption) *router.Routing {
	return config.RouteCmdConfigSetBool(serviceName, SettingEnabled, reporter.Enable, options...)
}

// RoutingForReporter returns the get/set routings for the telemetry enabled
// config parameter bound to the given Reporter.
func RoutingForReporter(serviceName fimptype.ServiceNameT, reporter Reporter, options ...config.RoutingOption) []*router.Routing {
	return []*router.Routing{
		RouteCmdGetEnabled(serviceName, reporter, options...),
		RouteCmdSetEnabled(serviceName, reporter, options...),
	}
}
