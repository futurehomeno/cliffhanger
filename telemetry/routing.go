package telemetry

import (
	"github.com/futurehomeno/cliffhanger/config"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/storage"
)

const (
	SettingEnabled = "telemetry_enabled"
	// SettingValidity is the config parameter name used by the FIMP
	// cmd.config.set_telemetry_validity / cmd.config.get_telemetry_validity
	// commands. Once the window elapses since the last Enable(true),
	// the reporter auto-disables.
	SettingValidity = "telemetry_validity"
	// SettingSuppressed is the config parameter name used by the FIMP
	// cmd.config.set_telemetry_suppressed / cmd.config.get_telemetry_suppressed
	// commands. The payload is a string array of domain names; for those
	// domains Report/Emit are dropped and only ReportRequired/EmitRequired
	// publish.
	SettingSuppressed = "telemetry_suppressed"
)

// RoutingForTelemetry returns the get/set FIMP routings for the telemetry
// config parameters (enabled, validity, suppressed) bound to the given
// Telemetry.
func Route(tel Telemetry, model any, save func() error) []*router.Routing {
	defaultConfig, ok := model.(storage.DefaultConfigIf)

	if ok {
		return []*router.Routing{}
	}

	return []*router.Routing{
		config.RouteCmdConfigGetBool(svc, SettingEnabled, tel.IsEnabled, options...),
		config.RouteCmdConfigSetBool(svc, SettingEnabled, tel.Enable, options...),
		config.RouteCmdConfigGetDuration(svc, SettingValidity, tel.Validity, options...),
		config.RouteCmdConfigSetDuration(svc, SettingValidity, tel.SetValidity, options...),
		config.RouteCmdConfigGetStringArray(svc, SettingSuppressed, tel.SuppressedDomains, options...),
		config.RouteCmdConfigSetStringArray(svc, SettingSuppressed, tel.SetSuppressedDomains, options...),
	}
}
