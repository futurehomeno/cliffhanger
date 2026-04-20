package config

// configFileName is the default name of the persisted configuration file.
const configFileName = "config.json"

// Default is a set of configuration settings that are common for almost all applications running on a hub.
type Default struct {
	WorkDir            string `json:"-"`
	ConfigDir          string `json:"-"`
	ConfigVersion      string `json:"config_version,omitempty"`
	MQTTServerURI      string `json:"mqtt_server_uri"`
	MQTTUsername       string `json:"mqtt_server_username"`
	MQTTPassword       string `json:"mqtt_server_password"`
	MQTTClientIDPrefix string `json:"mqtt_client_id_prefix"`
	InfoFile           string `json:"info_file"`
	LogFile            string `json:"log_file"`
	LogLevel           string `json:"log_level"`
	LogFormat          string `json:"log_format"`
	LogRevertTimeout   string `json:"log_revert_timeout,omitempty"`
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
