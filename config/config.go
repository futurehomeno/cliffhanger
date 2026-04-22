package config

import "time"

const configFileName = "config.json"

type Default struct {
	WorkDir            string    `json:"-"`
	ConfigDir          string    `json:"-"`
	ConfigVersion      string    `json:"config_version,omitempty"`
	MQTTServerURI      string    `json:"mqtt_server_uri"`
	MQTTUsername       string    `json:"mqtt_server_username"`
	MQTTPassword       string    `json:"mqtt_server_password"`
	MQTTClientIDPrefix string    `json:"mqtt_client_id_prefix"`
	InfoFile           string    `json:"info_file"`
	LogFile            string    `json:"log_file"`
	LogLevel           string    `json:"log_level"`
	LogFormat          string    `json:"log_format"`
	LogRevertAt        time.Time `json:"log_revert_at,omitempty"`
	RestartsCount      int       `json:"restarts_count,omitempty"`
	ConfiguredAt       string    `json:"configured_at"`
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
