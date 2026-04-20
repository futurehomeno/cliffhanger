package telemetry

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
	log "github.com/sirupsen/logrus"

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
	// SettingValidity is the config parameter name used by the FIMP
	// cmd.config.set_telemetry_validity / cmd.config.get_telemetry_validity
	// commands. Once the window elapses since the last Enable(true),
	// the reporter auto-disables.
	SettingValidity = "telemetry_validity"

	// DefaultValidity is the default window telemetry stays enabled after
	// Enable(true). After that it auto-disables on the next Report call.
	DefaultValidity = 30 * 24 * time.Hour
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
	// Enable toggles reporting. When true, resets the validity window
	// starting from now. When false, subsequent Report calls become silent
	// no-ops and return nil.
	Enable(enabled bool) error
	// IsEnabled reports whether telemetry is currently publishing.
	IsEnabled() bool
	// Validity returns the window telemetry stays enabled after Enable(true)
	// before it auto-disables.
	Validity() time.Duration
	// SetValidity updates the validity window. Must be positive.
	SetValidity(validity time.Duration) error
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

	r := &reporter{
		mqtt:      mqtt,
		source:    fimptype.ResourceNameT(source),
		topic:     defaultTelemetryEvtTopic,
		enabled:   true,
		validity:  DefaultValidity,
		enabledAt: time.Now(),
	}

	r.startTimerLocked(DefaultValidity)

	return r, nil
}

type reporter struct {
	mqtt   *fimpgo.MqttTransport
	source fimptype.ResourceNameT

	mu        sync.Mutex
	topic     string
	enabled   bool
	validity  time.Duration
	enabledAt time.Time
	timer     *time.Timer
}

func (r *reporter) Report(event, domain string, data map[string]any) error {
	if event == "" {
		return errors.New("telemetry event name is required")
	}

	r.mu.Lock()
	topic := r.topic
	enabled := r.enabled
	r.mu.Unlock()

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
	defer r.mu.Unlock()

	r.enabled = enabled
	r.stopTimerLocked()

	if enabled {
		r.enabledAt = time.Now()
		r.startTimerLocked(r.validity)
	} else {
		r.enabledAt = time.Time{}
	}

	return nil
}

func (r *reporter) IsEnabled() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.enabled
}

func (r *reporter) Validity() time.Duration {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.validity
}

func (r *reporter) SetValidity(validity time.Duration) error {
	if validity <= 0 {
		return errors.New("telemetry: validity must be positive")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.validity = validity

	if r.timer == nil || r.enabledAt.IsZero() {
		return nil
	}

	elapsed := time.Since(r.enabledAt)
	r.stopTimerLocked()

	if elapsed >= validity {
		r.disableLocked("validity reduced below elapsed time")

		return nil
	}

	r.startTimerLocked(validity - elapsed)

	return nil
}

func (r *reporter) startTimerLocked(d time.Duration) {
	var t *time.Timer
	t = time.AfterFunc(d, func() {
		r.mu.Lock()
		defer r.mu.Unlock()

		if r.timer != t {
			return
		}

		r.disableLocked("validity expired")
	})
	r.timer = t
}

func (r *reporter) stopTimerLocked() {
	if r.timer != nil {
		r.timer.Stop()
		r.timer = nil
	}
}

func (r *reporter) disableLocked(reason string) {
	r.enabled = false
	r.enabledAt = time.Time{}
	r.timer = nil

	log.Infof("[cliff] Telemetry disabled: %s", reason)
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

// RouteCmdGetValidity returns a routing for cmd.config.get_telemetry_validity
// that replies with the current validity window.
func RouteCmdGetValidity(serviceName fimptype.ServiceNameT, reporter Reporter, options ...config.RoutingOption) *router.Routing {
	return config.RouteCmdConfigGetDuration(serviceName, SettingValidity, reporter.Validity, options...)
}

// RouteCmdSetValidity returns a routing for cmd.config.set_telemetry_validity
// that updates the validity window.
func RouteCmdSetValidity(serviceName fimptype.ServiceNameT, reporter Reporter, options ...config.RoutingOption) *router.Routing {
	return config.RouteCmdConfigSetDuration(serviceName, SettingValidity, reporter.SetValidity, options...)
}

// RoutingForReporter returns the get/set routings for the telemetry config
// parameters (enabled, validity) bound to the given Reporter.
func RoutingForReporter(serviceName fimptype.ServiceNameT, reporter Reporter, options ...config.RoutingOption) []*router.Routing {
	return []*router.Routing{
		RouteCmdGetEnabled(serviceName, reporter, options...),
		RouteCmdSetEnabled(serviceName, reporter, options...),
		RouteCmdGetValidity(serviceName, reporter, options...),
		RouteCmdSetValidity(serviceName, reporter, options...),
	}
}
