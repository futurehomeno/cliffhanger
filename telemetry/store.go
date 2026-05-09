package telemetry

import (
	"time"
)

// NewMemoryStore returns an in-memory config.TelemetryStore suitable for
// tests or applications that do not need telemetry state to survive
// restarts.
func NewMemoryStore(enabled bool) store {
	return &memoryStore{
		enabled:  &enabled,
		validity: defaultTelemetryValidity,
	}
}

type memoryStore struct {
	enabled    *bool
	enabledAt  time.Time
	validity   time.Duration
	suppressed *bool
}

func (s *memoryStore) Enabled() *bool { return s.enabled }

func (s *memoryStore) SetEnabled(enabled *bool) error {
	s.enabled = enabled

	return nil
}

func (s *memoryStore) EnabledAt() time.Time { return s.enabledAt }

func (s *memoryStore) SetEnabledAt(t time.Time) error {
	s.enabledAt = t

	return nil
}

func (s *memoryStore) Validity() time.Duration { return s.validity }

func (s *memoryStore) SetValidity(d time.Duration) error {
	s.validity = d

	return nil
}

func (s *memoryStore) Suppressed() *bool { return s.suppressed }

func (s *memoryStore) SetSuppressed(suppressed *bool) error {
	s.suppressed = suppressed

	return nil
}
