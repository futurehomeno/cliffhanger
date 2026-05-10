package bootstrap

import (
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/config"
	"github.com/futurehomeno/cliffhanger/debug"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/telemetry"
)

func DefaultRoute(
	serviceName fimptype.ServiceNameT,
	store *config.DefaultStore,
	tel telemetry.Telemetry,
	options ...config.RoutingOption,
) []*router.Routing {
	routes := []*router.Routing{
		config.RouteCmdConfigGetReport(serviceName, store.Default, options...),
	}

	routes = append(routes, debug.Route(serviceName, options...)...)
	routes = append(routes, telemetry.Route(tel, options...)...)

	return routes
}
