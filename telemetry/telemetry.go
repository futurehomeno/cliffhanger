package telemetry

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/futurehomeno/fimpgo"
	log "github.com/sirupsen/logrus"
)

// Telemetry emits telemetry events over MQTT to the cloud backend-service.
type Telemetry interface {
	Store
	// Report publishes an event with the given name, optional domain, and
	// free-form data payload. Returns nil without publishing when telemetry
	// is disabled or the source is suppressed.
	Report(event, domain string, data map[string]any) error
	// ReportRequired publishes an event that always flows regardless of the
	// enabled flag or suppressed state. Use for critical events such as
	// device health transitions that must not be silenced.
	ReportRequired(event, domain string, data map[string]any) error
	// SetTargetTopic overrides the default target topic.
	// Passing an empty string restores the default. The override is
	// not persisted and resets to the default on restart.
	SetTargetTopic(topic string)
}

// Emit calls Report and logs on error. Nil-safe: returns immediately
// if tel is nil, so callers do not need a nil guard.
//
// Callers must pass the interface value directly, not a typed-nil pointer.
// A typed-nil (e.g. (*myImpl)(nil) stored in a Telemetry variable) is not
// caught by the nil check and will panic.
func Emit(tel Telemetry, event, domain string, data map[string]any) {
	if tel == nil {
		return
	}

	if err := tel.Report(event, domain, data); err != nil {
		log.WithError(err).Warnf("[cliff] Telemetry: %q", event)
	}
}

// EmitRequired calls ReportRequired and logs on error. If tel is nil,
// the event is dropped with a warning - required events should not be
// silently lost.
//
// Callers must pass the interface value directly, not a typed-nil pointer.
// A typed-nil (e.g. (*myImpl)(nil) stored in a Telemetry variable) is not
// caught by the nil check and will panic.
func EmitRequired(tel Telemetry, event, domain string, data map[string]any) {
	if tel == nil {
		log.Warnf("[cliff] Telemetry: dropping required event %q (reporter is nil)", event)

		return
	}

	if err := tel.ReportRequired(event, domain, data); err != nil {
		log.WithError(err).Warnf("[cliff] Telemetry: %q", event)
	}
}

// New returns a Telemetry that publishes telemetry events as the given source.
// The source becomes the FIMP src field and is used by the consumer to populate
// the app column, so it must uniquely identify the emitting application. The
// store is required and persists enabled / validity across restarts.
func New(mqtt *fimpgo.MqttTransport, store storeIf) (Telemetry, error) {
	if mqtt == nil {
		return nil, errors.New("telemetry: mqtt transport is nil")
	}

	if store == nil {
		return nil, errors.New("telemetry: store is required")
	}

	enabled := true
	if v := store.Enabled(); v != nil {
		enabled = *v
	}

	validity := store.Validity()
	if validity <= 0 {
		validity = defaultTelemetryValidity
	}

	r := &telemetryT{
		mqtt:  mqtt,
		store: store,
		topic: Topic,
	}

	return r, nil
}

// telemetryT holds the reporter's mutable state under a single mutex.
//
// Store writes (Enable, SetValidity, disableLocked) happen while the lock is
// held. That keeps the in-memory state and the on-disk state consistent under
// concurrent callers, at the cost of blocking Report / IsEnabled / Validity
// while the store persists. A slow or blocking store will back up those
// callers - acceptable for the file-backed config.Default store, but worth
// revisiting for stores with higher write latency.
type telemetryT struct {
	mqtt  *fimpgo.MqttTransport
	store storeIf
	lock  sync.Mutex
	topic string
	timer *time.Timer
}

func (r *telemetryT) Report(event, domain string, data map[string]any) error {
	r.lock.Lock()
	topic := r.topic

	r.lock.Unlock()

	// TODO: check if domain is suppressed here instead of in the caller, to avoid the overhead of constructing and publishing messages that will be dropped. Requires exposing the list of suppressed domains from the store, and a decision on how to handle changes to that list while running (e.g. if Report is in-flight while a new config is applied with an updated list).

	return r.publish(topic, event, domain, data)
}

func (r *telemetryT) ReportRequired(event, domain string, data map[string]any) error {
	r.lock.Lock()
	topic := r.topic
	r.lock.Unlock()

	return r.publish(topic, event, domain, data)
}

func (r *telemetryT) publish(topic, event, domain string, data map[string]any) error {
	if event == "" {
		return errors.New("telemetry: event name is required")
	}

	msg := fimpgo.NewObjectMessage(MessageType, Service, &Event{
		Event:  event,
		Domain: domain,
		Data:   data,
	}, nil, nil, nil)
	msg.Source = r.source

	if err := r.mqtt.PublishToTopic(topic, msg); err != nil {
		return fmt.Errorf("telemetry: publish event: %w", err)
	}

	return nil
}

func (r *telemetryT) SetTargetTopic(topic string) {
	if topic == "" {
		topic = Topic
	}

	r.lock.Lock()
	r.topic = topic
	r.lock.Unlock()
}

func (r *telemetryT) Enable(enabled bool) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	if err := r.store.SetEnabled(&enabled); err != nil {
		return fmt.Errorf("telemetry: persist enabled: %w", err)
	}

	r.stopTimerLocked()

	if enabled {
		r.startTimerLocked(r.validity) // TODO: take validity from Store()
	}

	return nil
}

func (r *telemetryT) IsEnabled() bool {
	r.lock.Lock()
	defer r.lock.Unlock()

	return r.enabled // TODO: take enabled from Store()
}

func (r *telemetryT) SetDisabledDomains(disabledDOmains []string) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	if err := r.store.SetDisabledDomains(disabledDOmains); err != nil {
		return fmt.Errorf("telemetry: persist suppressed: %w", err)
	}

	return nil
}

func (r *telemetryT) DisabledDomains() []string {
	r.lock.Lock()
	defer r.lock.Unlock()

	return r.disabledDomains
}

func (r *telemetryT) Validity() time.Duration {
	r.lock.Lock()
	defer r.lock.Unlock()

	return r.validity // TODO: take Validity from Store()
}

func (r *telemetryT) SetValidity(validity time.Duration) error {
	if validity <= 0 {
		return errors.New("telemetry: validity must be positive")
	}

	r.lock.Lock()
	defer r.lock.Unlock()

	// Compute elapsed once and use it for both the disable decision and the
	// post-write timer scheduling, avoiding a TOCTOU gap during slow saves.
	var elapsed time.Duration

	shouldDisable := false

	if r.enabled && !r.enabledAt.IsZero() {
		elapsed = time.Since(r.enabledAt)
		if elapsed < 0 {
			elapsed = 0 // clock skew: treat future timestamps as "just enabled"
		}

		if elapsed >= validity {
			shouldDisable = true
		}
	}

	if err := r.store.SetValidity(validity); err != nil {
		return fmt.Errorf("telemetry: persist validity: %w", err)
	}

	if shouldDisable {
		if err := r.store.SetEnabledAt(time.Time{}); err != nil {
			return fmt.Errorf("telemetry: persist cleared enabled_at: %w", err)
		}

		disabled := false
		if err := r.store.SetEnabled(&disabled); err != nil {
			return fmt.Errorf("telemetry: persist disabled state: %w", err)
		}
	}

	r.validity = validity
	r.stopTimerLocked()

	if shouldDisable {
		r.enabled = false
		r.enabledAt = time.Time{}

		log.Infof("[cliff] Telemetry disabled: validity reduced below elapsed time")
	} else if r.enabled && !r.enabledAt.IsZero() {
		r.startTimerLocked(validity - elapsed)
	}

	return nil
}

// startTimerLocked must be called with r.lock held, or before the reporter
// has been published to other goroutines (e.g. from inside New).
func (r *telemetryT) startTimerLocked(d time.Duration) {
	var t *time.Timer
	t = time.AfterFunc(d, func() {
		r.lock.Lock()
		defer r.lock.Unlock()

		// Guard against a stale callback: if stopTimerLocked replaced
		// r.timer since this AfterFunc was scheduled, bail out.
		if r.timer != t {
			return
		}

		r.disableLocked("validity expired")
	})
	r.timer = t
}

func (r *telemetryT) stopTimerLocked() {
	if r.timer != nil {
		r.timer.Stop()
		r.timer = nil
	}
}

// disableLocked disables telemetry in memory and persists the change.
// Store errors are logged and swallowed rather than propagated: this path is
// also reached from the timer goroutine where there is no caller to surface
// the error to. In the worst case on-disk enabled stays true while in-memory
// is false; New handles the "already-expired" case on next startup, so a
// failed persist leaks at most one reboot's worth of telemetry.
func (r *telemetryT) disableLocked(reason string) {
	r.timer = nil

	if err := r.store.SetEnabledAt(time.Time{}); err != nil {
		log.WithError(err).Errorf("[cliff] Telemetry: failed to persist cleared enabled_at")
	}

	disabled := false
	if err := r.store.SetEnabled(&disabled); err != nil {
		log.WithError(err).Errorf("[cliff] Telemetry: failed to persist disabled state")
	}

	log.Infof("[cliff] Telemetry disabled: %s", reason)
}
