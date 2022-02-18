package thing

import (
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"
	"github.com/futurehomeno/cliffhanger/adapter/service/meterelec"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
)

// CarChargerConfig represents a thing configuration.
type CarChargerConfig struct {
	InclusionReport   *fimptype.ThingInclusionReport
	ChargepointConfig *chargepoint.Config
	MeterElecConfig   *meterelec.Config // Optional
}

// NewCarCharger creates a thing that satisfies expectations for a car charger.
// Specification and implementation for electricity meter is optional.
func NewCarCharger(
	mqtt *fimpgo.MqttTransport,
	cfg *CarChargerConfig,
) adapter.Thing {
	services := []adapter.Service{
		chargepoint.NewService(mqtt, cfg.ChargepointConfig),
	}

	if cfg.MeterElecConfig != nil {
		services = append(services, meterelec.NewService(mqtt, cfg.MeterElecConfig))
	}

	return adapter.NewThing(cfg.InclusionReport, services...)
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
