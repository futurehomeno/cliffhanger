package thing

import (
	"time"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/meterelec"
	"github.com/futurehomeno/cliffhanger/adapter/service/numericsensor"
	"github.com/futurehomeno/cliffhanger/adapter/service/waterheater"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
)

// BoilerConfig represents a thing configuration.
type BoilerConfig struct {
	ThingConfig         *adapter.ThingConfig
	WaterHeaterConfig   *waterheater.Config
	SensorWatTempConfig *numericsensor.Config // Optional
	MeterElecConfig     *meterelec.Config     // Optional
}

// NewBoiler creates a thing that satisfies expectations for a boiler.
// Specification and implementations for temperature sensor and electricity meter are optional.
func NewBoiler(
	publisher adapter.Publisher,
	ts adapter.ThingState,
	cfg *BoilerConfig,
) adapter.Thing {
	services := []adapter.Service{
		waterheater.NewService(publisher, cfg.WaterHeaterConfig),
	}

	if cfg.SensorWatTempConfig != nil && cfg.SensorWatTempConfig.Specification.Name == numericsensor.SensorWatTemp {
		services = append(services, numericsensor.NewService(publisher, cfg.SensorWatTempConfig))
	}

	if cfg.MeterElecConfig != nil {
		services = append(services, meterelec.NewService(publisher, cfg.MeterElecConfig))
	}

	return adapter.NewThing(publisher, ts, cfg.ThingConfig, services...)
}

// RouteBoiler creates routing required to satisfy expectations for a boiler.
func RouteBoiler(adapter adapter.Adapter) []*router.Routing {
	return router.Combine(
		waterheater.RouteService(adapter),
		numericsensor.RouteService(adapter),
		meterelec.RouteService(adapter),
	)
}

// TaskBoiler creates background tasks specific for a boiler.
func TaskBoiler(
	adapter adapter.Adapter,
	reportingInterval time.Duration,
	reportingVoters ...task.Voter,
) []*task.Task {
	return []*task.Task{
		waterheater.TaskReporting(adapter, reportingInterval, reportingVoters...),
		numericsensor.TaskReporting(adapter, reportingInterval, reportingVoters...),
		meterelec.TaskReporting(adapter, reportingInterval, reportingVoters...),
	}
}
