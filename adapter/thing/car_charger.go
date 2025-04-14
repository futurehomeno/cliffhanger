package thing

import (
	"time"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"
	"github.com/futurehomeno/cliffhanger/adapter/service/numericmeter"
	"github.com/futurehomeno/cliffhanger/adapter/service/parameters"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
)

// CarChargerConfig represents a thing configuration.
type CarChargerConfig struct {
	ThingConfig            *adapter.ThingConfig
	ChargepointConfig      *chargepoint.Config
	MeterElecConfig        *numericmeter.Config // Optional
	ParameterServiceConfig *parameters.Config
}

// NewCarCharger creates a thing that satisfies expectations for a car charger.
// Specification and implementation for electricity meter is optional.
func NewCarCharger(
	publisher adapter.Publisher,
	ts adapter.ThingState,
	cfg *CarChargerConfig,
) adapter.Thing {
	services := []adapter.Service{
		chargepoint.NewService(publisher, cfg.ChargepointConfig),
	}

	if cfg.MeterElecConfig != nil {
		services = append(services, numericmeter.NewService(publisher, cfg.MeterElecConfig))
	}

	if cfg.ParameterServiceConfig != nil {
		services = append(services, parameters.NewService(publisher, cfg.ParameterServiceConfig))
	}

	return adapter.NewThing(publisher, ts, cfg.ThingConfig, services...)
}

// RouteCarCharger creates routing required to satisfy expectations for a car charger.
func RouteCarCharger(adapter adapter.Adapter) []*router.Routing {
	return router.Combine(
		chargepoint.RouteService(adapter),
		numericmeter.RouteService(adapter),
		parameters.RouteService(adapter),
	)
}

// TaskCarCharger creates background tasks specific for a car charger.
func TaskCarCharger(
	adapter adapter.Adapter,
	reportingInterval time.Duration,
	reportingVoters ...task.Voter,
) []*task.Task {
	return []*task.Task{
		chargepoint.TaskReporting(adapter, reportingInterval, reportingVoters...),
		numericmeter.TaskReporting(adapter, reportingInterval, reportingVoters...),
	}
}
