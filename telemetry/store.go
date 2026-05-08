package telemetry

import (
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/cliffhanger/config"
)

// State is the persisted telemetry configuration. All fields are written
// atomically by Store.Save so partial-update failures cannot occur.
type State struct {
	Enabled    bool
	EnabledAt  time.Time
	Validity   time.Duration
	Suppressed bool
}

// Store persists telemetry configuration so the enabled flag, the timestamp
// the reporter was last enabled at, and the validity window all survive
// application restarts. Consumer applications implement this against their
// own configuration storage.
//
// Implementations do not need to be thread-safe: the telemetry reporter
// serializes all Load/Save calls under its own mutex.
type Store interface {
	// Load returns the current persisted state.
	Load() State
	// Save atomically persists the full state.
	Save(State) error
}

// NewDefaultStore adapts a config.Default-backed persistence layer to the
// Store interface. The accessor must return a pointer to the embedded Default
// block; save persists the entire config to disk. Both callbacks must be non-nil.
func NewDefaultStore(accessor func() *config.Default, save func() error) Store {
	if accessor == nil {
		panic("telemetry: NewDefaultStore: accessor must not be nil")
	}

	if save == nil {
		panic("telemetry: NewDefaultStore: save must not be nil")
	}

	return &defaultStore{accessor: accessor, save: save}
}

type defaultStore struct {
	accessor func() *config.Default
	save     func() error
}

func (s *defaultStore) Load() State {
	cfg := s.accessor()

	var st State

	if cfg.TelemetryEnabled == nil {
		st.Enabled = true
	} else {
		st.Enabled = *cfg.TelemetryEnabled
	}

	if cfg.TelemetryEnabledAt != "" {
		if t, err := time.Parse(time.RFC3339Nano, cfg.TelemetryEnabledAt); err == nil {
			st.EnabledAt = t
		} else {
			log.WithError(err).Warnf("[cliff] Telemetry: ignoring malformed enabled_at %q", cfg.TelemetryEnabledAt)
		}
	}

	if cfg.TelemetryValidity != "" {
		d, err := time.ParseDuration(cfg.TelemetryValidity)

		switch {
		case err != nil:
			log.WithError(err).Warnf("[cliff] Telemetry: ignoring malformed validity %q", cfg.TelemetryValidity)
		case d <= 0:
			log.Warnf("[cliff] Telemetry: ignoring non-positive validity %q", cfg.TelemetryValidity)
		default:
			st.Validity = d
		}
	}

	if st.Validity <= 0 {
		st.Validity = DefaultValidity
	}

	if cfg.TelemetrySuppressed != nil {
		st.Suppressed = *cfg.TelemetrySuppressed
	}

	return st
}

func (s *defaultStore) Save(st State) error {
	cfg := s.accessor()

	// Snapshot so we can restore on save failure. The shared config.Default
	// must not carry unsaved telemetry mutations that a later unrelated
	// save() could flush to disk.
	prevEnabled := cfg.TelemetryEnabled
	prevEnabledAt := cfg.TelemetryEnabledAt
	prevValidity := cfg.TelemetryValidity
	prevSuppressed := cfg.TelemetrySuppressed

	v := st.Enabled
	cfg.TelemetryEnabled = &v

	if st.EnabledAt.IsZero() {
		cfg.TelemetryEnabledAt = ""
	} else {
		cfg.TelemetryEnabledAt = st.EnabledAt.UTC().Format(time.RFC3339Nano)
	}

	cfg.TelemetryValidity = st.Validity.String()

	sup := st.Suppressed
	cfg.TelemetrySuppressed = &sup

	if err := s.save(); err != nil {
		cfg.TelemetryEnabled = prevEnabled
		cfg.TelemetryEnabledAt = prevEnabledAt
		cfg.TelemetryValidity = prevValidity
		cfg.TelemetrySuppressed = prevSuppressed

		return err
	}

	return nil
}

// NewMemoryStore returns an in-memory Store suitable for tests or
// applications that do not need telemetry state to survive restarts.
func NewMemoryStore(enabled bool) Store {
	return &memoryStore{state: State{
		Enabled:  enabled,
		Validity: DefaultValidity,
	}}
}

type memoryStore struct {
	state State
}

func (s *memoryStore) Load() State {
	return s.state
}

func (s *memoryStore) Save(st State) error {
	s.state = st

	return nil
}
