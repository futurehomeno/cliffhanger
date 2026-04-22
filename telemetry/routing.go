package telemetry

import (
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/config"
	"github.com/futurehomeno/cliffhanger/router"
)

// RouteCmdGetEnabled returns a routing for cmd.config.get_telemetry_enabled
// that replies with the current telemetry enabled state.
func RouteCmdGetEnabled(svc fimptype.ServiceNameT, reporter Telemetry, options ...config.RoutingOption) *router.Routing {
	return config.RouteCmdConfigGetBool(svc, SettingEnabled, reporter.IsEnabled, options...)
}

// RouteCmdSetEnabled returns a routing for cmd.config.set_telemetry_enabled
// that toggles the telemetry enabled state.
func RouteCmdSetEnabled(svc fimptype.ServiceNameT, reporter Telemetry, options ...config.RoutingOption) *router.Routing {
	return config.RouteCmdConfigSetBool(svc, SettingEnabled, reporter.Enable, options...)
}

// RouteCmdGetValidity returns a routing for cmd.config.get_telemetry_validity
// that replies with the current validity window.
func RouteCmdGetValidity(svc fimptype.ServiceNameT, reporter Telemetry, options ...config.RoutingOption) *router.Routing {
	return config.RouteCmdConfigGetDuration(svc, SettingValidity, reporter.Validity, options...)
}

// RouteCmdSetValidity returns a routing for cmd.config.set_telemetry_validity
// that updates the validity window.
func RouteCmdSetValidity(svc fimptype.ServiceNameT, reporter Telemetry, options ...config.RoutingOption) *router.Routing {
	return config.RouteCmdConfigSetDuration(svc, SettingValidity, reporter.SetValidity, options...)
}

// RouteCmdGetSuppressed returns a routing for cmd.config.get_telemetry_suppressed
// that replies with the current telemetry suppressed state.
func RouteCmdGetSuppressed(svc fimptype.ServiceNameT, reporter Telemetry, options ...config.RoutingOption) *router.Routing {
	return config.RouteCmdConfigGetBool(svc, SettingSuppressed, reporter.IsSuppressed, options...)
}

// RouteCmdSetSuppressed returns a routing for cmd.config.set_telemetry_suppressed
// that toggles the telemetry suppressed state.
func RouteCmdSetSuppressed(svc fimptype.ServiceNameT, reporter Telemetry, options ...config.RoutingOption) *router.Routing {
	return config.RouteCmdConfigSetBool(svc, SettingSuppressed, reporter.SetSuppressed, options...)
}

// RoutingForTelemetry returns the get/set routings for the telemetry config
// parameters (enabled, validity, suppressed) bound to the given Telemetry instance.
func RoutingForTelemetry(svc fimptype.ServiceNameT, reporter Telemetry, options ...config.RoutingOption) []*router.Routing {
	return []*router.Routing{
		RouteCmdGetEnabled(svc, reporter, options...),
		RouteCmdSetEnabled(svc, reporter, options...),
		RouteCmdGetValidity(svc, reporter, options...),
		RouteCmdSetValidity(svc, reporter, options...),
		RouteCmdGetSuppressed(svc, reporter, options...),
		RouteCmdSetSuppressed(svc, reporter, options...),
	}
}
