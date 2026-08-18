package telemetry

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"runtime/debug"
	"slices"
	"strings"
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
	// Start begins polling the cloud for telemetry configuration.
	Start() error
	// Stop ends the configuration poll and the validity window timer.
	Stop() error
	emit(domain, event string, data map[string]any) error
	emitOnChange(domain, event string, data map[string]any, interval time.Duration) error
	emitIfMore(domain, event string, threshold int, reset bool, data map[string]any, interval time.Duration) error
	resetEventCounters(domain, event string, scope map[string]any) error
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
		log.Warnf("[cliff] Emit event=%q. err: %v", event, err)
	}
}

func EmitOnChange(tel Telemetry, domain, event string, data map[string]any, interval time.Duration) {
	if tel == nil {
		return
	}

	if err := tel.emitOnChange(domain, event, data, interval); err != nil {
		log.Warnf("[cliff] EmitOnChange event=%q. err: %v", event, err)
	}
}

func EmitIfMore(tel Telemetry, domain, event string, threshold int, reset bool, data map[string]any, interval time.Duration) {
	if tel == nil {
		return
	}

	if err := tel.emitIfMore(domain, event, threshold, reset, data, interval); err != nil {
		log.Warnf("[cliff] EmitIfMore event=%q. err: %v", event, err)
	}
}

// ResetEventCounters clears EmitIfMore counters for the given domain/event. A
// nil scope clears every counter for the event; a non-nil scope clears only the
// counters whose data contains all of its key/value pairs, so a success can
// reset just its own subject — e.g. {"device_id": 7} leaves other devices'
// counters intact.
func ResetEventCounters(tel Telemetry, domain, event string, scope map[string]any) {
	if tel == nil {
		return
	}

	if err := tel.resetEventCounters(domain, event, scope); err != nil {
		log.Warnf("[cliff] ResetEventCounters event=%q. err: %v", event, err)
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

	log.Errorf("[cliff] Panic in %s: %v\n%s", name, r, debug.Stack())

	Emit(tel, DomainPanic, name, map[string]any{"terminate": terminate})

	if terminate {
		panic(r)
	}
}

func New(mqtt *fimpgo.MqttTransport, sourceRn fimptype.ResourceNameT, store *config.DefaultStore, version string) (Telemetry, error) {
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
		version:  version,
		topic:    telemetryReportEvtTopic,
	}

	if err := t.resumeValidityWindow(); err != nil {
		return nil, err
	}

	t.pullCfg = config_poll.New(mqtt, t.sourceRn, t.applyConfigFromCloud)

	return t, nil
}

type telemetryT struct {
	mqtt     *fimpgo.MqttTransport
	sourceRn fimptype.ResourceNameT
	store    *config.DefaultStore
	version  string

	lock           sync.Mutex
	topic          string
	timer          *time.Timer
	emitTimestamps map[string]time.Time
	ifMoreStates   map[string]*ifMoreState

	pullCfg *config_poll.Config
}

type ifMoreState struct {
	count int
	last  time.Time
	data  map[string]any
}

// Start begins polling the cloud for telemetry configuration. Kept out of the constructor so that
// the goroutine, timers and MQTT subscription it owns are tied to the application lifecycle rather
// than to the lifetime of the process.
func (ptr *telemetryT) Start() error {
	// Re-armed on every start, not only in the constructor: Stop tears the timer down, so without
	// this a stop/start cycle in one process left telemetry enabled past its validity window until
	// a cloud config report happened to re-enable it.
	if err := ptr.resumeValidityWindow(); err != nil {
		return err
	}

	if ptr.pullCfg == nil {
		return nil
	}

	return ptr.pullCfg.Start()
}

func (ptr *telemetryT) Stop() error {
	if ptr.pullCfg != nil {
		ptr.pullCfg.Stop()
	}

	ptr.stopValidityTimer()

	return nil
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

func (ptr *telemetryT) emitIfMore(domain, event string, threshold int, reset bool, data map[string]any, interval time.Duration) error {
	if threshold < 1 {
		return nil
	}

	// \x00 separates the domain/event boundary from the data fingerprint so a
	// sibling event whose name shares this one's prefix can't collide with it.
	key := domain + "\x00" + event + "\x00" + dataFingerprint(data)

	ptr.lock.Lock()

	if ptr.ifMoreStates == nil {
		ptr.ifMoreStates = make(map[string]*ifMoreState)
	}

	st := ptr.ifMoreStates[key]
	if st == nil {
		st = &ifMoreState{data: maps.Clone(data)}
		ptr.ifMoreStates[key] = st
	}

	st.count++

	reached := st.count >= threshold
	throttled := !st.last.IsZero() && time.Since(st.last) < interval
	emitNow := reached && !throttled

	if emitNow {
		st.last = time.Now()
		if reset {
			st.count = 0
		}
	}

	ptr.lock.Unlock()

	if !emitNow {
		return nil
	}

	return ptr.emit(domain, event, data)
}

func (ptr *telemetryT) resetEventCounters(domain, event string, scope map[string]any) error {
	prefix := domain + "\x00" + event + "\x00"

	ptr.lock.Lock()
	defer ptr.lock.Unlock()

	for k, st := range ptr.ifMoreStates {
		if strings.HasPrefix(k, prefix) && matchScope(st.data, scope) {
			delete(ptr.ifMoreStates, k)
		}
	}

	return nil
}

// matchScope reports whether stored contains every key/value pair in scope,
// compared with the same JSON semantics used to bucket counters so a scope
// value need not match the stored value's exact Go type. A nil/empty scope
// matches everything, so a scopeless reset wipes all counters.
func matchScope(stored, scope map[string]any) bool {
	for k, v := range scope {
		sv, ok := stored[k]
		if !ok || dataFingerprint(map[string]any{k: sv}) != dataFingerprint(map[string]any{k: v}) {
			return false
		}
	}

	return true
}

func dataFingerprint(data map[string]any) string {
	b, err := json.Marshal(data)
	if err != nil {
		return fmt.Sprintf("%v", data)
	}

	return string(b)
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
		log.Warnf("[cliff] Read telemetry config, use defaults. err: %v", err)

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

	if ptr.version != "" {
		d := make(map[string]any, len(data)+1)
		maps.Copy(d, data)
		d["version"] = ptr.version
		data = d
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

	// Start() resumes the window as well as the constructor, so without this the second call would
	// arm a timer over the top of the first and leave it running until it expired. Stop-then-start,
	// like SetValidity and Enable.
	ptr.stopTimerLocked()

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
			log.Errorf("[cliff] Telemetry persist resume. err: %v", err)
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
		log.Errorf("[cliff] Telemetry persist disable. err: %v", err)
	}

	log.Infof("[cliff] Telemetry disabled: %s", reason)
}

func (ptr *telemetryT) applyConfigFromCloud(enabled bool, suppressed map[string]types.SuppressedEntry) {
	if err := ptr.Enable(enabled); err != nil {
		log.Errorf("[cliff] Enable telemetry=%v. err: %v", enabled, err)
	}

	if err := ptr.SetSuppressed(suppressed); err != nil {
		log.Errorf("[cliff] Set telemetry suppressed. err: %v", err)
	}
}
