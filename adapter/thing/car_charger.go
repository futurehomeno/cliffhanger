package thing

import (
	"time"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"
	"github.com/futurehomeno/cliffhanger/adapter/service/meterelec"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
)

// CarChargerConfig represents a thing configuration.
type CarChargerConfig struct {
	ThingConfig       *adapter.ThingConfig
	ChargepointConfig *chargepoint.Config
	MeterElecConfig   *meterelec.Config // Optional
}

// NewCarCharger creates a thing that satisfies expectations for a car charger.
// Specification and implementation for electricity meter is optional.
func NewCarCharger(
	a adapter.Adapter,
	ts adapter.ThingState,
	cfg *CarChargerConfig,
) adapter.Thing {
	services := []adapter.Service{
		chargepoint.NewService(a, cfg.ChargepointConfig),
	}

	if cfg.MeterElecConfig != nil {
		services = append(services, meterelec.NewService(a, cfg.MeterElecConfig))
	}

	return adapter.NewThing(a, ts, cfg.ThingConfig, services...)
}

// RouteCarCharger creates routing required to satisfy expectations for a car charger.
func RouteCarCharger(adapter adapter.Adapter) []*router.Routing {
	return router.Combine(
		chargepoint.RouteService(adapter),
		meterelec.RouteService(adapter),
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
		meterelec.TaskReporting(adapter, reportingInterval, reportingVoters...),
	}
}
