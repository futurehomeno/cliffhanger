package thing

import (
	"time"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/numericsensor"
	"github.com/futurehomeno/cliffhanger/adapter/service/presence"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
)

// SensorConfig represents a thing configuration.
type SensorConfig struct {
	ThingConfig          *adapter.ThingConfig
	NumericSensorConfigs []*numericsensor.Config
	PresenceConfig       *presence.Config
}

// NewSensor creates a thing that satisfies expectations for the battery.
func NewSensor(
	publisher adapter.Publisher,
	ts adapter.ThingState,
	cfg *SensorConfig,
) adapter.Thing {
	var services []adapter.Service

	if cfg.PresenceConfig != nil {
		services = append(services, presence.NewService(publisher, cfg.PresenceConfig))
	}

	for _, numericSensorConfig := range cfg.NumericSensorConfigs {
		services = append(services, numericsensor.NewService(publisher, numericSensorConfig))
	}

	return adapter.NewThing(publisher, ts, cfg.ThingConfig, services...)
}

// RouteSensor creates routing required to satisfy expectations for the battery.
func RouteSensor(adapter adapter.Adapter) []*router.Routing {
	return router.Combine(
		presence.RouteService(adapter),
		numericsensor.RouteService(adapter),
	)
}

// TaskSensor creates background tasks specific for the battery.
func TaskSensor(
	adapter adapter.Adapter,
	reportingInterval time.Duration,
	reportingVoters ...task.Voter,
) []*task.Task {
	return task.Combine(
		presence.TaskReporting(adapter, reportingInterval, reportingVoters...),
		numericsensor.TaskReporting(adapter, reportingInterval, reportingVoters...),
	)
}
