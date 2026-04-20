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

// Telemetry emits telemetryT events over MQTT to the cloud backend-service.
type Telemetry interface {
	// Report publishes an event with the given name, optional domain, and
	// free-form data payload. Returns nil without publishing when disabled.
	Report(event, domain string, data map[string]any) error
	// SetTargetTopic overrides the default target topic.
	// Passing an empty string restores the default.
	SetTargetTopic(topic string)
	// Enable toggles reporting. Enable(true) always re-stamps the
	// enabled-at timestamp and restarts the validity window — including
	// when the telemetryT is already enabled — which incurs a store write on
	// every call. Enable(false) makes subsequent Report calls silent
	// no-ops. Callers that want idempotent behavior should gate on
	// IsEnabled first.
	Enable(enabled bool) error
	// IsEnabled reports whether telemetryT is currently publishing.
	IsEnabled() bool
	// Validity returns the window telemetryT stays enabled after Enable(true)
	// before it auto-disables.
	Validity() time.Duration
	// SetValidity updates the validity window. Must be positive.
	SetValidity(validity time.Duration) error
}

// New returns a Telemetry that publishes telemetryT events as the given source.
// The source becomes the FIMP src field and is used by the consumer to populate
// the app column, so it must uniquely identify the emitting application. The
// store is required and persists enabled / validity across restarts.
func New(mqtt *fimpgo.MqttTransport, source string, store Store) (Telemetry, error) {
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

	r := &telemetryT{
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
				return nil, fmt.Errorf("telemetryT: persist enabled_at: %w", err)
			}
		}

		elapsed := time.Since(enabledAt)
		if elapsed >= validity {
			r.enabled = false

			if err := store.SetEnabled(false); err != nil {
				return nil, fmt.Errorf("telemetryT: persist enabled: %w", err)
			}

			if err := store.SetEnabledAt(time.Time{}); err != nil {
				return nil, fmt.Errorf("telemetryT: clear enabled_at: %w", err)
			}

			log.Infof("[cliff] Telemetry disabled: validity expired before startup")
		} else {
			r.enabledAt = enabledAt
			r.startTimerLocked(validity - elapsed)
		}
	}

	return r, nil
}

// telemetryT holds the telemetryT's mutable state under a single mutex.
//
// Store writes (Enable, SetValidity, disableLocked) happen while the lock is
// held. That keeps the in-memory state and the on-disk state consistent under
// concurrent callers, at the cost of blocking Report / IsEnabled / Validity
// while the store persists. A slow or blocking store will back up those
// callers — acceptable for the file-backed config.Default store, but worth
// revisiting for stores with higher write latency.
type telemetryT struct {
	mqtt   *fimpgo.MqttTransport
	source fimptype.ResourceNameT
	store  Store

	lock      sync.Mutex
	topic     string
	enabled   bool
	validity  time.Duration
	enabledAt time.Time
	timer     *time.Timer
}

func (r *telemetryT) Report(event, domain string, data map[string]any) error {
	if event == "" {
		return errors.New("telemetryT event name is required")
	}

	r.lock.Lock()
	topic := r.topic
	enabled := r.enabled
	r.lock.Unlock()

	if !enabled {
		return nil
	}

	msg := fimpgo.NewObjectMessage(MessageType, Service, &Event{
		Event:  event,
		Domain: domain,
		Data:   data,
	}, nil, nil, nil)
	msg.Source = r.source

	if err := r.mqtt.PublishToTopic(topic, msg); err != nil {
		return fmt.Errorf("publish telemetryT event err: %w", err)
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

	if err := r.store.SetEnabled(enabled); err != nil {
		return fmt.Errorf("telemetryT: persist enabled: %w", err)
	}

	r.enabled = enabled
	r.stopTimerLocked()

	if enabled {
		r.enabledAt = time.Now()

		if err := r.store.SetEnabledAt(r.enabledAt); err != nil {
			return fmt.Errorf("telemetryT: persist enabled_at: %w", err)
		}

		r.startTimerLocked(r.validity)
	} else {
		r.enabledAt = time.Time{}

		if err := r.store.SetEnabledAt(time.Time{}); err != nil {
			return fmt.Errorf("telemetryT: clear enabled_at: %w", err)
		}
	}

	return nil
}

func (r *telemetryT) IsEnabled() bool {
	r.lock.Lock()
	defer r.lock.Unlock()

	return r.enabled
}

func (r *telemetryT) Validity() time.Duration {
	r.lock.Lock()
	defer r.lock.Unlock()

	return r.validity
}

func (r *telemetryT) SetValidity(validity time.Duration) error {
	if validity <= 0 {
		return errors.New("telemetryT: validity must be positive")
	}

	r.lock.Lock()
	defer r.lock.Unlock()

	if err := r.store.SetValidity(validity); err != nil {
		return fmt.Errorf("telemetryT: persist validity: %w", err)
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

// startTimerLocked must be called with r.lock held, or before the telemetryT
// has been published to other goroutines (e.g. from inside New).
func (r *telemetryT) startTimerLocked(d time.Duration) {
	var t *time.Timer
	t = time.AfterFunc(d, func() {
		r.lock.Lock()
		defer r.lock.Unlock()

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

// disableLocked disables the telemetryT in memory and persists the change.
// Store errors are logged and swallowed rather than propagated: this path is
// also reached from the timer goroutine where there is no caller to surface
// the error to. In the worst case on-disk enabled stays true while in-memory
// is false; New handles the "already-expired" case on next startup, so a
// failed persist leaks at most one reboot's worth of telemetryT.
func (r *telemetryT) disableLocked(reason string) {
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
