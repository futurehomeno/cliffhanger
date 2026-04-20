package telemetry

import (
	"sync"
	"time"

	"github.com/futurehomeno/cliffhanger/config"
)

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
// The initial validity is DefaultValidity; with an empty enabledAt, New
// will stamp a fresh window on every restart.
func NewMemoryStore(enabled bool) Store {
	return &memoryStore{enabled: enabled, validity: DefaultValidity}
}

type memoryStore struct {
	lock      sync.Mutex
	enabled   bool
	enabledAt time.Time
	validity  time.Duration
}

func (s *memoryStore) Enabled() bool {
	s.lock.Lock()
	defer s.lock.Unlock()

	return s.enabled
}

func (s *memoryStore) SetEnabled(enabled bool) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.enabled = enabled

	return nil
}

func (s *memoryStore) EnabledAt() time.Time {
	s.lock.Lock()
	defer s.lock.Unlock()

	return s.enabledAt
}

func (s *memoryStore) SetEnabledAt(t time.Time) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.enabledAt = t

	return nil
}

func (s *memoryStore) Validity() time.Duration {
	s.lock.Lock()
	defer s.lock.Unlock()

	return s.validity
}

func (s *memoryStore) SetValidity(validity time.Duration) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.validity = validity

	return nil
}
