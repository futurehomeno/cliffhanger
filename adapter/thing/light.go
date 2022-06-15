package thing

import (
	"time"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/outlvlswitch"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
)

// LightConfig represents a thing configuration.
type LightConfig struct {
	InclusionReport    *fimptype.ThingInclusionReport
	OutLvlSwitchConfig *outlvlswitch.Config
}

// NewLight creates a thing that satisfies expectations for a light.
// Specification and implementation for electricity meter is optional.
func NewLight(
	mqtt *fimpgo.MqttTransport,
	cfg *LightConfig,
) adapter.Thing {
	services := []adapter.Service{
		outlvlswitch.NewService(mqtt, cfg.OutLvlSwitchConfig),
	}

	return adapter.NewThing(cfg.InclusionReport, services...)
}

// RouteLight creates routing required to satisfy expectations for a light.
func RouteLight(adapter adapter.Adapter) []*router.Routing {
	return router.Combine(
		outlvlswitch.RouteService(adapter),
	)
}

// TaskLight creates background tasks specific for a light.
func TaskLight(
	adapter adapter.Adapter,
	reportingInterval time.Duration,
	reportingVoters ...task.Voter,
) []*task.Task {
	return []*task.Task{
		outlvlswitch.TaskReporting(adapter, reportingInterval, reportingVoters...),
	}
}
