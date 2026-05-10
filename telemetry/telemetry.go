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

	"github.com/futurehomeno/cliffhanger/config"
	"github.com/futurehomeno/cliffhanger/storage"
	"github.com/futurehomeno/cliffhanger/telemetry/config_pull"
	"github.com/futurehomeno/cliffhanger/telemetry/types"
)

// Telemetry emits telemetry events over MQTT to the cloud backend-service.
type Telemetry interface {
	emit(event, domain string, data map[string]any) error
	emitRequired(event, domain string, data map[string]any) error
	SetEvtTopic(topic string)
	Enable(enabled bool) error
	IsEnabled() bool
	Validity() time.Duration
	SetValidity(validity time.Duration) error
	SetSuppressedDomains(domains []string) error
	SuppressedDomains() []string
}

func Emit(tel Telemetry, domain, event string, data map[string]any) {
	if tel == nil {
		return
	}

	if err := tel.emit(event, domain, data); err != nil {
		log.WithError(err).Warnf("[cliff] Emit event= %q", event)
	}
}

func EmitRequired(tel Telemetry, domain, event string, data map[string]any) {
	if tel == nil {
		log.Warnf("[cliff] Telemetry: dropping required event %q (reporter is nil)", event)

		return
	}

	if err := tel.emitRequired(event, domain, data); err != nil {
		log.WithError(err).Warnf("[cliff] Emit required event=%q", event)
	}
}

func New(mqtt *fimpgo.MqttTransport, source string, store storage.Storage[*config.Default]) (Telemetry, error) {
	if mqtt == nil {
		return nil, errors.New("telemetry: mqtt transport is nil")
	}

	if source == "" {
		return nil, errors.New("telemetry: source is not set")
	}

	if store == nil {
		return nil, errors.New("telemetry: store is required")
	}

	// Establish the cfgLocked invariant: from this point on the store
	// must always have a non-nil telemetry block.
	if store.Model().Telemetry == nil {
		store.Model().Telemetry = &types.TelemetryConfig{Enabled: true, EnabledAt: time.Now()}
	}

	t := &telemetryT{
		mqtt:     mqtt,
		sourceRn: fimptype.ResourceNameT(source),
		store:    store,
		topic:    defaultTelemetryEvtTopic,
	}

	if err := t.resumeValidityWindow(); err != nil {
		return nil, err
	}

	cp := config_pull.New(mqtt, t.sourceRn, t.applyConfigFromCloud)
	if err := cp.Start(); err != nil {
		t.stopValidityTimer()

		return nil, err
	}

	t.pullCfg = cp

	return t, nil
}

type telemetryT struct {
	mqtt     *fimpgo.MqttTransport
	sourceRn fimptype.ResourceNameT
	store    storage.Storage[*config.Default]

	lock  sync.Mutex
	topic string
	timer *time.Timer

	pullCfg *config_pull.Config
}

func (ptr *telemetryT) Stop() {
	if ptr.pullCfg != nil {
		ptr.pullCfg.Stop()
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
	cfg := ptr.store.Model().Telemetry

	if cfg == nil || !cfg.Enabled {
		return nil
	}

	if slices.Contains(cfg.SuppressedDomains, domain) {
		return nil
	}

	return ptr.publish(ptr.topic, event, domain, data)
}

func (ptr *telemetryT) emitRequired(event, domain string, data map[string]any) error {
	if ptr.config() != nil && ptr.config().Enabled {
		return nil
	}

	return ptr.publish(ptr.topic, event, domain, data)
}

func (ptr *telemetryT) config() *types.TelemetryConfig {
	if ptr.store == nil || ptr.store.Model() == nil || ptr.store.Model().Telemetry == nil {
		return &types.TelemetryConfig{}
	}

	return ptr.store.Model().Telemetry
}

func (ptr *telemetryT) saveConfig() {
	if err := ptr.store.Save(); err != nil {
		log.Errorf("[cliff] Telemetry save config error: %v", err)
	}
}

func (ptr *telemetryT) publish(topic, event, domain string, data map[string]any) error {
	if event == "" {
		return errors.New("telemetry: event name is required")
	}

	msg := fimpgo.NewObjectMessage(MessageType, fimptype.ServiceNameT(ptr.sourceRn), &Event{
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
		topic = defaultTelemetryEvtTopic
	}

	ptr.lock.Lock()
	ptr.topic = topic
	ptr.lock.Unlock()
}

func (ptr *telemetryT) Enable(enabled bool) error {
	ptr.lock.Lock()
	defer ptr.lock.Unlock()

	cfg := ptr.config()
	cfg.Enabled = enabled

	if enabled {
		cfg.EnabledAt = time.Now()
	} else {
		cfg.EnabledAt = time.Time{}
	}

	ptr.saveConfig()
	ptr.stopTimerLocked()

	if enabled {
		ptr.startTimerLocked(validityOrDefault(cfg))
	}

	return nil
}

func (ptr *telemetryT) IsEnabled() bool {
	ptr.lock.Lock()
	defer ptr.lock.Unlock()
	return ptr.config().Enabled
}

func (ptr *telemetryT) Validity() time.Duration {
	ptr.lock.Lock()
	defer ptr.lock.Unlock()
	return ptr.config().Validity
}

func (ptr *telemetryT) SetValidity(validity time.Duration) error {
	if validity <= 0 {
		return errors.New("telemetry: validity must be positive")
	}

	ptr.lock.Lock()
	defer ptr.lock.Unlock()

	var elapsed time.Duration

	shouldDisable := false

	if ptr.config().Enabled && !ptr.config().EnabledAt.IsZero() {
		elapsed = time.Since(ptr.config().EnabledAt)
		if elapsed < 0 {
			elapsed = 0
		}

		if elapsed >= validity {
			shouldDisable = true
		}
	}

	ptr.config().Validity = validity

	if shouldDisable {
		ptr.config().Enabled = false
		ptr.config().EnabledAt = time.Time{}
	}

	ptr.store.Model().Telemetry.Validity = validity
	ptr.stopTimerLocked()

	switch {
	case shouldDisable:
		log.Infof("[cliff] Telemetry disabled: validity reduced below elapsed time")
	case ptr.config().Enabled && !ptr.config().EnabledAt.IsZero():
		ptr.startTimerLocked(validity - elapsed)
	}

	return nil
}

func (ptr *telemetryT) SetSuppressedDomains(domains []string) error {
	ptr.lock.Lock()
	defer ptr.lock.Unlock()

	if len(domains) == 0 {
		ptr.config().SuppressedDomains = nil
	} else {
		ptr.config().SuppressedDomains = slices.Clone(domains)
	}

	ptr.saveConfig()

	return nil
}

func (ptr *telemetryT) SuppressedDomains() []string {
	ptr.lock.Lock()
	defer ptr.lock.Unlock()

	src := ptr.config().SuppressedDomains
	if src == nil {
		return nil
	}

	return slices.Clone(src)
}

func (ptr *telemetryT) resumeValidityWindow() error {
	ptr.lock.Lock()
	defer ptr.lock.Unlock()

	cfg := ptr.config()
	if !cfg.Enabled {
		return nil
	}

	validity := validityOrDefault(cfg)
	now := time.Now()
	enabledAt := cfg.EnabledAt

	switch {
	case enabledAt.IsZero():
		enabledAt = now
	case enabledAt.After(now):
		// Clock skew: future timestamp - normalize to now.
		enabledAt = now
	}

	if !cfg.EnabledAt.Equal(enabledAt) {
		newCfg := *ptr.config()
		newCfg.EnabledAt = enabledAt

		ptr.saveConfig()
	}

	elapsed := now.Sub(enabledAt)
	if elapsed >= validity {
		newCfg := *ptr.config()
		newCfg.Enabled = false
		newCfg.EnabledAt = time.Time{}

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

	cfg := *ptr.config()
	cfg.Enabled = false
	cfg.EnabledAt = time.Time{}

	ptr.saveConfig()

	log.Infof("[cliff] Telemetry disabled: %s", reason)
}

func (ptr *telemetryT) applyConfigFromCloud(enabled bool, suppressed []string) {
	if err := ptr.Enable(enabled); err != nil {
		log.Errorf("[cliff] Telemetry enable=%v err: %v", enabled, err)
	}

	if err := ptr.SetSuppressedDomains(suppressed); err != nil {
		log.Errorf("[cliff] Telemetry set suppressed domains=%v err: %v", suppressed, err)
	}
}
