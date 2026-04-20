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
	Topic = "pt:j1/mt:rsp/rt:cloud/rn:backend-service/ad:telemetry"
	// MessageType is the FIMP type used for telemetry events.
	MessageType = "evt.telemetry.report"
	// Service is the FIMP serv field used for telemetry events.
	Service fimptype.ServiceNameT = "telemetry"

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

// Store persists telemetry configuration so the enabled flag, the timestamp
// the reporter was last enabled at, and the validity window all survive
// application restarts. Consumer applications implement this against their
// own configuration storage.
type Store interface {
	Enabled() bool
	SetEnabled(enabled bool) error
	EnabledAt() time.Time
	SetEnabledAt(t time.Time) error
	Validity() time.Duration
	SetValidity(validity time.Duration) error
}

// NewDefaultStore adapts a config.Default-backed persistence layer to the
// Store interface. The accessor must return a pointer to the embedded Default
// block; save persists any field mutation to disk.
func NewDefaultStore(accessor func() *config.Default, save func() error) Store {
	return &defaultStore{accessor: accessor, save: save}
}

type defaultStore struct {
	accessor func() *config.Default
	save     func() error
}

func (s *defaultStore) Enabled() bool {
	v := s.accessor().TelemetryEnabled
	if v == nil {
		return true
	}

	return *v
}

func (s *defaultStore) SetEnabled(enabled bool) error {
	v := enabled
	s.accessor().TelemetryEnabled = &v

	return s.save()
}

func (s *defaultStore) EnabledAt() time.Time {
	raw := s.accessor().TelemetryEnabledAt
	if raw == "" {
		return time.Time{}
	}

	t, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return time.Time{}
	}

	return t
}

func (s *defaultStore) SetEnabledAt(t time.Time) error {
	if t.IsZero() {
		s.accessor().TelemetryEnabledAt = ""
	} else {
		s.accessor().TelemetryEnabledAt = t.UTC().Format(time.RFC3339Nano)
	}

	return s.save()
}

func (s *defaultStore) Validity() time.Duration {
	raw := s.accessor().TelemetryValidity
	if raw == "" {
		return DefaultValidity
	}

	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		return DefaultValidity
	}

	return d
}

func (s *defaultStore) SetValidity(validity time.Duration) error {
	s.accessor().TelemetryValidity = validity.String()

	return s.save()
}

// NewMemoryStore returns an in-memory Store suitable for tests or
// applications that do not need telemetry state to survive restarts.
func NewMemoryStore() Store {
	return &memoryStore{enabled: true, validity: DefaultValidity}
}

type memoryStore struct {
	mu        sync.Mutex
	enabled   bool
	enabledAt time.Time
	validity  time.Duration
}

func (s *memoryStore) Enabled() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.enabled
}

func (s *memoryStore) SetEnabled(enabled bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.enabled = enabled

	return nil
}

func (s *memoryStore) EnabledAt() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.enabledAt
}

func (s *memoryStore) SetEnabledAt(t time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.enabledAt = t

	return nil
}

func (s *memoryStore) Validity() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.validity
}

func (s *memoryStore) SetValidity(validity time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.validity = validity

	return nil
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
// the app column, so it must uniquely identify the emitting application. The
// store is required and persists enabled / validity across restarts.
func New(mqtt *fimpgo.MqttTransport, source string, store Store) (Reporter, error) {
	if mqtt == nil {
		return nil, errors.New("mqtt transport is nil")
	}

	if source == "" {
		return nil, errors.New("source is not set")
	}

	if store == nil {
		return nil, errors.New("store is required")
	}

	validity := store.Validity()
	if validity <= 0 {
		validity = DefaultValidity
	}

	r := &reporter{
		mqtt:     mqtt,
		source:   fimptype.ResourceNameT(source),
		store:    store,
		topic:    Topic,
		enabled:  store.Enabled(),
		validity: validity,
	}

	if r.enabled {
		enabledAt := store.EnabledAt()
		if enabledAt.IsZero() {
			enabledAt = time.Now()
			if err := store.SetEnabledAt(enabledAt); err != nil {
				return nil, fmt.Errorf("telemetry: persist enabled_at: %w", err)
			}
		}

		elapsed := time.Since(enabledAt)
		if elapsed >= validity {
			r.enabled = false

			if err := store.SetEnabled(false); err != nil {
				return nil, fmt.Errorf("telemetry: persist enabled: %w", err)
			}

			if err := store.SetEnabledAt(time.Time{}); err != nil {
				return nil, fmt.Errorf("telemetry: clear enabled_at: %w", err)
			}

			log.Infof("[cliff] Telemetry disabled: validity expired before startup")
		} else {
			r.enabledAt = enabledAt
			r.startTimerLocked(validity - elapsed)
		}
	}

	return r, nil
}

type reporter struct {
	mqtt   *fimpgo.MqttTransport
	source fimptype.ResourceNameT
	store  Store

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
		topic = Topic
	}

	r.mu.Lock()
	r.topic = topic
	r.mu.Unlock()
}

func (r *reporter) Enable(enabled bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := r.store.SetEnabled(enabled); err != nil {
		return fmt.Errorf("telemetry: persist enabled: %w", err)
	}

	r.enabled = enabled
	r.stopTimerLocked()

	if enabled {
		r.enabledAt = time.Now()

		if err := r.store.SetEnabledAt(r.enabledAt); err != nil {
			return fmt.Errorf("telemetry: persist enabled_at: %w", err)
		}

		r.startTimerLocked(r.validity)
	} else {
		r.enabledAt = time.Time{}

		if err := r.store.SetEnabledAt(time.Time{}); err != nil {
			return fmt.Errorf("telemetry: clear enabled_at: %w", err)
		}
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

	if err := r.store.SetValidity(validity); err != nil {
		return fmt.Errorf("telemetry: persist validity: %w", err)
	}

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

	if err := r.store.SetEnabled(false); err != nil {
		log.WithError(err).Errorf("[cliff] Telemetry: failed to persist disabled state")
	}

	if err := r.store.SetEnabledAt(time.Time{}); err != nil {
		log.WithError(err).Errorf("[cliff] Telemetry: failed to clear enabled_at")
	}

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
