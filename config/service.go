package config

import (
	"time"

	"github.com/futurehomeno/cliffhanger/storage"
)

// Service is a generic thread-safe configuration service base for applications.
// The configuration model must embed Default, exposed through the defaults accessor.
// The optional redact function strips credentials for public reporting, satisfying app.PublicModeler.
// Service shares the DefaultStore lock, since both mutate the same underlying model.
// The lock is not reentrant: callbacks passed to Update, Persist and Migrate must not
// call back into Service or DefaultStore methods.
type Service[C any] struct {
	storage.Storage[C]

	defaultStore *DefaultStore
	redact       func(C) any
}

func NewService[C any](s storage.Storage[C], defaults func(C) *Default, redact func(C) any) *Service[C] {
	return &Service[C]{
		Storage:      s,
		defaultStore: NewDefaultStoreFromStorage(s, defaults),
		redact:       redact,
	}
}

// Update applies fn to the model under lock, stamps the configuration time and saves.
func (s *Service[C]) Update(fn func(model C)) error {
	s.defaultStore.lock.Lock()
	defer s.defaultStore.lock.Unlock()

	fn(s.Model())

	return s.defaultStore.saveStamped()
}

// Persist applies fn to the model under lock and saves without stamping the configuration
// time, for caches and internal state that are not user configuration.
func (s *Service[C]) Persist(fn func(model C)) error {
	s.defaultStore.lock.Lock()
	defer s.defaultStore.lock.Unlock()

	fn(s.Model())

	return s.Save()
}

// Reset restores the configuration to defaults under lock.
func (s *Service[C]) Reset() error {
	s.defaultStore.lock.Lock()
	defer s.defaultStore.lock.Unlock()

	return s.Storage.Reset()
}

// Migrate runs migrations under lock and saves the model if any step was applied.
func (s *Service[C]) Migrate(migrations ...Migration) error {
	s.defaultStore.lock.Lock()
	defer s.defaultStore.lock.Unlock()

	applied, err := s.defaultStore.accessor().Migrate(migrations...)
	if err != nil {
		return err
	}

	if applied == 0 {
		return nil
	}

	return s.Save()
}

// DefaultStore exposes the embedded Default settings, as required by the app builders.
func (s *Service[C]) DefaultStore() *DefaultStore {
	return s.defaultStore
}

// PublicModel returns the redacted configuration for public reporting, or the full model
// if no redact function was provided.
func (s *Service[C]) PublicModel() any {
	s.defaultStore.lock.RLock()
	defer s.defaultStore.lock.RUnlock()

	if s.redact == nil {
		return s.Model()
	}

	return s.redact(s.Model())
}

// Get reads a setting from the model under read lock. The accessor should return
// a value type or a copy: a returned reference type aliases the live model after
// the lock is released.
func Get[C, V any](s *Service[C], get func(model C) V) V {
	s.defaultStore.lock.RLock()
	defer s.defaultStore.lock.RUnlock()

	return get(s.Model())
}

// GetDuration reads a duration setting persisted as string, falling back to def when unset or invalid.
func GetDuration[C any](s *Service[C], get func(model C) string, def time.Duration) time.Duration {
	return parseDurationOr(Get(s, get), def)
}

func parseDurationOr(s string, def time.Duration) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return def
	}

	return d
}
