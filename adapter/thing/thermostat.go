package thing

import (
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/meterelec"
	"github.com/futurehomeno/cliffhanger/adapter/service/numericsensor"
	"github.com/futurehomeno/cliffhanger/adapter/service/thermostat"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
)

// ThermostatConfig represents a thing configuration.
type ThermostatConfig struct {
	InclusionReport  *fimptype.ThingInclusionReport
	ThermostatConfig *thermostat.Config
	SensorTempConfig *numericsensor.Config // Optional
	MeterElecConfig  *meterelec.Config     // Optional
}

// NewThermostat creates a thing that satisfies expectations for a thermostat controller.
// Specification and implementations for temperature sensor and electricity meter are optional.
func NewThermostat(
	mqtt *fimpgo.MqttTransport,
	cfg *ThermostatConfig,
) adapter.Thing {
	services := []adapter.Service{
		thermostat.NewService(mqtt, cfg.ThermostatConfig),
	}

	if cfg.SensorTempConfig != nil && cfg.SensorTempConfig.Specification.Name == numericsensor.SensorTemp {
		services = append(services, numericsensor.NewService(mqtt, cfg.SensorTempConfig))
	}

	if cfg.MeterElecConfig != nil {
		services = append(services, meterelec.NewService(mqtt, cfg.MeterElecConfig))
	}

	return adapter.NewThing(cfg.InclusionReport, services...)
}

// RouteThermostat creates routing required to satisfy expectations for a thermostat controller.
func RouteThermostat(adapter adapter.Adapter) []*router.Routing {
	return router.Combine(
		thermostat.RouteService(adapter),
		numericsensor.RouteService(adapter),
		meterelec.RouteService(adapter),
	)
}

// TaskThermostat creates background tasks specific for a thermostat controller.
func TaskThermostat(
	adapter adapter.Adapter,
	reportingInterval time.Duration,
	reportingVoters ...task.Voter,
) []*task.Task {
	return []*task.Task{
		thermostat.TaskReporting(adapter, reportingInterval, reportingVoters...),
		numericsensor.TaskReporting(adapter, reportingInterval, reportingVoters...),
		meterelec.TaskReporting(adapter, reportingInterval, reportingVoters...),
	}
}
