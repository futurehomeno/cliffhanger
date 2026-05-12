package bootstrap

import (
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/config"
	"github.com/futurehomeno/cliffhanger/debug"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/telemetry"
)

// DefaultRoute bundles the standard config-report, debug and telemetry routes for a service.
func DefaultRoute(
	serviceName fimptype.ServiceNameT,
	configGetter func() any,
	tel telemetry.Telemetry,
	options ...config.RoutingOption,
) []*router.Routing {
	routes := []*router.Routing{
		config.RouteCmdConfigGetReport(serviceName, configGetter, options...),
	}

	routes = append(routes, debug.Route(serviceName, options...)...)
	routes = append(routes, telemetry.Route(tel, options...)...)

	return routes
}
