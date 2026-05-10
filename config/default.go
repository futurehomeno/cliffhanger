package config

import (
	"sync"
	"time"

	"github.com/futurehomeno/cliffhanger/storage"
	"github.com/futurehomeno/cliffhanger/telemetry/types"
)

const configFileName = "config.json"

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

// GetTelemetry returns a copy of the persisted telemetry block, or a zero
// value when none is set. The error return satisfies storage.DefaultConfigIf.
func (d *Default) GetTelemetry() (types.TelemetryConfig, error) {
	if d.Telemetry == nil {
		return types.TelemetryConfig{}, nil
	}

	return *d.Telemetry, nil
}

// SetTelemetry persists the given block by replacing the embedded pointer.
func (d *Default) SetTelemetry(cfg types.TelemetryConfig) {
	c := cfg
	d.Telemetry = &c
}

// SetConfiguredAt persists the given time using RFC3339.
func (d *Default) SetConfiguredAt(t time.Time) {
	d.ConfiguredAt = t.Format(time.RFC3339)
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

	return s.save()
}

func (s *DefaultStore) Default() *Default {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.accessor()
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

	return s.save()
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

	return s.save()
}

func (s *DefaultStore) File() string {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.accessor().LogFile
}

func (s *DefaultStore) SetFile(file string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.accessor().LogFile = file

	return s.save()
}

func (s *DefaultStore) RevertTimeout() time.Duration {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.accessor().LogRevertTimeout
}

func (s *DefaultStore) SetRevertTimeout(d time.Duration) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.accessor().LogRevertTimeout = d

	return s.save()
}

func (s *DefaultStore) RevertAt() time.Time {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.accessor().LogRevertAt
}

func (s *DefaultStore) SetRevertAt(t time.Time) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.accessor().LogRevertAt = t

	return s.save()
}

func (s *DefaultStore) Telemetry() *types.TelemetryConfig {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.accessor().Telemetry
}

func (s *DefaultStore) SetTelemetry(cfg *types.TelemetryConfig) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.accessor().Telemetry = cfg

	return s.save()
}
