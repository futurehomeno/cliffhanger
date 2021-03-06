package config

// Name is a default name used by configuration file.
const Name = "config.json"

// Default is a set of configuration settings that are common for almost all applications running on a hub.
type Default struct {
	WorkDir            string `json:"-"`
	ConfigDir          string `json:"-"`
	MQTTServerURI      string `json:"mqtt_server_uri"`
	MQTTUsername       string `json:"mqtt_server_username"`
	MQTTPassword       string `json:"mqtt_server_password"`
	MQTTClientIDPrefix string `json:"mqtt_client_id_prefix"`
	InfoFile           string `json:"info_file"`
	LogFile            string `json:"log_file"`
	LogLevel           string `json:"log_level"`
	LogFormat          string `json:"log_format"`
	ConfiguredAt       string `json:"configured_at"`
}

// NewDefault creates a new instance of a default configuration.
func NewDefault(workDir string) Default {
	return Default{
		WorkDir:   workDir,
		ConfigDir: workDir,
	}
}

// NewCanonicalDefault creates a new instance of a canonical default configuration.
func NewCanonicalDefault(cfgDir, workDir string) Default {
	return Default{
		WorkDir:   workDir,
		ConfigDir: cfgDir,
	}
}
