package config

import (
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/futurehomeno/cliffhanger/storage"
	"github.com/futurehomeno/cliffhanger/telemetry/types"
)

type DefaultStore struct {
	accessor func() *Default
	save     func() error
	lock     sync.RWMutex
}

type Default struct {
	WorkDir            string                 `json:"-"`
	ConfigDir          string                 `json:"-"`
	ConfigVersion      int                    `json:"config_version,omitempty"`
	MQTTServerURI      string                 `json:"mqtt_server_uri"`
	MQTTUsername       string                 `json:"mqtt_server_username"`
	MQTTPassword       string                 `json:"mqtt_server_password"`
	MQTTClientIDPrefix string                 `json:"mqtt_client_id_prefix"`
	InfoFile           string                 `json:"info_file"`
	LogFile            string                 `json:"log_file"`
	LogLevel           string                 `json:"log_level"`
	LogFormat          string                 `json:"log_format"`
	LogRevertTimeout   time.Duration          `json:"log_revert_timeout,omitempty"`
	LogRevertAt        time.Time              `json:"log_revert_at"`
	RestartsCount      int                    `json:"restarts_count,omitempty"`
	Telemetry          *types.TelemetryConfig `json:"telemetry,omitempty"`
	ConfiguredAt       string                 `json:"configured_at"`
}

func NewDefault(workDir string) Default {
	return Default{
		WorkDir:   workDir,
		ConfigDir: workDir,
	}
}

func NewCanonicalDefault(cfgDir, workDir string) Default {
	return Default{
		WorkDir:   workDir,
		ConfigDir: cfgDir,
	}
}

func (d *Default) IncrementRestartsCount() int {
	d.RestartsCount++

	return d.RestartsCount
}

func (d *Default) GetTelemetry() (types.TelemetryConfig, error) {
	if d.Telemetry == nil {
		return types.TelemetryConfig{}, nil
	}

	return *d.Telemetry, nil
}

func (d *Default) SetTelemetry(cfg types.TelemetryConfig) {
	c := cfg
	d.Telemetry = &c
}

func (d *Default) SetConfiguredAt(t time.Time) {
	d.ConfiguredAt = t.Format(time.RFC3339Nano)
}

func NewDefaultStore(accessor func() *Default, save func() error) *DefaultStore {
	return &DefaultStore{accessor: accessor, save: save}
}

func NewDefaultStoreFromStorage[T any](s storage.Storage[T], pick func(T) *Default) *DefaultStore {
	return NewDefaultStore(
		func() *Default { return pick(s.Model()) },
		s.Save,
	)
}

func (s *DefaultStore) Save() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	return s.saveStamped()
}

func (s *DefaultStore) saveStamped() error {
	s.accessor().SetConfiguredAt(time.Now())

	return s.save()
}

// Default returns a deep copy of the current Default, safe to use after
// the store lock is released - including for deferred JSON marshaling
// (e.g. fimpgo report payloads). The copy is taken under the read lock
// so it does not race with concurrent setters, and the embedded
// *TelemetryConfig (and its slice) is cloned so callers cannot mutate
// shared state.
func (s *DefaultStore) Default() *Default {
	s.lock.RLock()
	defer s.lock.RUnlock()

	snap := *s.accessor()
	if snap.Telemetry != nil {
		tc := *snap.Telemetry
		if tc.SuppressedDomains != nil {
			tc.SuppressedDomains = slices.Clone(tc.SuppressedDomains)
		}

		snap.Telemetry = &tc
	}

	return &snap
}

func (s *DefaultStore) Level() string {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.accessor().LogLevel
}

func (s *DefaultStore) SetLevel(level string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.accessor().LogLevel = level

	return s.saveStamped()
}

func (s *DefaultStore) Format() string {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.accessor().LogFormat
}

func (s *DefaultStore) SetFormat(format string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.accessor().LogFormat = format

	return s.saveStamped()
}

func (s *DefaultStore) LogFile() string {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.accessor().LogFile
}

func (s *DefaultStore) SetLogFile(file string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.accessor().LogFile = file

	return s.saveStamped()
}

func (s *DefaultStore) LogRevertTimeout() time.Duration {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.accessor().LogRevertTimeout
}

func (s *DefaultStore) SetLogRevertTimeout(d time.Duration) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.accessor().LogRevertTimeout = d

	return s.saveStamped()
}

func (s *DefaultStore) LogRevertAt() time.Time {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.accessor().LogRevertAt
}

func (s *DefaultStore) SetLogRevertAt(t time.Time) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.accessor().LogRevertAt = t

	return s.saveStamped()
}

func (s *DefaultStore) Telemetry() (types.TelemetryConfig, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if s.accessor().Telemetry == nil {
		return types.TelemetryConfig{}, fmt.Errorf("not_found")
	}

	return *s.accessor().Telemetry, nil
}

func (s *DefaultStore) SetTelemetry(cfg *types.TelemetryConfig) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	clone := *cfg
	if cfg.SuppressedDomains != nil {
		clone.SuppressedDomains = slices.Clone(cfg.SuppressedDomains)
	}

	s.accessor().Telemetry = &clone

	return s.saveStamped()
}

func (s *DefaultStore) IncrementRestartsCount() (int, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	count := s.accessor().IncrementRestartsCount()

	if err := s.save(); err != nil {
		return 0, err
	}

	return count, nil
}
