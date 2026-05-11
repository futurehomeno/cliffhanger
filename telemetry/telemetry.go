package telemetry

import (
	"errors"
	"fmt"
	"runtime/debug"
	"slices"
	"sync"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/cliffhanger/config"
	"github.com/futurehomeno/cliffhanger/telemetry/config_poll"
	"github.com/futurehomeno/cliffhanger/telemetry/types"
)

const defaultTelemetryValidity = 30 * 24 * time.Hour

type Telemetry interface {
	emit(domain, event string, data map[string]any) error
	emitOnChange(domain, event string, data map[string]any, interval time.Duration) error
	SetEvtTopic(topic string)
	Enable(enabled bool) error
	IsEnabled() bool
	Validity() time.Duration
	SetValidity(validity time.Duration) error
	SetSuppressed(suppressed map[string]types.SuppressedEntry) error
	Suppressed() map[string]types.SuppressedEntry
	ServiceName() fimptype.ServiceNameT
}

func Emit(tel Telemetry, domain, event string, data map[string]any) {
	if tel == nil {
		return
	}

	if err := tel.emit(domain, event, data); err != nil {
		log.WithError(err).Warnf("[cliff] Emit event= %q", event)
	}
}

func EmitOnChange(tel Telemetry, domain, event string, data map[string]any, interval time.Duration) {
	if tel == nil {
		return
	}

	if err := tel.emitOnChange(domain, event, data, interval); err != nil {
		log.WithError(err).Warnf("[cliff] EmitOnChange event=%q", event)
	}
}

// EmitRebootMilestone emits a DomainReboot/EventRebootMilestone event when
// count is a positive multiple of restartMilestoneStep, so callers can call
// it on every boot and only milestone boots reach the pipeline.
func EmitRebootMilestone(tel Telemetry, count int) {
	if count <= 0 || count%restartMilestoneStep != 0 {
		return
	}

	Emit(tel, DomainReboot, EventRebootMilestone, map[string]any{"count": count})
}

func RecoverAndEmit(tel Telemetry, name string, terminate bool) {
	r := recover()
	if r == nil {
		return
	}

	log.Errorf("[cliff] Panic in %s:\n%s", name, string(debug.Stack()))

	Emit(tel, DomainPanic, name, map[string]any{"terminate": terminate})

	if terminate {
		panic(r)
	}

	log.Error(r)
}

func New(mqtt *fimpgo.MqttTransport, sourceRn fimptype.ResourceNameT, store *config.DefaultStore) (Telemetry, error) {
	if mqtt == nil {
		return nil, errors.New("telemetry: mqtt transport is nil")
	}

	if sourceRn == "" {
		return nil, errors.New("telemetry: source is not set")
	}

	if store == nil {
		return nil, errors.New("telemetry: store is required")
	}

	if _, err := store.Telemetry(); err != nil {
		if err := store.SetTelemetry(&types.TelemetryConfig{Enabled: true, EnabledAt: time.Now()}); err != nil {
			return nil, fmt.Errorf("telemetry: seed config: %w", err)
		}
	}

	t := &telemetryT{
		mqtt:     mqtt,
		sourceRn: sourceRn,
		store:    store,
		topic:    telemetryReportEvtTopic,
	}

	if err := t.resumeValidityWindow(); err != nil {
		return nil, err
	}

	cp := config_poll.New(mqtt, t.sourceRn, t.applyConfigFromCloud)
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

	lock           sync.Mutex
	topic          string
	timer          *time.Timer
	emitTimestamps map[string]time.Time

	pullCfg *config_poll.Config
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

func (ptr *telemetryT) stopValidityTimer() {
	ptr.lock.Lock()
	ptr.stopTimerLocked()
	ptr.lock.Unlock()
}

func validityOrDefault(c *types.TelemetryConfig) time.Duration {
	if c != nil && c.Validity > 0 {
		return c.Validity
	}

	return defaultTelemetryValidity
}

func (ptr *telemetryT) emit(domain, event string, data map[string]any) error {
	cfg := ptr.config()
	if !cfg.Enabled {
		return nil
	}

	if s := cfg.Suppressed; s != nil {
		if len(s.Domains) == 0 && len(s.Events) == 0 {
			return nil
		}

		if slices.Contains(s.Domains, domain) || slices.Contains(s.Events, event) {
			return nil
		}
	}

	return ptr.publish(ptr.evtTopic(), domain, event, data)
}

func (ptr *telemetryT) emitOnChange(domain, event string, data map[string]any, interval time.Duration) error {
	key := domain + "/" + event

	ptr.lock.Lock()

	if ptr.emitTimestamps == nil {
		ptr.emitTimestamps = make(map[string]time.Time)
	}

	last := ptr.emitTimestamps[key]
	throttled := !last.IsZero() && time.Since(last) < interval

	if !throttled {
		ptr.emitTimestamps[key] = time.Now()
	}

	ptr.lock.Unlock()

	if throttled {
		return nil
	}

	return ptr.emit(domain, event, data)
}

func (ptr *telemetryT) evtTopic() string {
	ptr.lock.Lock()
	defer ptr.lock.Unlock()

	return ptr.topic
}

func (ptr *telemetryT) config() types.TelemetryConfig {
	if ptr.store == nil {
		return types.TelemetryConfig{}
	}

	snap, err := ptr.store.Telemetry()
	if err != nil {
		return types.TelemetryConfig{}
	}

	if snap.Suppressed != nil {
		e := *snap.Suppressed
		e.Domains = slices.Clone(e.Domains)
		e.Events = slices.Clone(e.Events)
		snap.Suppressed = &e
	}

	return snap
}

func (ptr *telemetryT) publish(topic, domain, event string, data map[string]any) error {
	if event == "" {
		return errors.New("telemetry: event name is required")
	}

	msg := fimpgo.NewObjectMessage(telemetryInterface, fimptype.ServiceNameT(ptr.sourceRn), &Event{
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
		topic = telemetryReportEvtTopic
	}

	ptr.lock.Lock()
	ptr.topic = topic
	ptr.lock.Unlock()
}

func (ptr *telemetryT) Enable(enabled bool) error {
	ptr.lock.Lock()
	defer ptr.lock.Unlock()

	next := ptr.config()
	next.Enabled = enabled

	if enabled {
		next.EnabledAt = time.Now()
	} else {
		next.EnabledAt = time.Time{}
	}

	if err := ptr.store.SetTelemetry(&next); err != nil {
		return fmt.Errorf("telemetry: persist enable=%v: %w", enabled, err)
	}

	ptr.stopTimerLocked()

	if enabled {
		ptr.startTimerLocked(validityOrDefault(&next))
	}

	return nil
}

func (ptr *telemetryT) IsEnabled() bool {
	return ptr.config().Enabled
}

func (ptr *telemetryT) Validity() time.Duration {
	cfg := ptr.config()

	return validityOrDefault(&cfg)
}

func (ptr *telemetryT) SetValidity(validity time.Duration) error {
	if validity <= 0 {
		return errors.New("telemetry: validity must be positive")
	}

	ptr.lock.Lock()
	defer ptr.lock.Unlock()

	next := ptr.config()

	var (
		elapsed       time.Duration
		shouldDisable bool
	)

	if next.Enabled && !next.EnabledAt.IsZero() {
		elapsed = max(time.Since(next.EnabledAt), 0)
		if elapsed >= validity {
			shouldDisable = true
		}
	}

	next.Validity = validity

	if shouldDisable {
		next.Enabled = false
		next.EnabledAt = time.Time{}
	}

	if err := ptr.store.SetTelemetry(&next); err != nil {
		return fmt.Errorf("telemetry: persist validity: %w", err)
	}

	ptr.stopTimerLocked()

	switch {
	case shouldDisable:
		log.Infof("[cliff] Telemetry validity ended: validity=%s elapsed=%s", validity, elapsed)
	case next.Enabled && !next.EnabledAt.IsZero():
		ptr.startTimerLocked(validity - elapsed)
	}

	return nil
}

func (ptr *telemetryT) SetSuppressed(suppressed map[string]types.SuppressedEntry) error {
	ptr.lock.Lock()
	defer ptr.lock.Unlock()

	next := ptr.config()

	entry, ok := suppressed[string(ptr.sourceRn)]

	switch {
	case !ok:
		// dont suppress anything
		next.Suppressed = nil
	case len(entry.Domains) == 0 && len(entry.Events) == 0:
		// suppresses the whole app
		next.Suppressed = &types.SuppressedEntry{}
	default:
		// clean all supressions rules
		next.Suppressed = &types.SuppressedEntry{
			Domains: slices.Clone(entry.Domains),
			Events:  slices.Clone(entry.Events),
		}
	}

	if err := ptr.store.SetTelemetry(&next); err != nil {
		return fmt.Errorf("telemetry: persist suppressed: %w", err)
	}

	return nil
}

func (ptr *telemetryT) Suppressed() map[string]types.SuppressedEntry {
	s := ptr.config().Suppressed
	if s == nil {
		return map[string]types.SuppressedEntry{}
	}

	return map[string]types.SuppressedEntry{
		string(ptr.sourceRn): {
			Domains: slices.Clone(s.Domains),
			Events:  slices.Clone(s.Events),
		},
	}
}

func (ptr *telemetryT) resumeValidityWindow() error {
	ptr.lock.Lock()
	defer ptr.lock.Unlock()

	next := ptr.config()
	if !next.Enabled {
		return nil
	}

	validity := validityOrDefault(&next)
	now := time.Now()
	enabledAt := next.EnabledAt

	switch {
	case enabledAt.IsZero():
		enabledAt = now
	case enabledAt.After(now):
		enabledAt = now
	}

	dirty := !next.EnabledAt.Equal(enabledAt)
	next.EnabledAt = enabledAt

	elapsed := now.Sub(enabledAt)
	if elapsed >= validity {
		next.Enabled = false
		next.EnabledAt = time.Time{}
		dirty = true
	}

	if dirty {
		if err := ptr.store.SetTelemetry(&next); err != nil {
			log.WithError(err).Errorf("[cliff] Telemetry: persist resume")
		}
	}

	if !next.Enabled {
		log.Infof("[cliff] Telemetry disabled: validity expired before startup")

		return nil
	}

	ptr.startTimerLocked(validity - elapsed)

	log.Infof("[cliff] Telemetry enabled (source=%s, validity=%s)", ptr.sourceRn, validity)

	return nil
}

func (ptr *telemetryT) startTimerLocked(d time.Duration) {
	var t *time.Timer

	t = time.AfterFunc(d, func() {
		ptr.lock.Lock()
		defer ptr.lock.Unlock()

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

func (ptr *telemetryT) disableLocked(reason string) {
	ptr.timer = nil

	next := ptr.config()
	next.Enabled = false
	next.EnabledAt = time.Time{}

	if err := ptr.store.SetTelemetry(&next); err != nil {
		log.WithError(err).Errorf("[cliff] Telemetry: persist disable")
	}

	log.Infof("[cliff] Telemetry disabled: %s", reason)
}

func (ptr *telemetryT) applyConfigFromCloud(enabled bool, suppressed map[string]types.SuppressedEntry) {
	if err := ptr.Enable(enabled); err != nil {
		log.Errorf("[cliff] Telemetry enable=%v err: %v", enabled, err)
	}

	if err := ptr.SetSuppressed(suppressed); err != nil {
		log.Errorf("[cliff] Telemetry set suppressed err: %v", err)
	}
}
