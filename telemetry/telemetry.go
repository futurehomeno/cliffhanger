package telemetry

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
	log "github.com/sirupsen/logrus"
)

// Telemetry emits telemetry events over MQTT to the cloud backend-service.
type Telemetry interface {
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
	// Enable toggles reporting. Enable(true) always re-stamps the
	// enabled-at timestamp and restarts the validity window - including
	// when telemetry is already enabled - which incurs a store write on
	// every call. Enable(false) makes subsequent Report calls silent
	// no-ops. Callers that want idempotent behavior should gate on
	// IsEnabled first.
	Enable(enabled bool) error
	// IsEnabled reports whether telemetry is currently publishing.
	IsEnabled() bool
	// Validity returns the window telemetry stays enabled after Enable(true)
	// before it auto-disables.
	Validity() time.Duration
	// SetValidity updates the validity window. Must be positive.
	SetValidity(validity time.Duration) error
	// SetSuppressed marks this source as suppressed (true) or active (false).
	// When suppressed, Report returns nil but ReportRequired still publishes.
	SetSuppressed(suppressed bool) error
	// IsSuppressed reports whether this source is currently suppressed.
	IsSuppressed() bool
}

// Emit calls Report and logs on error. Nil-safe: returns immediately
// if tel is nil, so callers do not need a nil guard.
func Emit(tel Telemetry, event, domain string, data map[string]any) {
	if tel == nil {
		return
	}

	if err := tel.Report(event, domain, data); err != nil {
		log.WithError(err).Warnf("[cliff] Telemetry: %s", event)
	}
}

// EmitRequired calls ReportRequired and logs on error. If tel is nil,
// the event is dropped with a warning - required events should not be
// silently lost.
func EmitRequired(tel Telemetry, event, domain string, data map[string]any) {
	if tel == nil {
		log.Warnf("[cliff] Telemetry: dropping required event %q (reporter is nil)", event)

		return
	}

	if err := tel.ReportRequired(event, domain, data); err != nil {
		log.WithError(err).Warnf("[cliff] Telemetry: %s", event)
	}
}

// New returns a Telemetry that publishes telemetry events as the given source.
// The source becomes the FIMP src field and is used by the consumer to populate
// the app column, so it must uniquely identify the emitting application. The
// store is required and persists enabled / validity across restarts.
func New(mqtt *fimpgo.MqttTransport, source string, store Store) (Telemetry, error) {
	if mqtt == nil {
		return nil, errors.New("telemetry: mqtt transport is nil")
	}

	if source == "" {
		return nil, errors.New("telemetry: source is not set")
	}

	if store == nil {
		return nil, errors.New("telemetry: store is required")
	}

	st := store.Load()
	if st.Validity <= 0 {
		st.Validity = DefaultValidity
	}

	r := &telemetryT{
		mqtt:       mqtt,
		source:     fimptype.ResourceNameT(source),
		store:      store,
		topic:      Topic,
		enabled:    st.Enabled,
		suppressed: st.Suppressed,
		validity:   st.Validity,
	}

	if r.enabled { //nolint:nestif
		enabledAt := st.EnabledAt
		if enabledAt.IsZero() {
			enabledAt = time.Now()

			st.EnabledAt = enabledAt
			if err := store.Save(st); err != nil {
				return nil, fmt.Errorf("telemetry: persist enabled_at: %w", err)
			}
		}

		elapsed := time.Since(enabledAt)
		if elapsed < 0 {
			// Clock skew: persisted timestamp is in the future.
			// Normalize to now so SetValidity doesn't overshoot.
			enabledAt = time.Now()
			elapsed = 0

			st.EnabledAt = enabledAt
			if err := store.Save(st); err != nil {
				return nil, fmt.Errorf("telemetry: persist normalized enabled_at: %w", err)
			}
		}

		if elapsed >= st.Validity {
			r.enabled = false

			st.Enabled = false
			st.EnabledAt = time.Time{}

			if err := store.Save(st); err != nil {
				return nil, fmt.Errorf("telemetry: persist disabled state: %w", err)
			}

			log.Infof("[cliff] Telemetry disabled: validity expired before startup")
		} else {
			r.enabledAt = enabledAt

			// Hold the lock so r.timer is assigned before the AfterFunc
			// callback can acquire it - prevents a race on tiny durations.
			r.lock.Lock()
			r.startTimerLocked(st.Validity - elapsed)
			r.lock.Unlock()
		}
	}

	if r.enabled {
		log.Infof("[cliff] Telemetry enabled (source=%s, validity=%s)", source, r.validity)
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
	mqtt   *fimpgo.MqttTransport
	source fimptype.ResourceNameT
	store  Store

	lock       sync.Mutex
	topic      string
	enabled    bool
	suppressed bool
	validity   time.Duration
	enabledAt  time.Time
	timer      *time.Timer
}

func (r *telemetryT) Report(event, domain string, data map[string]any) error {
	r.lock.Lock()
	topic := r.topic
	enabled := r.enabled
	suppressed := r.suppressed
	r.lock.Unlock()

	if !enabled || suppressed {
		return nil
	}

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

	st := State{
		Enabled:    enabled,
		Validity:   r.validity,
		Suppressed: r.suppressed,
	}

	if enabled {
		st.EnabledAt = time.Now()
	}

	if err := r.store.Save(st); err != nil {
		return fmt.Errorf("telemetry: persist state: %w", err)
	}

	r.stopTimerLocked()
	r.enabled = enabled
	r.enabledAt = st.EnabledAt

	if enabled {
		r.startTimerLocked(r.validity)
	}

	return nil
}

func (r *telemetryT) IsEnabled() bool {
	r.lock.Lock()
	defer r.lock.Unlock()

	return r.enabled
}

func (r *telemetryT) SetSuppressed(suppressed bool) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	st := State{
		Enabled:    r.enabled,
		EnabledAt:  r.enabledAt,
		Validity:   r.validity,
		Suppressed: suppressed,
	}

	if err := r.store.Save(st); err != nil {
		return fmt.Errorf("telemetry: persist suppressed: %w", err)
	}

	r.suppressed = suppressed

	return nil
}

func (r *telemetryT) IsSuppressed() bool {
	r.lock.Lock()
	defer r.lock.Unlock()

	return r.suppressed
}

func (r *telemetryT) Validity() time.Duration {
	r.lock.Lock()
	defer r.lock.Unlock()

	return r.validity
}

func (r *telemetryT) SetValidity(validity time.Duration) error {
	if validity <= 0 {
		return errors.New("telemetry: validity must be positive")
	}

	r.lock.Lock()
	defer r.lock.Unlock()

	// Compute elapsed once and use it for both the Save decision and the
	// post-Save timer scheduling, avoiding a TOCTOU gap during slow saves.
	var elapsed time.Duration

	newEnabled := r.enabled
	newEnabledAt := r.enabledAt
	shouldDisable := false

	if r.enabled && !r.enabledAt.IsZero() {
		elapsed = time.Since(r.enabledAt)
		if elapsed < 0 {
			elapsed = 0 // clock skew: treat future timestamps as "just enabled"
		}

		if elapsed >= validity {
			newEnabled = false
			newEnabledAt = time.Time{}
			shouldDisable = true
		}
	}

	st := State{
		Enabled:    newEnabled,
		EnabledAt:  newEnabledAt,
		Validity:   validity,
		Suppressed: r.suppressed,
	}

	if err := r.store.Save(st); err != nil {
		return fmt.Errorf("telemetry: persist validity: %w", err)
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
	r.enabled = false
	r.enabledAt = time.Time{}
	r.timer = nil

	st := State{Validity: r.validity, Suppressed: r.suppressed}

	if err := r.store.Save(st); err != nil {
		log.WithError(err).Errorf("[cliff] Telemetry: failed to persist disabled state")
	}

	log.Infof("[cliff] Telemetry disabled: %s", reason)
}
