package config

import (
	"time"

	"github.com/futurehomeno/cliffhanger/telemetry/types"
)

const configFileName = "config.json"

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

func (d *Default) SetTelemetry(telemetry *types.TelemetryConfig) {
	d.Telemetry = telemetry
}

func (d *Default) SetConfiguredAt(t time.Time) {
	d.ConfiguredAt = t.Format(time.RFC3339)
}
