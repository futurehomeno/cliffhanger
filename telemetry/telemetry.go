package telemetry

import (
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/cliffhanger/telemetry/types"
)

// Telemetry emits telemetry events over MQTT to the cloud backend-service.
type Telemetry interface {
	// Report publishes an optional event. Dropped silently when telemetry
	// is globally disabled or when the domain is in SuppressedDomains.
	emit(event, domain string, data map[string]any) error
	// ReportRequired publishes a critical event. Dropped only when
	// telemetry is globally disabled.
	emitRequired(event, domain string, data map[string]any) error
	// SetEvtTopic overrides the default target topic.
	// Passing an empty string restores the default. The override is
	// not persisted and resets to the default on restart.
	SetEvtTopic(topic string)

	// Enable toggles the global enabled flag. Enable(true) re-stamps
	// EnabledAt and starts the validity timer. Enable(false) cancels it.
	Enable(enabled bool) error
	// IsEnabled reports the current global enabled flag.
	IsEnabled() bool
	// Validity returns the auto-disable window.
	Validity() time.Duration
	// SetValidity updates the validity window. Must be positive.
	SetValidity(validity time.Duration) error

	// SetSuppressedDomains replaces the list of domains for which
	// Report/Emit are dropped. ReportRequired/EmitRequired still publish.
	// An empty or nil list clears all suppressions.
	SetSuppressedDomains(domains []string) error
	// SuppressedDomains returns a copy of the current suppressed-domains list.
	SuppressedDomains() []string

	// Stop tears down the embedded cloud config-pull and cancels the
	// validity-expiry timer. After Stop the reporter still answers gates
	// and getters, but no further cloud config will be applied. Safe to
	// call multiple times.
	Stop()
}

// New returns a Telemetry that publishes as the given source and embeds
// a cloud config-pull that auto-starts. The store is the single source
// of truth for the persisted telemetry config.
func New(mqtt *fimpgo.MqttTransport, source string, cfg *types.TelemetryConfig, saveConfig func() error) (Telemetry, error) {
	if mqtt == nil {
		return nil, errors.New("telemetry: mqtt transport is nil")
	}

	if source == "" {
		return nil, errors.New("telemetry: source is not set")
	}

	if cfg == nil {
		return nil, errors.New("telemetry: store is required")
	}

	t := &telemetryT{
		mqtt:          mqtt,
		sourceRn:      fimptype.ResourceNameT(source),
		cfg:           cfg,
		topic:         DefaultTelemetryEvtTopic,
		saveConfigPtr: saveConfig,
	}

	if err := t.resumeValidityWindow(); err != nil {
		return nil, err
	}

	cp := newConfigPull(mqtt, t.sourceRn, t)
	if err := cp.start(); err != nil {
		t.stopValidityTimer()

		return nil, err
	}

	t.pull = cp

	return t, nil
}

// Emit calls Report and logs on error. Nil-safe: returns immediately
// if tel is nil.
func Emit(tel Telemetry, domain, event string, data map[string]any) {
	if tel == nil {
		return
	}

	if err := tel.emit(event, domain, data); err != nil {
		log.WithError(err).Warnf("[cliff] Telemetry: %q", event)
	}
}

func EmitRequired(tel Telemetry, domain, event string, data map[string]any) {
	if tel == nil {
		log.Warnf("[cliff] Telemetry: dropping required event %q (reporter is nil)", event)

		return
	}

	if err := tel.emitRequired(event, domain, data); err != nil {
		log.WithError(err).Warnf("[cliff] Telemetry: %q", event)
	}
}

// telemetryT serializes Telemetry-state access under one mutex. The
// persisted state lives entirely in store; the cached fields are the
// constants set at construction (mqtt, sourceRn, store) plus the runtime
// topic override, the validity-expiry timer, and the embedded pull.
type telemetryT struct {
	mqtt          *fimpgo.MqttTransport
	sourceRn      fimptype.ResourceNameT
	cfg           *types.TelemetryConfig
	saveConfigPtr func() error

	lock  sync.Mutex
	topic string
	timer *time.Timer

	pull *configPull
}

func (ptr *telemetryT) Stop() {
	if ptr.pull != nil {
		ptr.pull.stop()
	}

	ptr.stopValidityTimer()
}

// stopValidityTimer cancels the auto-disable timer started by Enable
// or resumeValidityWindow. Safe to call when no timer is running.
func (ptr *telemetryT) stopValidityTimer() {
	ptr.lock.Lock()
	ptr.stopTimerLocked()
	ptr.lock.Unlock()
}

// validityOrDefault returns the configured validity, falling back to the
// package default when unset or non-positive.
func validityOrDefault(c *types.TelemetryConfig) time.Duration {
	if c != nil && c.Validity > 0 {
		return c.Validity
	}

	return types.DefaultTelemetryValidity
}

func (ptr *telemetryT) emit(event, domain string, data map[string]any) error {
	if !ptr.cfg.Enabled {
		return nil
	}

	if slices.Contains(ptr.cfg.SuppressedDomains, domain) {
		return nil
	}

	return ptr.publish(ptr.topic, event, domain, data)
}

func (ptr *telemetryT) emitRequired(event, domain string, data map[string]any) error {
	if ptr.cfg.Enabled {
		return nil
	}

	return ptr.publish(ptr.topic, event, domain, data)
}

func (ptr *telemetryT) saveConfig() {
	if err := ptr.saveConfigPtr(); err != nil {
		log.Errorf("[cliff] Telemetry save config error: %v", err)
	}
}

func (ptr *telemetryT) publish(topic, event, domain string, data map[string]any) error {
	if event == "" {
		return errors.New("telemetry: event name is required")
	}

	msg := fimpgo.NewObjectMessage(MessageType, Service, &Event{
		Event:  event,
		Domain: domain,
		Data:   data,
	}, nil, nil, nil)
	msg.Source = ptr.sourceRn

	if err := ptr.mqtt.PublishToTopic(topic, msg); err != nil {
		return fmt.Errorf("telemetry: publish event: %w", err)
	}

	return nil
}

func (ptr *telemetryT) SetEvtTopic(topic string) {
	if topic == "" {
		topic = DefaultTelemetryEvtTopic
	}

	ptr.lock.Lock()
	ptr.topic = topic
	ptr.lock.Unlock()
}

func (ptr *telemetryT) Enable(enabled bool) error {
	ptr.lock.Lock()
	defer ptr.lock.Unlock()

	ptr.cfg.Enabled = enabled

	if enabled {
		ptr.cfg.EnabledAt = time.Now()
	} else {
		ptr.cfg.EnabledAt = time.Time{}
	}

	ptr.saveConfig()
	ptr.stopTimerLocked()

	if enabled {
		ptr.startTimerLocked(validityOrDefault(ptr.cfg))
	}

	return nil
}

func (ptr *telemetryT) IsEnabled() bool {
	ptr.lock.Lock()
	defer ptr.lock.Unlock()
	return ptr.cfg.Enabled
}

func (ptr *telemetryT) Validity() time.Duration {
	ptr.lock.Lock()
	defer ptr.lock.Unlock()
	return ptr.cfg.Validity
}

func (ptr *telemetryT) SetValidity(validity time.Duration) error {
	if validity <= 0 {
		return errors.New("telemetry: validity must be positive")
	}

	ptr.lock.Lock()
	defer ptr.lock.Unlock()

	var elapsed time.Duration

	shouldDisable := false

	if ptr.cfg.Enabled && !ptr.cfg.EnabledAt.IsZero() {
		elapsed = time.Since(ptr.cfg.EnabledAt)
		if elapsed < 0 {
			elapsed = 0
		}

		if elapsed >= validity {
			shouldDisable = true
		}
	}

	ptr.cfg.Validity = validity

	if shouldDisable {
		ptr.cfg.Enabled = false
		ptr.cfg.EnabledAt = time.Time{}
	}

	ptr.cfg.Validity = validity
	ptr.stopTimerLocked()

	switch {
	case shouldDisable:
		log.Infof("[cliff] Telemetry disabled: validity reduced below elapsed time")
	case ptr.cfg.Enabled && !ptr.cfg.EnabledAt.IsZero():
		ptr.startTimerLocked(validity - elapsed)
	}

	return nil
}

func (ptr *telemetryT) SetSuppressedDomains(domains []string) error {
	ptr.lock.Lock()
	defer ptr.lock.Unlock()

	if len(domains) == 0 {
		ptr.cfg.SuppressedDomains = nil
	} else {
		ptr.cfg.SuppressedDomains = slices.Clone(domains)
	}

	ptr.saveConfig()

	return nil
}

func (ptr *telemetryT) SuppressedDomains() []string {
	ptr.lock.Lock()
	defer ptr.lock.Unlock()

	src := ptr.cfg.SuppressedDomains
	if src == nil {
		return nil
	}

	return slices.Clone(src)
}

func (ptr *telemetryT) resumeValidityWindow() error {
	ptr.lock.Lock()
	defer ptr.lock.Unlock()

	if !ptr.cfg.Enabled {
		return nil
	}

	validity := validityOrDefault(ptr.cfg)
	now := time.Now()
	enabledAt := ptr.cfg.EnabledAt

	switch {
	case enabledAt.IsZero():
		enabledAt = now
	case enabledAt.After(now):
		// Clock skew: future timestamp - normalize to now.
		enabledAt = now
	}

	if !ptr.cfg.EnabledAt.Equal(enabledAt) {
		ptr.cfg.EnabledAt = enabledAt

		ptr.saveConfig()
	}

	elapsed := now.Sub(enabledAt)
	if elapsed >= validity {
		ptr.cfg.Enabled = false
		ptr.cfg.EnabledAt = time.Time{}

		ptr.saveConfig()

		log.Infof("[cliff] Telemetry disabled: validity expired before startup")

		return nil
	}

	// Hold the lock so ptr.timer is assigned before the AfterFunc callback
	// can acquire it - prevents a race on tiny durations.
	ptr.startTimerLocked(validity - elapsed)

	log.Infof("[cliff] Telemetry enabled (source=%s, validity=%s)", ptr.sourceRn, validity)

	return nil
}

// startTimerLocked must be called with ptr.lock held, or before the reporter
// has been published to other goroutines (e.g. from inside New).
func (ptr *telemetryT) startTimerLocked(d time.Duration) {
	var t *time.Timer

	t = time.AfterFunc(d, func() {
		ptr.lock.Lock()
		defer ptr.lock.Unlock()

		// Guard against a stale callback: if stopTimerLocked replaced
		// ptr.timer since this AfterFunc was scheduled, bail out.
		if ptr.timer != t {
			return
		}

		ptr.disableLocked("validity expired")
	})
	ptr.timer = t
}

func (ptr *telemetryT) stopTimerLocked() {
	if ptr.timer != nil {
		ptr.timer.Stop()
		ptr.timer = nil
	}
}

// disableLocked disables telemetry in the store. Errors are logged and
// swallowed: this path runs from the timer goroutine where there is no
// caller to surface the error to.
func (ptr *telemetryT) disableLocked(reason string) {
	ptr.timer = nil

	ptr.cfg.Enabled = false
	ptr.cfg.EnabledAt = time.Time{}

	ptr.saveConfig()

	log.Infof("[cliff] Telemetry disabled: %s", reason)
}
