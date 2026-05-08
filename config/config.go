package config

import "time"

const configFileName = "config.json"

type Default struct {
	WorkDir             string        `json:"-"`
	ConfigDir           string        `json:"-"`
	ConfigVersion       int           `json:"config_version,omitempty"`
	MQTTServerURI       string        `json:"mqtt_server_uri"`
	MQTTUsername        string        `json:"mqtt_server_username"`
	MQTTPassword        string        `json:"mqtt_server_password"`
	MQTTClientIDPrefix  string        `json:"mqtt_client_id_prefix"`
	InfoFile            string        `json:"info_file"`
	LogFile             string        `json:"log_file"`
	LogLevel            string        `json:"log_level"`
	LogFormat           string        `json:"log_format"`
	LogRevertTimeout    time.Duration `json:"log_revert_timeout,omitempty"`
	LogRevertAt         time.Time     `json:"log_revert_at"`
	RestartsCount       int           `json:"restarts_count,omitempty"`
	TelemetryEnabled    *bool         `json:"telemetry_enabled,omitempty"`
	TelemetryEnabledAt  string        `json:"telemetry_enabled_at,omitempty"`
	TelemetryValidity   string        `json:"telemetry_validity,omitempty"`
	TelemetrySuppressed *bool         `json:"telemetry_suppressed,omitempty"`
	ConfiguredAt        string        `json:"configured_at"`
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

// DefaultProvider is implemented by any config model that embeds Default.
type DefaultProvider interface {
	GetDefault() *Default
}

func (d *Default) GetDefault() *Default {
	return d
}

// IncrementRestartsCount increments the restart counter in the model.
func (d *Default) IncrementRestartsCount() int {
	d.RestartsCount++

	return d.RestartsCount
}
