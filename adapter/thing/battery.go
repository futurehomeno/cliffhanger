package thing

import (
	"time"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/battery"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
)

// BatteryConfig represents a thing configuration.
type BatteryConfig struct {
	ThingConfig   *adapter.ThingConfig
	BatteryConfig *battery.Config
}

// NewBattery creates a thing that satisfies expectations for the battery.
func NewBattery(
	publisher adapter.Publisher,
	ts adapter.ThingState,
	cfg *BatteryConfig,
) adapter.Thing {
	Battery := battery.NewService(publisher, cfg.BatteryConfig)

	return adapter.NewThing(publisher, ts, cfg.ThingConfig, Battery)
}

// RouteBattery creates routing required to satisfy expectations for the battery.
func RouteBattery(adapter adapter.Adapter) []*router.Routing {
	return battery.RouteService(adapter)
}

// TaskBattery creates background tasks specific for the battery.
func TaskBattery(
	adapter adapter.Adapter,
	reportingInterval time.Duration,
	reportingVoters ...task.Voter,
) []*task.Task {
	return []*task.Task{
		battery.TaskReporting(adapter, reportingInterval, reportingVoters...),
	}
}
