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
	"github.com/futurehomeno/cliffhanger/telemetry/config_pull"
	"github.com/futurehomeno/cliffhanger/telemetry/types"
)

// defaultTelemetryValidity is the default window telemetry stays enabled
// after Enable(true) before it auto-disables.
const defaultTelemetryValidity = 30 * 24 * time.Hour

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
	ServiceName() fimptype.ServiceNameT
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
		log.Warn("[cliff] Telemetry is nil in EmitRequired")
		return
	}

	if err := tel.emitRequired(event, domain, data); err != nil {
		log.WithError(err).Warnf("[cliff] Emit required event=%q", event)
	}
}

func New(mqtt *fimpgo.MqttTransport, source string, store *config.DefaultStore) (Telemetry, error) {
	if mqtt == nil {
		return nil, errors.New("telemetry: mqtt transport is nil")
	}

	if source == "" {
		return nil, errors.New("telemetry: source is not set")
	}

	if store == nil {
		return nil, errors.New("telemetry: store is required")
	}

	if store.Telemetry() == nil {
		if err := store.SetTelemetry(&types.TelemetryConfig{Enabled: true, EnabledAt: time.Now()}); err != nil {
			return nil, fmt.Errorf("telemetry: seed config: %w", err)
		}
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
	store    *config.DefaultStore

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

func (ptr *telemetryT) ServiceName() fimptype.ServiceNameT {
	return fimptype.ServiceNameT(ptr.sourceRn)
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

	return defaultTelemetryValidity
}

func (ptr *telemetryT) emit(event, domain string, data map[string]any) error {
	ptr.lock.Lock()
	cfg := ptr.config()
	enabled := cfg.Enabled
	suppressed := slices.Contains(cfg.SuppressedDomains, domain)
	topic := ptr.topic
	ptr.lock.Unlock()

	if !enabled || suppressed {
		return nil
	}

	return ptr.publish(topic, event, domain, data)
}

func (ptr *telemetryT) emitRequired(event, domain string, data map[string]any) error {
	ptr.lock.Lock()
	enabled := ptr.config().Enabled
	topic := ptr.topic
	ptr.lock.Unlock()

	if !enabled {
		return nil
	}

	return ptr.publish(topic, event, domain, data)
}

func (ptr *telemetryT) config() *types.TelemetryConfig {
	if ptr.store == nil {
		return &types.TelemetryConfig{}
	}

	cfg := ptr.store.Telemetry()
	if cfg == nil {
		return &types.TelemetryConfig{}
	}

	return cfg
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

	ptr.saveConfig()
	ptr.stopTimerLocked()

	switch {
	case shouldDisable:
		log.Infof("[cliff] Telemetry valididty ended: validity=%s elapsed=%s", validity, elapsed)
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
		ptr.config().EnabledAt = enabledAt

		ptr.saveConfig()
	}

	elapsed := now.Sub(enabledAt)
	if elapsed >= validity {
		ptr.config().Enabled = false
		ptr.config().EnabledAt = time.Time{}

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

	cfg := ptr.config()
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
